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
        hobsmtx.Lock()
        for i := 0; i < sdrini.Nch; i++ {
            ch := &sdrch[i]
            // Safe copy SNR and pseudorange to ObsV
            snr := 0.0
            el := 0.0
            if len(ch.Trk.S) > 0 {
                snr = ch.Trk.S[0]
            }
            if len(ch.Trk.L) > 0 {
                el = ch.Trk.L[0]
            }
            if len(sdrstat.ObsV) > i*11+1 {
                sdrstat.ObsV[i*11+0] = snr
                sdrstat.ObsV[i*11+1] = el
            }
        }
        hobsmtx.Unlock()
        time.Sleep(10 * time.Millisecond)
    }
    fmt.Println("SDR syncthread finished!")
}

// sdrThread: translated from C sdrthread
func sdrThread(arg interface{}) {
    sdr := arg.(*SdrCh)
    var buffloc uint64
    var acqpower []float64
    for sdrstat.StopFlag == 0 {
        if sdr.FlagAcq == 0 {
            acqpower = make([]float64, sdr.NSamp*sdr.Acq.Nfreq)
            buffloc = SdrAcquisition(&sdrini, &sdrstat, sdr, acqpower)
        }
        if sdr.FlagAcq != 0 {
            sdrtracking(sdr, buffloc, 0)
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
    var buffloc, bufflocnow, cnt, loopcnt uint64
    var acqpower []float64
    var snr, el float64
    var ret int
    startAcqTimer := time.Now()
    var elapsedAcqTime float64
    sleepms(sdr.No * 500)
    for sdrstat.StopFlag == 0 {
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
            buffloc = SdrAcquisition(&sdrini, &sdrstat, sdr, acqpower)
            startAcqTimer = time.Now()
        }
        // Tracking
        if sdr.FlagAcq != 0 {
            bufflocnow = sdrtracking(sdr, buffloc, cnt)
            if sdr.FlagTrk != 0 {
                cumsumcorr(&sdr.Trk, int(sdr.Nav.OCode[sdr.Nav.OCodeI]))
                sdr.Trk.FlagLoopFilter = 0
                if sdr.Nav.FlagSync == 0 {
                    pll(sdr, &sdr.Trk.Prm1, sdr.CTime)
                    dll(sdr, &sdr.Trk.Prm1, sdr.CTime)
                    sdr.Trk.FlagLoopFilter = 1
                } else if sdr.Nav.SwLoop != 0 {
                    pll(sdr, &sdr.Trk.Prm2, float64(sdr.Trk.LoopMs)/1000)
                    dll(sdr, &sdr.Trk.Prm2, float64(sdr.Trk.LoopMs)/1000)
                    sdr.Trk.FlagLoopFilter = 2
                    mlock(&hobsmtx)
                    if loopcnt%uint64(SNSMOOTHMS/sdr.Trk.LoopMs) == 0 {
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