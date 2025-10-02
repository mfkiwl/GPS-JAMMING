package main

import (
	"fmt"
	"time"
)

// Assume all required types, constants, and helper functions are defined elsewhere

const MAXSAT = 32

var sdrekf SdrEkf

func init_sdrgui_messages() {
	for i := 0; i < MAX_MESSAGES; i++ {
		sdrgui.Messages[i] = ""
	}
	sdrgui.MessageCount = 0
}

func add_message(msg string) {
	hmsgmtx.Lock()
	defer hmsgmtx.Unlock()
	if sdrgui.MessageCount < MAX_MESSAGES {
		sdrgui.Messages[sdrgui.MessageCount] = msg
		sdrgui.MessageCount++
	} else {
		for i := 1; i < MAX_MESSAGES; i++ {
			sdrgui.Messages[i-1] = sdrgui.Messages[i]
		}
		sdrgui.Messages[MAX_MESSAGES-1] = msg
	}
}

func gps_to_utc(gps_week int, gps_tow float64) int64 {
	// GPS epoch: Jan 6, 1980
	const gps_epoch = 315964800 // Unix time for 1980-01-06 00:00:00
	gps_seconds := int64(gps_week)*604800 + int64(gps_tow)
	// Leap seconds handling omitted for simplicity
	return gps_epoch + gps_seconds
}

func updateNavStatusWin(counter int) {
	var prn, flagacq, flagsync, flagdec [32]int
	var vk1_v, rk1_v [32]float64
	var obs_v [MAXSAT*11]float64
	nsat := sdrstat.NsatValid
	lat := sdrstat.Lat
	lon := sdrstat.Lon
	hgt := sdrstat.Hgt
	gdop := sdrstat.Gdop
	clkBias := sdrstat.Xyzdt[3]
	gps_tow := sdrstat.ObsV[(sdrstat.ObsValidList[0]-1)*11+6]
	gps_week := int(sdrstat.ObsV[(sdrstat.ObsValidList[0]-1)*11+7])
	hobsvecmtx.Lock()
	for i := 0; i < 32; i++ {
		prn[i] = sdrch[i].Prn
		flagacq[i] = sdrch[i].FlagAcq
		flagsync[i] = sdrch[i].Nav.FlagSync
		flagdec[i] = sdrch[i].Nav.FlagDec
	}
	for m := 0; m < 32*11; m++ {
		obs_v[m] = sdrstat.ObsV[m]
	}
	for n := 0; n < 32; n++ {
		vk1_v[n] = sdrstat.Vk1V[n]
		rk1_v[n] = sdrekf.Rk1V[n]
	}
	hobsvecmtx.Unlock()
	utc_time_seconds := gps_to_utc(gps_week, gps_tow+clkBias/CTIME)
	utc_tm := time.Unix(utc_time_seconds, 0).UTC()
	bufferNav := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%03d", utc_tm.Year(), utc_tm.Month(), utc_tm.Day(), utc_tm.Hour(), utc_tm.Minute(), utc_tm.Second(), int(gps_tow*1000)%1000)
	fmt.Printf("ETIME|%.3f\n", sdrstat.ElapsedTime)
	fmt.Printf("TIME|%s\n", bufferNav)
	if sdrini.EkfFilterOn != 0 {
		fmt.Println("FILTER|EKF")
	} else {
		fmt.Println("FILTER|WLS")
	}
	bufferNav = ""
	for i := 0; i < 32; i++ {
		if flagacq[i] == 1 {
			bufferNav += fmt.Sprintf("%02d ", prn[i])
		}
	}
	fmt.Printf("ACQSV|%s\n", bufferNav)
	bufferNav = ""
	for i := 0; i < 32; i++ {
		if flagsync[i] == 1 {
			bufferNav += fmt.Sprintf("%02d ", prn[i])
		}
	}
	fmt.Printf("TRACKED|%s\n", bufferNav)
	bufferNav = ""
	for i := 0; i < 32; i++ {
		if flagdec[i] == 1 {
			bufferNav += fmt.Sprintf("%02d ", prn[i])
		}
	}
	fmt.Printf("DECODED|%s\n", bufferNav)
	bufferNav = fmt.Sprintf("%.7f|%.7f|%.1f|%.2f|%.5e", lat, lon, hgt, gdop, clkBias/CTIME)
	fmt.Printf("LLA|%02d|%s\n", nsat, bufferNav)
	for i := 0; i < nsat; i++ {
		prn := sdrstat.ObsValidList[i]
		bufferNav = fmt.Sprintf("%02d|%.1f|%d|%.1f|%.1f|%05.1f|%04.1f|%05.1f|%7.1f",
			int(obs_v[(prn-1)*11+0]),
			obs_v[(prn-1)*11+6],
			int(obs_v[(prn-1)*11+7]),
			obs_v[(prn-1)*11+8],
			obs_v[(prn-1)*11+5],
			obs_v[(prn-1)*11+9],
			obs_v[(prn-1)*11+10],
			rk1_v[(prn-1)],
			vk1_v[(prn-1)])
		fmt.Printf("OBS|%s\n", bufferNav)
	}
}
