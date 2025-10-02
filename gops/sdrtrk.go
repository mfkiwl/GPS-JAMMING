// sdrtrk.go : SDR tracking functions (Go version)
// Translated from sdrtrk.c
package main

import (
	"math"
)

// Tracking function
func sdrtracking(sdr *SdrCh, buffloc, cnt uint64) uint64 {
	// Memory allocation for data buffer
	data := make([]byte, (sdr.NSamp+100)*sdr.Dtype)
	var bufflocnow uint64
	// Current buffer location
	hreadmtx.Lock()
	bufflocnow = uint64(sdrstat.FendBuffSize)*sdrstat.BuffCnt - uint64(sdr.NSamp)
	hreadmtx.Unlock()

	if bufflocnow > buffloc {
	sdr.CurrNSamp = int((float64(sdr.Clen)-sdr.Trk.RemCode)/(sdr.Trk.CodeFreq/sdr.FSf))
		rcvgetbuff(&sdrini, buffloc, sdr.CurrNSamp, sdr.Ftype, sdr.Dtype, data)
		copy(sdr.Trk.OldI, sdr.Trk.II)
		copy(sdr.Trk.OldQ, sdr.Trk.QQ)
		sdr.Trk.OldRemCode = sdr.Trk.RemCode
		sdr.Trk.OldRemCarr = sdr.Trk.RemCarr
		// Correlation
		correlator(data, sdr.Dtype, sdr.Ti, sdr.CurrNSamp, sdr.Trk.CarrFreq,
			sdr.Trk.OldRemCarr, sdr.Trk.CodeFreq, sdr.Trk.OldRemCode,
			sdr.Trk.CorrP, sdr.Trk.CorrN, sdr.Trk.QQ, sdr.Trk.II,
			&sdr.Trk.RemCode, &sdr.Trk.RemCarr, sdr.Code, sdr.Clen)
		// Navigation
		sdrnavigation(sdr, buffloc, cnt)
	sdr.FlagTrk = ON
	} else {
		sleepms(1)
	}
	return bufflocnow
}

// Cumulative sum of correlation output
func cumsumcorr(trk *SdrTrk, polarity int) {
	// Direct translation of cumulative sum of correlation output
	for i := 0; i < 1+2*trk.CorrN; i++ {
		trk.II[i] *= float64(polarity)
		trk.QQ[i] *= float64(polarity)
		trk.OldSumI[i] += trk.OldI[i]
		trk.OldSumQ[i] += trk.OldQ[i]
		trk.SumI[i] += trk.II[i]
		trk.SumQ[i] += trk.QQ[i]
	}
}

func clearcumsumcorr(trk *SdrTrk) {
	// Direct translation of clear cumulative sum
	for i := 0; i < 1+2*trk.CorrN; i++ {
		trk.OldSumI[i] = 0
		trk.OldSumQ[i] = 0
		trk.SumI[i] = 0
		trk.SumQ[i] = 0
	}
}

// Phase/frequency lock loop (PLL)
func pll(sdr *SdrCh, prm *SdrTrkPrm, dt float64) {
	// Direct translation of PLL logic
	IP := sdr.Trk.SumI[0]
	QP := sdr.Trk.SumQ[0]
	oldIP := sdr.Trk.OldSumI[0]
	oldQP := sdr.Trk.OldSumQ[0]
	var carrErr, freqErr float64
	if IP > 0 {
		carrErr = math.Atan2(QP, IP) / PI
	} else {
		carrErr = math.Atan2(-QP, -IP) / PI
	}
	f1 := math.Atan2(QP, IP)
	f2 := math.Atan2(oldQP, oldIP)
	freqErr = f1 - f2
	if freqErr > PI/2 {
		freqErr = PI - freqErr
	}
	if freqErr < -PI/2 {
		freqErr = -PI - freqErr
	}
	sdr.Trk.CarrNco += prm.PllAw*(carrErr-sdr.Trk.CarrErr) + prm.PllW2*dt*carrErr + prm.FllW*dt*freqErr
	sdr.Trk.CarrFreq = sdr.Acq.AcqFreq + sdr.Trk.CarrNco
	sdr.Trk.CarrErr = carrErr
	sdr.Trk.FreqErr = freqErr
}

// Delay lock loop (DLL)
func dll(sdr *SdrCh, prm *SdrTrkPrm, dt float64) {
	// Direct translation of DLL logic
	IE := sdr.Trk.SumI[sdr.Trk.Ne]
	IL := sdr.Trk.SumI[sdr.Trk.Nl]
	QE := sdr.Trk.SumQ[sdr.Trk.Ne]
	QL := sdr.Trk.SumQ[sdr.Trk.Nl]
	codeErr := (math.Sqrt(IE*IE+QE*QE) - math.Sqrt(IL*IL+QL*QL)) / (math.Sqrt(IE*IE+QE*QE) + math.Sqrt(IL*IL+QL*QL))
	sdr.Trk.CodeNco += prm.DllAw*(codeErr-sdr.Trk.CodeErr) + prm.DllW2*dt*codeErr
	sdr.Trk.CodeFreq = sdr.CRate - sdr.Trk.CodeNco + (sdr.Trk.CarrFreq - sdr.FIf - sdr.FOffset) / (sdr.FCf / sdr.CRate)
	sdr.Trk.CodeErr = codeErr
}

// Set observation data
func setobsdata(sdr *SdrCh, buffloc, cnt uint64, trk *SdrTrk, snrflag int) {
	// Direct translation of set observation data
	// Przesuwanie danych
	copy(trk.Tow[1:], trk.Tow[:OBSINTERPN-1])
	copy(trk.L[1:], trk.L[:OBSINTERPN-1])
	copy(trk.D[1:], trk.D[:OBSINTERPN-1])
	copy(trk.CodeI[1:], trk.CodeI[:OBSINTERPN-1])
	copy(trk.CntOut[1:], trk.CntOut[:OBSINTERPN-1])
	copy(trk.RemCOut[1:], trk.RemCOut[:OBSINTERPN-1])

	trk.Tow[0] = sdr.Nav.FirstSfTow + float64(cnt-sdr.Nav.FirstSfCnt)*sdr.CTime
	trk.CodeI[0] = buffloc
	trk.CntOut[0] = cnt
	trk.RemCOut[0] = trk.OldRemCode * sdr.FSf / trk.CodeFreq

	// Doppler
	trk.D[0] = -(trk.CarrFreq - sdr.FIf - sdr.FOffset)

	// Carrier phase
	if trk.FlagRemCarrAdd == 0 {
		trk.L[0] -= trk.RemCarr / DPI
		trk.FlagRemCarrAdd = ON
	}
	if sdr.Nav.FlagSyncF == 1 && trk.FlagPolarityAdd == 0 {
		if sdr.Nav.Polarity == 1 {
			trk.L[0] += 0.5
		}
		trk.FlagPolarityAdd = ON
	}
	trk.L[0] += trk.D[0] * (float64(trk.LoopMs) * float64(sdr.CurrNSamp) / sdr.FSf)
	trk.ISum += math.Abs(trk.SumI[0])
	if snrflag != 0 {
		copy(trk.S[1:], trk.S[:OBSINTERPN-1])
		copy(trk.CodeISum[1:], trk.CodeISum[:OBSINTERPN-1])
		trk.S[0] = 10*math.Log(trk.ISum/100.0/100.0) + math.Log(500.0) + 5
		trk.CodeISum[0] = buffloc
		trk.ISum = 0
	}
}

func rcvgetbuff(ini *SdrIni, buffloc uint64, n, ftype, dtype int, expbuf []byte) int {
	return RcvGetBuff(ini, buffloc, n, ftype, dtype, &sdrstat, expbuf)
}

// correlator: actual implementation
func correlator(data []byte, dtype int, ti float64, currnsamp int, carrfreq, oldremcarr, codefreq, oldremcode float64, corrp []int, corrn int, QQ, II []float64, remcode *float64, remcarr *float64, code []int16, codelen int) {
	// Allocate buffers
	dataI := make([]float64, currnsamp+64)
	dataQ := make([]float64, currnsamp+64)
	code_e := make([]int16, currnsamp+2*corrp[corrn-1])
	codebuf := code_e[corrp[corrn-1]:]

	// Mix local carrier
	// Use MixCarr and ResCode from sdrcmn.go
	// MixCarr expects II, QQ as []int16, so allocate and convert
	dataI16 := make([]int16, currnsamp+64)
	dataQ16 := make([]int16, currnsamp+64)
	*remcarr = MixCarr(data, dtype, ti, currnsamp, carrfreq, oldremcarr, dataI16, dataQ16)
	// Convert int16 to float64 for dot_xx
	for i := range dataI {
		dataI[i] = float64(dataI16[i])
		dataQ[i] = float64(dataQ16[i])
	}

	// Resample code
	*remcode = ResCode(code, codelen, oldremcode, corrp[corrn-1], ti*codefreq, currnsamp, codebuf)

	// Convert codebuf to float64 for dot_xx functions
	codebufF := make([]float64, len(codebuf))
	for i := range codebuf {
		codebufF[i] = float64(codebuf[i])
	}

	// Early/Late code slices
	early := codebufF[corrp[0]:]
	late := codebufF[:len(codebufF)-corrp[0]]

	// Multiply code and integrate
	dot_23(dataI, dataQ, codebufF, early, late, currnsamp, II, QQ)
	for i := 1; i < corrn; i++ {
		e := codebufF[corrp[i]:]
		l := codebufF[:len(codebufF)-corrp[i]]
		dot_22(dataI, dataQ, e, l, currnsamp, II[1+i*2:], QQ[1+i*2:])
	}
	for i := 0; i < 1+2*corrn; i++ {
		II[i] *= CSCALE
		QQ[i] *= CSCALE
	}
}
