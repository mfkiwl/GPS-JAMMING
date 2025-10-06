// sdracq.go : SDR acquisition functions (Go version)
// Translated from sdracq.c
package main

import (
	"gonum.org/v1/gonum/dsp/fourier"
	"math"
	"time"
	"fmt"
	// "os"
)

// SDR acquisition function
func SdrAcquisition(ini *SdrIni, sdrstat *SdrStat, sdr *SdrCh, power []float64) uint64 {
	// XCode jest już proper generated w sdrinit.go - nie regeneruj
	fmt.Printf("SdrAcquisition: Using pre-generated XCode for PRN %d, first values: %v\n", sdr.Prn, sdr.XCode[0:4])
	
	// Wyzeruj bufor power przed acquisition (akumulacja wymaga zerowych wartości początkowych)
	for i := range power {
		power[i] = 0.0
	}
	
	buffloc := (uint64(sdrstat.FendBuffSize) * uint64(sdrstat.BuffCnt)) - uint64((sdr.Acq.Intg+1)*sdr.NSamp)
	var i int
	SDRPRINTF("SdrAcquisition: BuffCnt=%d, buffloc=%d, NSamp=%d, Nfft=%d, Dtype=%d\n", sdrstat.BuffCnt, buffloc, sdr.NSamp, sdr.Acq.Nfft, sdr.Dtype)
	for i = 0; i < sdr.Acq.Intg; i++ {
		data := make([]byte, 2*sdr.NSamp*sdr.Dtype)
		RcvGetBuff(ini, buffloc, 2*sdr.NSamp, sdr.Ftype, sdr.Dtype, sdrstat, data)
		buffloc += uint64(sdr.NSamp)
		SDRPRINTF("SdrAcquisition: loop %d, buffloc=%d\n", i, buffloc)
		Pcorrelator(data, sdr.Dtype, sdr.Ti, sdr.NSamp, sdr.Acq.Freq, sdr.Acq.Nfreq, sdr.CRate, sdr.Acq.Nfft, sdr.XCode, power)
		if CheckAcquisition(power, sdr) {
			sdr.FlagAcq = ON
			SDRPRINTF("SdrAcquisition: PRN %d acquired!\n", sdr.Prn)
			break
		}
	}
	if sdr.FlagAcq == ON {
		buffloc += -(uint64(i+1) * uint64(sdr.NSamp)) + uint64(sdr.Acq.AcqCodeI)
		sdr.Trk.CarrFreq = sdr.Acq.AcqFreq
		sdr.Trk.CodeFreq = sdr.CRate
	} else {
		SDRPRINTF("SdrAcquisition: PRN %d not acquired.\n", sdr.Prn)
		Sleepms(ACQSLEEP)
	}
	return buffloc
}

// CheckAcquisition: sprawdza wynik akwizycji, oblicza C/N0 i peak ratio
func CheckAcquisition(P []float64, sdr *SdrCh) bool {
       // Znajdź maksimum w całej tablicy
       maxP, maxi := MaxVD(P, sdr.NSamp*sdr.Acq.Nfreq, -1, -1)
       codei, freqi := Ind2Sub(maxi, sdr.NSamp, sdr.Acq.Nfreq)
       
       // C/N0 calculation - średnia z wykluczeniem obszaru wokół piku
       exinds := codei - 2*sdr.NSampChip
       if exinds < 0 {
	       exinds += sdr.NSamp
       }
       exinde := codei + 2*sdr.NSampChip
       if exinde >= sdr.NSamp {
	       exinde -= sdr.NSamp
       }
       meanP := MeanVD(P[freqi*sdr.NSamp:], sdr.NSamp, exinds, exinde)
       sdr.Acq.Cn0 = 10 * math.Log10(maxP/meanP/sdr.CTime)
       
       // Peak ratio - drugi największy pik w tym samym paśmie częstotliwości z wykluczeniem obszaru głównego piku
       maxP2, _ := MaxVD(P[freqi*sdr.NSamp:], sdr.NSamp, exinds, exinde)
       sdr.Acq.PeakR = maxP / maxP2
       
       sdr.Acq.AcqCodeI = codei
       sdr.Acq.FreqI = freqi
       sdr.Acq.AcqFreq = sdr.Acq.Freq[freqi]
	//    if sdr.Acq.PeakR > 3 {
	// 	   fmt.Printf("CheckAcquisition WARNING: High peak ratio %.2f for PRN %d\n", sdr.Acq.PeakR, sdr.Prn)
	// 	   os.Exit(1)
	//    }

       fmt.Printf("CheckAcquisition: maxP=%.2e, maxP2=%.2e, peakR=%.3f, Cn0=%.2f, threshold=%.3f, codei=%d, freqi=%d\n", 
                  maxP, maxP2, sdr.Acq.PeakR, sdr.Acq.Cn0, ACQTH, codei, freqi)
       return sdr.Acq.PeakR > ACQTH
}

// Pcorrelator: FFT-based parallel correlator
func Pcorrelator(data []byte, dtype int, ti float64, nsamp int, freq []float64, nfreq int, crate float64, nfft int, xcode []complex64, power []float64) {
       CSCALE := 1.0 / 32.0
       
       // Zero-padded data buffer (zgodnie z C: memset + memcpy)
       dataR := make([]byte, nfft*dtype)
       copy(dataR, data[:2*nsamp*dtype])  // Zero-padding: reszta pozostaje zerowa
       
       for i := 0; i < nfreq; i++ {
	       II := make([]int16, nfft)
	       QQ := make([]int16, nfft)
	       MixCarr(dataR, dtype, ti, nfft, freq[i], 0.0, II, QQ)
	       
	       // Konwersja do complex z prawidłowym skalowaniem zgodnie z C: CSCALE/m
	       cpx := make([]complex128, nfft)
	       scale := CSCALE / float64(nfft)  // Exactly like C: CSCALE/m
	       for j := 0; j < nfft; j++ {
		       cpx[j] = complex(float64(II[j])*scale, float64(QQ[j])*scale)
	       }
	       
	       // FFT danych (jak w cpxconv: cpxfft(plan,cpxa,m))
	       fft := fourier.NewCmplxFFT(nfft)
	       dataFFT := fft.Coefficients(nil, cpx)
	       
	       // Powrót do DOKŁADNEGO C multiplication
	       conv := make([]complex128, nfft)
	       for j := 0; j < nfft; j++ {
		       // Exact C logic: real=-p[0]*q[0]-p[1]*q[1], p[1]=p[0]*q[1]-p[1]*q[0]
		       p0 := real(dataFFT[j])
		       p1 := imag(dataFFT[j])  
		       q0 := float64(real(xcode[j]))
		       q1 := float64(imag(xcode[j]))
		       
		       real_part := -p0*q0 - p1*q1
		       imag_part := p0*q1 - p1*q0
		       conv[j] = complex(real_part, imag_part)
	       }
	       
	       // KLUCZOWE: proper IFFT implementation
	       // W FFTW: FFTW_BACKWARD daje unscaled inverse transform  
	       // Gonum nie ma IFFT, więc symulujemy przez: conj → FFT → conj
	       
	       // 1. Conjugate
	       for j := 0; j < nfft; j++ {
		       conv[j] = complex(real(conv[j]), -imag(conv[j]))
	       }
	       
	       // 2. Forward FFT  
	       ifft := fft.Coefficients(nil, conv)
	       
	       // 3. Conjugate result (symuluje IFFT)
	       for j := 0; j < nfft; j++ {
		       ifft[j] = complex(real(ifft[j]), -imag(ifft[j]))
	       }
	       
	       // Obliczanie mocy jak w C
	       m2 := float64(nfft * nfft)
	       for j := 0; j < nsamp; j++ {
		       power[i*nsamp+j] += (real(ifft[j])*real(ifft[j]) + imag(ifft[j])*imag(ifft[j])) / m2
	       }
       }
}

// Generate FFT of resampled C/A code for acquisition (właściwe parametry)
func GenerateXCodeProperly(caCode []int16, clen int, ci float64, nsamp int, nfft int) []complex64 {
       // Zero-padded array do nfft (zgodnie z C)
       rcode := make([]int16, nfft)
       
       // Właściwy resampling (zgodnie z C: rescode call)
       ResCode(caCode, clen, 0.0, 0, ci, nsamp, rcode)
       
       // Convert to complex and apply FFT
       rcodeComplex := make([]complex128, nfft)
       for i := 0; i < nfft; i++ {
	       if i < nsamp {
		       rcodeComplex[i] = complex(float64(rcode[i]), 0)
	       } else {
		       rcodeComplex[i] = 0  // zero padding od nsamp do nfft
	       }
       }
       
       // FFT (zgodnie z C: cpxfft)
       fft := fourier.NewCmplxFFT(nfft)
       xcodeC := fft.Coefficients(nil, rcodeComplex)
       
       // Convert to complex64
       xcode := make([]complex64, nfft)
       for i := 0; i < nfft; i++ {
	       xcode[i] = complex(float32(real(xcodeC[i])), float32(imag(xcodeC[i])))
       }
       
       return xcode
}

// Generate FFT of resampled C/A code for acquisition
func GenerateXCode(caCode []int16, nfft int) []complex64 {
       // Zero-padded array do nfft (zgodnie z C)
       rcode := make([]int16, nfft)
       
       // Parametry resampingu (zgodnie z C: rescode call)
       codeLen := len(caCode)
       
       // Proste resample do nsamp próbek (jak w C: rescode)
       // W C: rescode(sdr->code,sdr->clen,0,0,sdr->ci,sdr->nsamp,rcode)
       // Używamy ci ≈ codeLen/nsamp dla uproszczenia
       ci := float64(codeLen) / 2048.0  // Zakładamy nsamp=2048 dla GPS
       ResCode(caCode, codeLen, 0.0, 0, ci, 2048, rcode)
       
       // Konwersja do complex (zgodnie z C: cpxcpx)
       rcodeComplex := make([]complex128, nfft)
       for i := 0; i < nfft; i++ {
	       if i < len(rcode) {
		       rcodeComplex[i] = complex(float64(rcode[i]), 0)
	       } else {
		       rcodeComplex[i] = 0  // zero padding do nfft
	       }
       }
       
       // Pełne kompleksowe FFT (zgodnie z C: cpxfft)
       fft := fourier.NewCmplxFFT(nfft)
       xcodeC := fft.Coefficients(nil, rcodeComplex)
       
       // Konwersja do complex64
       xcode := make([]complex64, nfft)
       for i := 0; i < nfft; i++ {
	       xcode[i] = complex(float32(real(xcodeC[i])), float32(imag(xcodeC[i])))
       }
       
       return xcode
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
