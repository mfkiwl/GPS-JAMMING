package main

import (
    "fmt"
    "sync"
    "time"
    "os"
)

// Global SDR state variables
// UpdateNavStatusWin: stub based on original C
func UpdateNavStatusWin(counter int) {
    // Print navigation status for each channel
    fmt.Printf("NavStatusWin [%d]:\n", counter)
    for i := 0; i < sdrini.Nch; i++ {
        ch := &sdrch[i]
        snr := 0.0
        el := 0.0
        if len(ch.Trk.S) > 0 {
            snr = ch.Trk.S[0]
        }
        if len(ch.Trk.L) > 0 {
            el = ch.Trk.L[0]
        }
        fmt.Printf("PRN:%2d SNR:%.1f El:%.1f Acq:%d Trk:%d\n", ch.Prn, snr, el, ch.FlagAcq, ch.FlagTrk)
    }
}

// SDRPRINTF: stub based on original C
func SDRPRINTF(format string, a ...interface{}) {
    fmt.Printf(format, a...)
}

// rcvquit: stub based on original C
func rcvquit(ini *SdrIni) int {
    // Stop data grabber, free memory, close files
    sdrstat.StopFlag = ON
    if ini.fp != nil {
        ini.fp.Close()
        ini.fp = nil
    }
    sdrstat.Buff = nil
    sdrstat.TmpBuff = nil
    fmt.Println("rcvquit: resources cleaned up")
    return 0
}
// Mutexes (simulate with sync.Mutex)
var hbuffmtx sync.Mutex
var hreadmtx sync.Mutex
var hfftmtx sync.Mutex
var hobsmtx sync.Mutex
var hresetmtx sync.Mutex
var hobsvecmtx sync.Mutex
var hmsgmtx sync.Mutex

// openHandles: initialize mutexes (translated from C)
func openHandles() {
    // In Go, mutexes are zero-value initialized, but you can reset if needed
    hbuffmtx = sync.Mutex{}
    hreadmtx = sync.Mutex{}
    hfftmtx = sync.Mutex{}
    hobsmtx = sync.Mutex{}
    hresetmtx = sync.Mutex{}
    hobsvecmtx = sync.Mutex{}
    hmsgmtx = sync.Mutex{}
}

// syncThread: translated from C syncthread
func syncThread(arg interface{}) {
    // Copy tracking data, compute pseudorange, update observation vectors
    for sdrstat.StopFlag == 0 {
        hobsvecmtx.Lock()
        
        // Copy tracking data to ObsV like original C code
        for i := 0; i < sdrini.Nch; i++ {
            ch := &sdrch[i]
            
            // Check if this channel has good tracking (temporarily ignore ephemeris for testing)
            if ch.FlagTrk == 1 && ch.FlagAcq == 1 && len(ch.Trk.S) > 0 {
                prn := ch.Prn
                
                // Make sure ObsV is large enough
                if len(sdrstat.ObsV) <= (prn-1)*11+10 {
                    newSize := prn*11 + 11
                    newObsV := make([]float64, newSize)
                    copy(newObsV, sdrstat.ObsV)
                    sdrstat.ObsV = newObsV
                }
                
                // Fill observation data like original C code
                sdrstat.ObsV[(prn-1)*11+0] = float64(prn)  // PRN
                sdrstat.ObsV[(prn-1)*11+1] = 1             // Validity flag
                
                // Pseudorange (simplified - needs proper calculation)
                if len(ch.Trk.CodeI) > 0 {
                    sdrstat.ObsV[(prn-1)*11+5] = float64(ch.Trk.CodeI[0]) * CLIGHT * ch.Ti  // PR (simplified)
                }
                
                // Time of week (simplified)
                if len(ch.Trk.Tow) > 0 {
                    sdrstat.ObsV[(prn-1)*11+6] = ch.Trk.Tow[0]  // TOW
                }
                
                // GPS week (from nav data)
                sdrstat.ObsV[(prn-1)*11+7] = float64(ch.Nav.SdrEph.WeekGpst)  // GPS week
                
                // SNR
                if len(ch.Trk.S) > 0 {
                    sdrstat.ObsV[(prn-1)*11+8] = ch.Trk.S[0]  // SNR
                }
                
                // Elevation (if available)
                if len(ch.Trk.L) > 0 {
                    sdrstat.ObsV[(prn-1)*11+10] = ch.Trk.L[0]  // Elevation
                }
                
                fmt.Printf("syncThread: Set obs for PRN %d, SNR=%.1f, valid=1\n", prn, ch.Trk.S[0])
            }
        }
        hobsvecmtx.Unlock()
        
        // Call PVT processing like in original C code
        ret := updateObsList()
        if ret == 0 {
            precheckObs()
            
            mlock(&hobsvecmtx)
            nsat := sdrstat.NsatValid
            unmlock(&hobsvecmtx)
            
            fmt.Printf("syncThread: Found %d valid satellites\n", nsat)
            
            if nsat >= 4 {
                ret := pvtProcessor()
                if ret != 0 {
                    fmt.Println("errorDetected: exiting pvtProcessor")
                } else {
                    // Print coordinates when PVT solution is successful
                    mlock(&hobsvecmtx)
                    lat := sdrstat.Lat
                    lon := sdrstat.Lon
                    hgt := sdrstat.Hgt
                    gdop := sdrstat.Gdop
                    unmlock(&hobsvecmtx)
                    
                    fmt.Printf("=== POSITION SOLUTION ===\n")
                    fmt.Printf("Latitude:  %.8f°\n", lat)
                    fmt.Printf("Longitude: %.8f°\n", lon)
                    fmt.Printf("Height:    %.3f m\n", hgt)
                    fmt.Printf("GDOP:      %.2f\n", gdop)
                    fmt.Printf("Satellites: %d\n", nsat)
                    fmt.Printf("=========================\n")
                    os.Exit(0)
                }
            } else {
                if nsat > 0 {
                    fmt.Printf("pvtProcessor: PVT not solved for, less than four SVs (have %d)\n", nsat)
                }
            }
        }
        
        time.Sleep(100 * time.Millisecond)  // Slower for debugging
    }
    fmt.Println("SDR syncthread finished!")
}

// sdrThread: translated from C sdrthread
func sdrThread(arg interface{}) {
    sdr := arg.(*SdrCh)
    fmt.Printf("SDRTHREAD START: PRN %d\n", sdr.Prn)
    var buffloc uint64
    var acqpower []float64
    var loopcnt uint64
    var cnt uint64
    
    for sdrstat.StopFlag == 0 {
        if sdr.FlagAcq == 0 {
            acqpower = make([]float64, sdr.NSamp*sdr.Acq.Nfreq)
            buffloc = SdrAcquisition(&sdrini, &sdrstat, sdr, acqpower)
        }
        if sdr.FlagAcq != 0 {
            fmt.Printf("BEFORE TRACKING: PRN %d, FlagAcq=%d, FlagTrk=%d\n", sdr.Prn, sdr.FlagAcq, sdr.FlagTrk)
            sdrtracking(sdr, buffloc, cnt)
            fmt.Printf("AFTER TRACKING: PRN %d, FlagTrk=%d\n", sdr.Prn, sdr.FlagTrk)
            
            // Tracking and SNR calculation logic
            if sdr.FlagTrk != 0 {
                fmt.Printf("DEBUG TRK: PRN %d, FlagSync=%d, SwLoop=%d\n", sdr.Prn, sdr.Nav.FlagSync, sdr.Nav.SwLoop)
                cumsumcorr(&sdr.Trk, int(sdr.Nav.OCode[sdr.Nav.OCodeI]))
                sdr.Trk.FlagLoopFilter = 0
                if sdr.Nav.FlagSync == 0 && sdr.Nav.SwLoop == 0 {
                    fmt.Printf("DEBUG LOOP: PRN %d, FlagSync=0, SwLoop=0 - using Prm1\n", sdr.Prn)
                    pll(sdr, &sdr.Trk.Prm1, sdr.CTime)
                    dll(sdr, &sdr.Trk.Prm1, sdr.CTime)
                    sdr.Trk.FlagLoopFilter = 1
                } else if sdr.Nav.SwLoop != 0 {
                    fmt.Printf("DEBUG LOOP: PRN %d, SwLoop=%d - entering SNR section (FlagSync=%d)\n", sdr.Prn, sdr.Nav.SwLoop, sdr.Nav.FlagSync)
                    pll(sdr, &sdr.Trk.Prm2, float64(sdr.Trk.LoopMs)/1000)
                    dll(sdr, &sdr.Trk.Prm2, float64(sdr.Trk.LoopMs)/1000)
                    sdr.Trk.FlagLoopFilter = 1
                    
                    // SNR calculation timing - INSIDE SwLoop block
                    modVal := uint64(SNSMOOTHMS)/uint64(sdr.Trk.LoopMs)
                    condition := loopcnt%modVal == 0
                    fmt.Printf("DEBUG SNR TIMING: PRN %d, loopcnt=%d, modVal=%d, condition=%t, SNSMOOTHMS=%d, LoopMs=%d\n", 
                        sdr.Prn, loopcnt, modVal, condition, SNSMOOTHMS, sdr.Trk.LoopMs)
                    if condition {
                        fmt.Printf("DEBUG SNR: PRN %d, loopcnt=%d, condition met - calling setobsdata with snrflag=1\n", sdr.Prn, loopcnt)
                        setobsdata(sdr, buffloc, loopcnt, &sdr.Trk, 1)
                        fmt.Printf("DEBUG SNR calculated: PRN %d, SNR=%.2f\n", sdr.Prn, sdr.Trk.S[0])
                    } else {
                        setobsdata(sdr, buffloc, loopcnt, &sdr.Trk, 0)
                    }
                    loopcnt++
                } else {
                    fmt.Printf("DEBUG LOOP: PRN %d, FlagSync=1, SwLoop=0 - no SNR calculation\n", sdr.Prn)
                }
                
                // Clear cumsum if loop filter was used (like in original C)
                if sdr.Trk.FlagLoopFilter != 0 {
                    clearcumsumcorr(&sdr.Trk)
                }
                
                // Increment cnt like in original C code
                cnt++
                buffloc += uint64(sdr.CurrNSamp)
            }
        }
        time.Sleep(10 * time.Millisecond)
    }
    fmt.Printf("SDR channel thread finished for PRN %d!\n", sdr.Prn)
}

// dataThread: translated from C datathread
func dataThread(arg interface{}) {
    // Data grabber loop: read samples from file and push to buffer
    if RcvGrabStart(&sdrini) < 0 {
        quitsdr(&sdrini, 4)
        return
    }
    for sdrstat.StopFlag == 0 {
        if RcvGrabData(&sdrini, &sdrstat) < 0 {
            sdrstat.StopFlag = ON
        }
    }
    fmt.Println("SDR dataThread finished!")
}
var sdrini SdrIni
var sdrstat SdrStat
var sdrch [32]SdrCh // adjust size as needed
var sdrgui SdrGui

// RcvInit: receiver initialization stub

func startsdr() {
    // Receiver initialization
    if err := RcvInit(&sdrini, &sdrstat); err != nil {
        fmt.Printf("error: RcvInit: %v\n", err)
        quitsdr(&sdrini, 1)
        return
    }
    
    // Initialize SDR channel struct
    for i := 0; i < sdrini.Nch; i++ {
        err := InitSdrCh(i+1, sdrini.Sys[i], sdrini.Prn[i], sdrini.Ctype[i],
            sdrini.Dtype[sdrini.Ftype[i]-1], sdrini.Ftype[i],
            sdrini.FGain[sdrini.Ftype[i]-1], sdrini.FBias[sdrini.Ftype[i]-1],
            sdrini.FClock[sdrini.Ftype[i]-1], sdrini.FCf[sdrini.Ftype[i]-1],
            sdrini.FSf[sdrini.Ftype[i]-1], sdrini.FIf[sdrini.Ftype[i]-1],
            &sdrch[i], &sdrini)
        if err != nil {
            fmt.Printf("error: InitSdrCh: %v\n", err)
            quitsdr(&sdrini, 2)
            return
        }
    }
    // Mutexes and events
        openHandles()

    // Create threads
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
            syncThread(nil)
    }()

    // SDR channel threads
    for i := 0; i < sdrini.Nch; i++ {
        if sdrch[i].Ctype == CTYPE_L1CA || sdrch[i].Ctype == CTYPE_L1SBAS {
            wg.Add(1)
            go func(idx int) {
                defer wg.Done()
                    sdrThread(&sdrch[idx])
            }(i)
        }
    }

    // Data grabber thread
    wg.Add(1)
    go func() {
        defer wg.Done()
            dataThread(nil)
    }()

    // Start timer
    startTime := time.Now()
    sdrstat.ElapsedTime = 0.0
    counter := 0

    // Main while loop
    for sdrstat.StopFlag == 0 {
        // Update elapsed time
        currentTime := time.Now()
        sdrstat.ElapsedTime = currentTime.Sub(startTime).Seconds()
        // Update both status windows
        UpdateNavStatusWin(counter)
        counter++
        // Update GUI at desired rate
        time.Sleep(100 * time.Millisecond)
    }

    // Cleanup threads and messages
    for i := 0; i < sdrgui.MessageCount; i++ {
        sdrgui.Messages[i] = ""
    }

    // Wait for threads to finish
    wg.Wait()

    // SDR termination
    quitsdr(&sdrini, 0)
    SDRPRINTF("GNSS-SDRLIB is finished!\n")
}

func quitsdr(ini *SdrIni, stop int) {
    if stop == 1 {
        return
    }
    // SDR termination
    rcvquit(ini)
    if stop == 2 {
        return
    }
    // Free memory
    for i := 0; i < ini.Nch; i++ {
        freesdrch(&sdrch[i]) // stub below
    }
    if stop == 3 {
        return
    }
    // Mutexes and events
    closehandles() // stub below

	if stop == 4 {
        return
    }
}
// freesdrch: stub for freeing SDR channel (Go GC handles memory)

func freesdrch(ch *SdrCh) {
    // In Go, memory is garbage collected. Zero the struct if needed.
    *ch = SdrCh{}
}

// closehandles: stub for closing mutexes (not needed in Go)
func closehandles() {
    // In Go, nothing needed for closing mutexes
}

// sleepms: sleep for ms milliseconds
func sleepms(ms int) {
    time.Sleep(time.Duration(ms) * time.Millisecond)
}

// mlock: lock a mutex
func mlock(mtx *sync.Mutex) {
    mtx.Lock()
}

// unmlock: unlock a mutex
func unmlock(mtx *sync.Mutex) {
    mtx.Unlock()
}

func sdrthread(arg interface{}) interface{} {
    sdr := arg.(*SdrCh)
    fmt.Printf("SDRTHREAD START: PRN %d\n", sdr.Prn)
    var buffloc, bufflocnow, cnt, loopcnt uint64
    var acqpower []float64
    var snr, el float64
    var ret int
    startAcqTimer := time.Now()
    var elapsedAcqTime float64
    sleepms(sdr.No * 500)
    for sdrstat.StopFlag == 0 {
        loopStart := time.Now()
        currentTime := time.Now()
        if sdr.FlagAcq != 0 {
            elapsedAcqTime = currentTime.Sub(startAcqTimer).Seconds()
        }
        if elapsedAcqTime > 60 {
            mlock(&hobsmtx)
            snr = sdr.Trk.S[0]
            unmlock(&hobsmtx)
            if snr < SNR_RESET_THRES {
                msg := fmt.Sprintf("%.3f  G%02d resetting with SNR of %.1f and flagacq of %d\n", sdrstat.ElapsedTime, sdr.Prn, snr, sdr.FlagAcq)
                add_message(msg)
                i := sdr.Prn - 1
                ret = resetStructs(&sdrch[i])
                elapsedAcqTime = 0
                if ret == -1 {
                    fmt.Println("resetStructs: error")
                }
                continue
            }
        }
        if elapsedAcqTime > 60 {
            nav := sdr.Nav
            eph := nav.SdrEph
            if nav.FlagDec == 0 || nav.FlagSync == 0 || (eph.WeekGpst < GPS_WEEK) {
                msg := fmt.Sprintf("%.3f  G%02d resetting, flagdec:%d, flagsync:%d, Week:%d\n", sdrstat.ElapsedTime, sdr.Prn, nav.FlagDec, nav.FlagSync, eph.WeekGpst)
                add_message(msg)
                i := sdr.Prn - 1
                ret = resetStructs(&sdrch[i])
                elapsedAcqTime = 0
                if ret == -1 {
                    fmt.Println("resetStructs: error")
                }
                continue
            }
        }
        if elapsedAcqTime > 60 {
            mlock(&hobsmtx)
            i := sdr.Prn - 1
            el = sdrstat.ObsV[i*11+10]
            unmlock(&hobsmtx)
            if el < SV_EL_RESET_MASK {
                msg := fmt.Sprintf("%.3f  G%02d resetting, SV el: %.1f\n", sdrstat.ElapsedTime, i+1, el)
                add_message(msg)
                i := sdr.Prn - 1
                ret = resetStructs(&sdrch[i])
                elapsedAcqTime = 0
                if ret == -1 {
                    fmt.Println("resetStructs: error")
                }
                continue
            }
        }
        // Acquisition
        if sdr.FlagAcq == 0 {
            acqpower = make([]float64, sdr.NSamp*sdr.Acq.Nfreq)
            acqStart := time.Now()
            buffloc = SdrAcquisition(&sdrini, &sdrstat, sdr, acqpower)
            acqTime := time.Since(acqStart).Seconds()
            if acqTime > 0.1 { // Log slow acquisitions
                fmt.Printf("SLOW ACQ: PRN %d took %.3f seconds\n", sdr.Prn, acqTime)
            }
            startAcqTimer = time.Now()
        }
        // Tracking
        if sdr.FlagAcq != 0 {
            trkStart := time.Now()
            bufflocnow = sdrtracking(sdr, buffloc, cnt)
            trkTime := time.Since(trkStart).Seconds()
            if trkTime > 0.01 { // Log tracking longer than 10ms
                fmt.Printf("SLOW TRK: PRN %d took %.3f seconds\n", sdr.Prn, trkTime)
            }
            if sdr.FlagTrk != 0 {
                cumsumcorr(&sdr.Trk, int(sdr.Nav.OCode[sdr.Nav.OCodeI]))
                sdr.Trk.FlagLoopFilter = 0
                if sdr.Nav.FlagSync == 0 {
                    pll(sdr, &sdr.Trk.Prm1, sdr.CTime)
                    dll(sdr, &sdr.Trk.Prm1, sdr.CTime)
                    sdr.Trk.FlagLoopFilter = 1
                } else if sdr.Nav.SwLoop != 0 {
                    // Normal SwLoop condition without forced debugging
                    pll(sdr, &sdr.Trk.Prm2, float64(sdr.Trk.LoopMs)/1000)
                    dll(sdr, &sdr.Trk.Prm2, float64(sdr.Trk.LoopMs)/1000)
                    sdr.Trk.FlagLoopFilter = 2
                    mlock(&hobsmtx)
                    modVal := uint64(SNSMOOTHMS/sdr.Trk.LoopMs)
                    condition := loopcnt%modVal == 0
                    if condition {
                        setobsdata(sdr, buffloc, cnt, &sdr.Trk, 1)
                    } else {
                        setobsdata(sdr, buffloc, cnt, &sdr.Trk, 0)
                    }
                    unmlock(&hobsmtx)
                    loopcnt++
                }
                if sdr.Trk.FlagLoopFilter != 0 {
                    clearcumsumcorr(&sdr.Trk)
                }
                cnt++
                buffloc += uint64(sdr.CurrNSamp)
            }
        }
        sdr.Trk.BuffLoc = buffloc
        loopTime := time.Since(loopStart).Seconds()
        if loopTime > 0.001 { // Log loops longer than 1ms
            fmt.Printf("SLOW LOOP: PRN %d took %.3f seconds\n", sdr.Prn, loopTime)
        }
    }
    if sdr.FlagAcq != 0 {
        SDRPRINTF("SDR channel %s thread finished! Delay=%d [ms]\n", sdr.SatStr, int((bufflocnow-buffloc)/uint64(sdr.NSamp)))
    } else {
        SDRPRINTF("SDR channel %s thread finished!\n", sdr.SatStr)
    }
    return nil
}

func datathread(arg interface{}) interface{} {
    if RcvGrabStart(&sdrini) < 0 {
        quitsdr(&sdrini, 4)
    }
    for sdrstat.StopFlag == 0 {
        if RcvGrabData(&sdrini, &sdrstat) < 0 {
            sdrstat.StopFlag = ON
        }
    }
    return nil
}

func resetStructs(arg interface{}) int {
    sdr := arg.(*SdrCh)
    mlock(&hobsvecmtx)
    prn := sdr.Prn
    i := prn - 1
    sdrch[i] = SdrCh{} // zero value
    sdrstat.AzElCalculatedFlag = 0
    err := InitSdrCh(i+1, sdrini.Sys[i], sdrini.Prn[i], sdrini.Ctype[i],
        sdrini.Dtype[sdrini.Ftype[i]-1], sdrini.Ftype[i],
        sdrini.FGain[sdrini.Ftype[i]-1], sdrini.FBias[sdrini.Ftype[i]-1],
        sdrini.FClock[sdrini.Ftype[i]-1], sdrini.FCf[sdrini.Ftype[i]-1],
        sdrini.FSf[sdrini.Ftype[i]-1], sdrini.FIf[sdrini.Ftype[i]-1],
        &sdrch[i], &sdrini)
    if err != nil {
        SDRPRINTF("error: initsdrch call in resetStructs: %v\n", err)
        quitsdr(&sdrini, 2)
    }
    unmlock(&hobsvecmtx)
    msg := fmt.Sprintf("%.3f  resetStructs: G%02d channel has been reset and will reacquire in 10s", sdrstat.ElapsedTime, prn)
    add_message(msg)
    sleepms(10000)
    return 0
}

func checkObsDelay(prn int) int {
    i := prn - 1
    resetFlag := 0
    ret := 0
    nsat := sdrstat.NsatValid
    if sdrch[i].FlagAcq == 1 {
        if sdrch[i].ElapsedTimeNav > 90 {
            resetFlag = 1
            for j := 0; j < nsat; j++ {
                if prn == sdrstat.ObsValidList[j] {
                    resetFlag = 0
                }
            }
        }
    }
    if resetFlag != 0 {
    msg := fmt.Sprintf("%.3f  checkObsDelay: resetting G%02d due to mismatch", sdrstat.ElapsedTime, prn)
        add_message(msg)
        ret = resetStructs(&sdrch[i])
        if ret == -1 {
            fmt.Println("resetStructs: error")
        }
    }
    return 0
}

func main() {
    // Print startup message
    fmt.Println("GNSS-SDRLIB starting...")

    // Get file path from command line argument
    if len(os.Args) <= 1 {
        fmt.Println("No input file specified. Usage: sdrmain <file>")
        return
    }
    filename := os.Args[1]

    // Load initial values from file
    if err := LoadInit(&sdrini, filename); err != nil {
        fmt.Printf("error: LoadInit: %v\n", err)
        return
    }

    // Start SDR decoding process
    startsdr()
}