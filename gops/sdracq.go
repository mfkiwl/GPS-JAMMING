// sdracq.go : SDR acquisition functions (Go version)
// Translated from sdracq.c
package main

import (
	"gonum.org/v1/gonum/dsp/fourier"
	"math"
	"time"
)

// SDR acquisition function
func SdrAcquisition(ini *SdrIni, sdrstat *SdrStat, sdr *SdrCh, power []float64) uint64 {
	// Assume ini and sdrstat are available in context or passed as arguments
	// You may need to update the function signature to include ini and sdrstat
	// You must pass sdrstat as argument or get from context
	buffloc := (uint64(sdrstat.FendBuffSize) * uint64(sdrstat.BuffCnt)) - uint64((sdr.Acq.Intg+1)*sdr.NSamp)
	var i int
	for i = 0; i < sdr.Acq.Intg; i++ {
		data := make([]byte, 2*sdr.NSamp*sdr.Dtype)
		RcvGetBuff(ini, buffloc, 2*sdr.NSamp, sdr.Ftype, sdr.Dtype, sdrstat, data)
		buffloc += uint64(sdr.NSamp)
		Pcorrelator(data, sdr.Dtype, sdr.Ti, sdr.NSamp, sdr.Acq.Freq, sdr.Acq.Nfreq, sdr.CRate, sdr.Acq.Nfft, sdr.XCode, power)
		if CheckAcquisition(power, sdr) {
			sdr.FlagAcq = ON
			break
		}
	}
	if sdr.FlagAcq == ON {
		buffloc += -(uint64(i+1) * uint64(sdr.NSamp)) + uint64(sdr.Acq.AcqCodeI)
		sdr.Trk.CarrFreq = sdr.Acq.AcqFreq
		sdr.Trk.CodeFreq = sdr.CRate
	} else {
		Sleepms(ACQSLEEP)
	}
	return buffloc
}

// CheckAcquisition: sprawdza wynik akwizycji, oblicza C/N0 i peak ratio
func CheckAcquisition(P []float64, sdr *SdrCh) bool {
	maxP, maxi := MaxVD(P, -1, -1)
	codei, freqi := Ind2Sub(maxi, sdr.NSamp, sdr.Acq.Nfreq)
	// C/N0 calculation
	exinds := codei - 2*sdr.NSampChip
	if exinds < 0 {
		exinds += sdr.NSamp
	}
	exinde := codei + 2*sdr.NSampChip
	if exinde >= sdr.NSamp {
		exinde -= sdr.NSamp
	}
	meanP := MeanVD(P[freqi*sdr.NSamp:], exinds, exinde)
	sdr.Acq.Cn0 = 10 * math.Log10(maxP/meanP/sdr.CTime)
	// Peak ratio
	maxP2, _ := MaxVD(P[freqi*sdr.NSamp:], exinds, exinde)
	sdr.Acq.PeakR = maxP / maxP2
	sdr.Acq.AcqCodeI = codei
	sdr.Acq.FreqI = freqi
	sdr.Acq.AcqFreq = sdr.Acq.Freq[freqi]
	return sdr.Acq.PeakR > ACQTH
}

// Pcorrelator: FFT-based parallel correlator
func Pcorrelator(data []byte, dtype int, ti float64, nsamp int, freq []float64, nfreq int, crate float64, nfft int, xcode []complex64, power []float64) {
	// Zakładamy, że xcode to kod w dziedzinie częstotliwości
	for i := 0; i < nfreq; i++ {
		II := make([]int16, nsamp)
		QQ := make([]int16, nsamp)
		MixCarr(data, dtype, ti, nsamp, freq[i], 0.0, II, QQ)
		cpx := make([]complex128, nsamp)
		for j := 0; j < nsamp; j++ {
			cpx[j] = complex(float64(II[j]), float64(QQ[j])) / complex(float64(nsamp), 0)
		}
	fft := fourier.NewCmplxFFT(nsamp)
	dataFFT := fft.Coefficients(nil, cpx)
		xcode128 := make([]complex128, nsamp)
		for j := 0; j < nsamp; j++ {
			xcode128[j] = complex(float64(real(xcode[j])), float64(imag(xcode[j])))
		}
		conv := make([]complex128, nsamp)
		for j := 0; j < nsamp; j++ {
			conv[j] = dataFFT[j] * complex(real(xcode128[j]), -imag(xcode128[j]))
		}
		ifft := fft.Sequence(nil, conv)
		for j := 0; j < nsamp; j++ {
			power[i*nsamp+j] = real(ifft[j])*real(ifft[j]) + imag(ifft[j])*imag(ifft[j])
		}
	}
}

// Pomocnicza konwersja []complex64 -> []float64
func toFloat64Slice(in []complex64) []float64 {
	out := make([]float64, len(in))
	for i, v := range in {
		out[i] = float64(real(v))
	}
	return out
}

// Sleepms: uśpienie w milisekundach
func Sleepms(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
	}
