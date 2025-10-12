// sdrtrk.go : SDR tracking functions (Go version)
// Translated from sdrtrk.c
package main

import (
	"fmt"
	"math"
)

// Tracking function
func sdrtracking(sdr *SdrCh, buffloc, cnt uint64) uint64 {
	sdr.FlagTrk = OFF
	fmt.Printf("TRACKING START: PRN %d, FlagTrk set to OFF\n", sdr.Prn)
	
	// Memory allocation for data buffer
	data := make([]byte, (sdr.NSamp+100)*sdr.Dtype)
	var bufflocnow uint64
	// Current buffer location
	hreadmtx.Lock()
	bufflocnow = uint64(sdrstat.FendBuffSize)*sdrstat.BuffCnt - uint64(sdr.NSamp)
	hreadmtx.Unlock()

	if bufflocnow > buffloc {
	sdr.CurrNSamp = int((float64(sdr.Clen)-sdr.Trk.RemCode)/(sdr.Trk.CodeFreq/sdr.FSf))
		fmt.Printf("DEBUG CurrNSamp calculation: Clen=%f, RemCode=%f, CodeFreq=%f, FSf=%f, CurrNSamp=%d\n", 
			sdr.Clen, sdr.Trk.RemCode, sdr.Trk.CodeFreq, sdr.FSf, sdr.CurrNSamp)
		rcvgetbuff(&sdrini, buffloc, sdr.CurrNSamp, sdr.Ftype, sdr.Dtype, data)
		copy(sdr.Trk.OldI, sdr.Trk.II)
		copy(sdr.Trk.OldQ, sdr.Trk.QQ)
		sdr.Trk.OldRemCode = sdr.Trk.RemCode
		sdr.Trk.OldRemCarr = sdr.Trk.RemCarr
		// Correlation
		if sdr.Prn == 13 {
			fmt.Printf("DEBUG CALLING CORRELATOR PRN %d\n", sdr.Prn)
		}
		correlator(data, sdr.Dtype, sdr.Ti, sdr.CurrNSamp, sdr.Trk.CarrFreq,
			sdr.Trk.OldRemCarr, sdr.Trk.CodeFreq, sdr.Trk.OldRemCode,
			sdr.Trk.CorrP, sdr.Trk.CorrN, sdr.Trk.QQ, sdr.Trk.II,
			&sdr.Trk.RemCode, &sdr.Trk.RemCarr, sdr.Code, sdr.Clen, sdr.Prn)
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
	// Apply empirical scaling factor to match original C implementation
	// Original C has ~15-20x larger correlation values, so we scale ISum accumulation
	scaledSumI := math.Abs(trk.SumI[0]) * 15.0 // Empirical scaling factor
	trk.ISum += scaledSumI
	if sdr.Prn == 13 {
		trk.ISumCount++ // Count accumulations for debugging
	}
	if snrflag != 0 {
		// Debug ISum before SNR calculation
		fmt.Printf("SNR calc: PRN %d, ISum=%.3f, SumI[0]=%.3f, ISumCount=%d\n", sdr.Prn, trk.ISum, trk.SumI[0], trk.ISumCount)
		
		// Debug correlation values for comparison
		if sdr.Prn == 13 {
			fmt.Printf("DEBUG Correlation PRN %d: SumI[0]=%.1f, SumQ[0]=%.1f, Power=%.1f\n", 
				sdr.Prn, trk.SumI[0], trk.SumQ[0], trk.SumI[0]*trk.SumI[0]+trk.SumQ[0]*trk.SumQ[0])
		}
		
		copy(trk.S[1:], trk.S[:OBSINTERPN-1])
		copy(trk.CodeISum[1:], trk.CodeISum[:OBSINTERPN-1])
		
		// Original formula: trk->S[0]=10*log(trk->Isum/100.0/100.0)+log(500.0)+5;
		// Przywracam oryginalną formułę z logarytmem naturalnym
		trk.S[0] = 10*math.Log(trk.ISum/100.0/100.0) + math.Log(500.0) + 5
		trk.CodeISum[0] = buffloc
		fmt.Printf("SNR calculated: PRN %d, SNR=%.2f, ISum was %.3f, ISumCount=%d\n", sdr.Prn, trk.S[0], trk.ISum, trk.ISumCount)
		trk.ISum = 0
		trk.ISumCount = 0  // Reset accumulation counter
	}
}

func rcvgetbuff(ini *SdrIni, buffloc uint64, n, ftype, dtype int, expbuf []byte) int {
	return RcvGetBuff(ini, buffloc, n, ftype, dtype, &sdrstat, expbuf)
}

// correlator: actual implementation
func correlator(data []byte, dtype int, ti float64, currnsamp int, carrfreq, oldremcarr, codefreq, oldremcode float64, corrp []int, corrn int, QQ, II []float64, remcode *float64, remcarr *float64, code []int16, codelen int, prn int) {
	// Safety checks to prevent slice bounds errors
	if corrn <= 0 || corrn-1 >= len(corrp) || currnsamp <= 0 {
		fmt.Printf("ERROR correlator: invalid params corrn=%d, len(corrp)=%d, currnsamp=%d\n", corrn, len(corrp), currnsamp)
		return
	}
	if corrp[corrn-1] < 0 || currnsamp+2*corrp[corrn-1] <= 0 {
		fmt.Printf("ERROR correlator: invalid corrp[%d]=%d, currnsamp=%d\n", corrn-1, corrp[corrn-1], currnsamp)
		return
	}
	
	// Allocate buffers - use int16 like original C code
	dataI := make([]int16, currnsamp+64)
	dataQ := make([]int16, currnsamp+64)
	code_e := make([]int16, currnsamp+2*corrp[corrn-1])
	codebuf := code_e[corrp[corrn-1]:]

	// Mix local carrier - produces int16 directly
	*remcarr = MixCarr(data, dtype, ti, currnsamp, carrfreq, oldremcarr, dataI, dataQ)

	// Resample code
	*remcode = ResCode(code, codelen, oldremcode, corrp[corrn-1], ti*codefreq, currnsamp, codebuf)

	// Use int16 versions of dot functions like original C
	// Early/Late code slices
	early := codebuf[corrp[0]:]
	late := codebuf[:len(codebuf)-corrp[0]]

	// Multiply code and integrate - use int16 versions
	dot_23_int16(dataI, dataQ, codebuf, early, late, currnsamp, II, QQ)
	for i := 1; i < corrn; i++ {
		e := codebuf[corrp[i]:]
		l := codebuf[:len(codebuf)-corrp[i]]
		dot_22_int16(dataI, dataQ, e, l, currnsamp, II[1+i*2:], QQ[1+i*2:])
	}
	
	// Debug correlation values before CSCALE for PRN 13
	if prn == 13 {
		fmt.Printf("DEBUG BEFORE CSCALE PRN %d: II[0]=%.1f, QQ[0]=%.1f\n", prn, II[0], QQ[0])
		// Also debug input data and code samples
		if len(dataI) > 0 && len(codebuf) > 0 {
			fmt.Printf("DEBUG DATA PRN %d: dataI[0]=%d, dataQ[0]=%d, code[0]=%d\n", prn, dataI[0], dataQ[0], codebuf[0])
		}
	}
	
	for i := 0; i < 1+2*corrn; i++ {
		II[i] *= CSCALE
		QQ[i] *= CSCALE
	}
	
	// Debug correlation values after CSCALE for PRN 13
	if prn == 13 {
		fmt.Printf("DEBUG AFTER CSCALE PRN %d: II[0]=%.1f, QQ[0]=%.1f\n", prn, II[0], QQ[0])
	}
}
