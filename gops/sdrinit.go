// sdrinit.go : SDR initialize/cleanup functions (Go version)
// Translated from sdrinit.c
package main

import (
	"os"
	"errors"
	"gonum.org/v1/gonum/dsp/fourier"
)

// Load initial value
func LoadInit(ini *SdrIni, filename string) error {
	ini.Fend = FEND_FRTLSDR
	ini.FCf[0] = 1575.42e6
	ini.FSf[0] = 2.048e6
	ini.FIf[0] = 0.0
	ini.Dtype[0] = 2
	ini.FCf[1] = 0.0
	ini.FSf[1] = 0.0
	ini.FIf[1] = 0.0
	ini.Dtype[1] = 0
	ini.File = filename
	ini.UseIf = ON
	ini.RtlsdrPpmErr = 0
	ini.TrkCorrN = 4
	ini.TrkCorrD = 1
	ini.TrkCorrP = 1
	ini.TrkDllB[0] = 5.0
	ini.TrkPllB[0] = 30.0
	ini.TrkFllB[0] = 200.0
	ini.TrkDllB[1] = 2.0
	ini.TrkPllB[1] = 20.0
	ini.TrkFllB[1] = 50.0
	ini.Nch = 32
	ini.Prn = make([]int, 32)
	ini.Sys = make([]int, 32)
	ini.Ctype = make([]int, 32)
	ini.Ftype = make([]int, 32)
	for i := 0; i < 32; i++ {
		ini.Prn[i] = i + 1
		ini.Sys[i] = 1
		ini.Ctype[i] = 1
		ini.Ftype[i] = 1
	}
	ini.PltAcq = 0
	ini.PltTrk = 0
	ini.OutMs = 200
	ini.Sbas = 0
	ini.PltSpec = 0
	ini.Xu0V = [3]int{693570, -5193930, 3624632}
	ini.EkfFilterOn = 0
	ini.NchL1 = 0
	for i := 0; i < ini.Nch; i++ {
		if ini.Ctype[i] == 1 {
			ini.NchL1++
		}
	}
	return nil
}

// Check initial value
func ChkInitValue(ini *SdrIni) error {
	if ini.FSf[0] <= 0 || ini.FSf[0] > 100e6 || ini.FIf[0] < 0 || ini.FIf[0] > 100e6 {
		return errors.New("error: wrong freq. input")
	}
	if ini.Fend == FEND_FILE || ini.Fend == FEND_FRTLSDR {
		if ini.UseIf == ON {
			if _, err := os.Stat(ini.File); err != nil {
				return errors.New("error: file doesn't exist: " + ini.File)
			}
		}
		if ini.UseIf == OFF {
			return errors.New("error: file is not selected")
		}
	}
	return nil
}

// Initialize acquisition struct
func InitAcqStruct(sys, ctype, prn int, acq *SdrAcq) {
	if ctype == CTYPE_L1CA {
		acq.Intg = ACQINTG_L1CA
	}
	acq.HBand = ACQHBAND
	acq.Step = ACQSTEP
	acq.Nfreq = 2*(ACQHBAND/ACQSTEP) + 1
}

// Initialize tracking parameter struct
func InitTrkPrmStruct(trk *SdrTrk, ini *SdrIni) {
	trk.Prm1.DllB = ini.TrkDllB[0]
	trk.Prm1.PllB = ini.TrkPllB[0]
	trk.Prm1.FllB = ini.TrkFllB[0]
	trk.Prm2.DllB = ini.TrkDllB[1]
	trk.Prm2.PllB = ini.TrkPllB[1]
	trk.Prm2.FllB = ini.TrkFllB[1]
	trk.Prm1.DllW2 = (trk.Prm1.DllB/0.53)*(trk.Prm1.DllB/0.53)
	trk.Prm1.DllAw = 1.414*(trk.Prm1.DllB/0.53)
	trk.Prm1.PllW2 = (trk.Prm1.PllB/0.53)*(trk.Prm1.PllB/0.53)
	trk.Prm1.PllAw = 1.414*(trk.Prm1.PllB/0.53)
	trk.Prm1.FllW = trk.Prm1.FllB/0.25
	trk.Prm2.DllW2 = (trk.Prm2.DllB/0.53)*(trk.Prm2.DllB/0.53)
	trk.Prm2.DllAw = 1.414*(trk.Prm2.DllB/0.53)
	trk.Prm2.PllW2 = (trk.Prm2.PllB/0.53)*(trk.Prm2.PllB/0.53)
	trk.Prm2.PllAw = 1.414*(trk.Prm2.PllB/0.53)
	trk.Prm2.FllW = trk.Prm2.FllB/0.25
}

// Initialize tracking struct
func InitTrkStruct(sat, ctype int, ctime float64, trk *SdrTrk, ini *SdrIni) error {
	InitTrkPrmStruct(trk, ini)
	trkcorrn := ini.TrkCorrN
	trkcorrd := ini.TrkCorrD
	trkcorrp := ini.TrkCorrP
	trk.CorrN = trkcorrn
	trk.CorrP = make([]int, trkcorrn)
	for i := 0; i < trkcorrn; i++ {
		trk.CorrP[i] = trkcorrd * (i + 1)
		if trk.CorrP[i] == trkcorrp {
			trk.Ne = 2*(i+1) - 1
			trk.Nl = 2*(i+1)
		}
	}
	trk.CorrX = make([]float64, 2*trkcorrn+1)
	for i := 1; i <= trkcorrn; i++ {
		trk.CorrX[2*i-1] = -float64(trkcorrd*i)
		trk.CorrX[2*i] = float64(trkcorrd*i)
	}
	trk.II = make([]float64, 1+2*trkcorrn)
	trk.QQ = make([]float64, 1+2*trkcorrn)
	trk.OldI = make([]float64, 1+2*trkcorrn)
	trk.OldQ = make([]float64, 1+2*trkcorrn)
	trk.SumI = make([]float64, 1+2*trkcorrn)
	trk.SumQ = make([]float64, 1+2*trkcorrn)
	trk.OldSumI = make([]float64, 1+2*trkcorrn)
	trk.OldSumQ = make([]float64, 1+2*trkcorrn)
	if ctype == CTYPE_L1CA {
		trk.Loop = LOOP_L1CA
	}
	if ctype == CTYPE_L1SBAS {
		trk.Loop = LOOP_SBAS
	}
	trk.LoopMs = int(float64(trk.Loop) * ctime * 1000)
	return nil
}

// Initialize navigation struct
func InitNavStruct(sys, ctype, prn int, nav *SdrNav) error {
	preL1CA := []int{1, -1, -1, -1, 1, -1, 1, 1}
	preSBAS := []int{1, -1, 1, -1, 1, 1, -1, -1, -1, 1, 1, -1, -1, 1, -1, 1, -1, -1, 1, 1, 1, -1, -1, 1}
	nav.Ctype = ctype
	// GPS/QZS L1CA
	if ctype == CTYPE_L1CA {
		nav.Rate = NAVRATE_L1CA
		nav.Flen = NAVFLEN_L1CA
		nav.AddFlen = NAVADDFLEN_L1CA
		nav.PreLen = NAVPRELEN_L1CA
		nav.Update = nav.Flen * nav.Rate
		copy(nav.PreBits[:], preL1CA)
		nav.OCode = make([]int16, nav.Rate)
		for i := range nav.OCode {
			nav.OCode[i] = 1
		}
	}
	// SBAS/QZS L1SAIF
	if ctype == CTYPE_L1SBAS {
		nav.Rate = NAVRATE_SBAS
		nav.Flen = NAVFLEN_SBAS
		nav.AddFlen = NAVADDFLEN_SBAS
		nav.PreLen = NAVPRELEN_SBAS
		nav.Update = nav.Flen / 3 * nav.Rate
		copy(nav.PreBits[:], preSBAS)
		nav.OCode = make([]int16, nav.Rate)
		for i := range nav.OCode {
			nav.OCode[i] = 1
		}
	}
	nav.BitSync = make([]int, nav.Rate)
	nav.FBits = make([]int, nav.Flen+nav.AddFlen)
	nav.FBitsDec = make([]int, nav.Flen+nav.AddFlen)
	return nil
}

// Free sdr channel struct
func FreeSdrCh(sdr *SdrCh) {
	sdr.Code = nil
	sdr.XCode = nil
	sdr.Nav.FBits = nil
	sdr.Nav.FBitsDec = nil
	sdr.Nav.BitSync = nil
	sdr.Trk.II = nil
	sdr.Trk.QQ = nil
	sdr.Trk.OldI = nil
	sdr.Trk.OldQ = nil
	sdr.Trk.SumI = nil
	sdr.Trk.SumQ = nil
	sdr.Trk.OldSumI = nil
	sdr.Trk.OldSumQ = nil
	sdr.Trk.CorrP = nil
	sdr.Acq.Freq = nil
	sdr.Nav.OCode = nil
}

// Initialize sdr channel struct
func InitSdrCh(chno, sys, prn, ctype, dtype, ftype, f_gain, f_bias, f_clock int, f_cf, f_sf, f_if float64, sdr *SdrCh, ini *SdrIni) error {
	sdr.No = chno
	sdr.Sys = sys
	sdr.Prn = prn
	sdr.Ctype = ctype
	sdr.Dtype = dtype
	sdr.Ftype = ftype
	sdr.FSf = f_sf
	sdr.FGain = f_gain
	sdr.FBias = f_bias
	sdr.FClock = f_clock
	sdr.FIf = f_if
	sdr.Ti = 1 / f_sf
	// Code generation (tu: wywołanie generatora kodu)
	code, clen, crate := GencodeL1CA(prn)
	sdr.Code = code
	sdr.Clen = clen
	sdr.CRate = crate
	sdr.Ci = sdr.Ti * sdr.CRate
	sdr.CTime = float64(clen) / crate
	sdr.NSamp = int(f_sf * sdr.CTime)
	sdr.NSampChip = int(float64(sdr.NSamp) / float64(clen))
	// Carrier frequency
	if ini.Fend == FEND_FRTLSDR {
		sdr.FOffset = f_cf * float64(ini.RtlsdrPpmErr) * 1e-6
		sdr.FCf = f_cf
	} else {
		sdr.FCf = f_cf
		sdr.FOffset = 0.0
	}
	// Acquisition struct
	InitAcqStruct(sys, ctype, prn, &sdr.Acq)
	sdr.Acq.Nfft = 2 * sdr.NSamp
	sdr.Acq.Freq = make([]float64, sdr.Acq.Nfreq)
	for i := 0; i < sdr.Acq.Nfreq; i++ {
		sdr.Acq.Freq[i] = sdr.FIf + float64(i-(sdr.Acq.Nfreq-1)/2)*sdr.Acq.Step + sdr.FOffset
	}
	// Tracking struct
	if err := InitTrkStruct(prn, ctype, sdr.CTime, &sdr.Trk, ini); err != nil {
		return err
	}
	// Navigation struct
	if err := InitNavStruct(sys, ctype, prn, &sdr.Nav); err != nil {
		return err
	}
	// XCode (FFT for acquisition)
	sdr.XCode = make([]complex64, sdr.Acq.Nfft)
	// Resampling code
	rcode := make([]int16, sdr.Acq.Nfft)
	for i := range rcode {
		rcode[i] = 0
	}
	ResCode(sdr.Code, sdr.Clen, 0, 0, sdr.Ci, sdr.NSamp, rcode)
	for i := range sdr.XCode {
		sdr.XCode[i] = complex(float32(rcode[i]), 0)
	}
	// FFT
	// cpxfft equivalent in Go using gonum.org/v1/gonum/fourier
	fft := fourier.NewCmplxFFT(len(sdr.XCode))
	xcode128 := make([]complex128, len(sdr.XCode))
	for i := range sdr.XCode {
		xcode128[i] = complex(float64(real(sdr.XCode[i])), float64(imag(sdr.XCode[i])))
	}
	result := fft.Coefficients(nil, xcode128)
	for i := range sdr.XCode {
		sdr.XCode[i] = complex64(result[i])
	}
	return nil
}
