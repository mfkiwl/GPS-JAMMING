// sdrrcv.go : SDR receiver functions (Go version)
// Translated from sdrrcv.c
package main

import (
	"os"
	"sync"
)

var buffmtx sync.Mutex
var readmtx sync.Mutex
var err error // global error variable for fallback returns

// SDR receiver initialization
func RcvInit(ini *SdrIni, sdrstat *SdrStat) error {
	sdrstat.Buff = nil
	sdrstat.TmpBuff = nil
	switch ini.Fend {
	case FEND_FILE:
		var err error
		ini.fp, err = os.Open(ini.File)
		if err != nil {
			SDRPRINTF("error: failed to open file: %s\n", ini.File)
			return err
		}
		sdrstat.FendBuffSize = FILE_BUFFSIZE
		sdrstat.BuffSize = FILE_BUFFSIZE * MEMBUFFLEN
		if ini.fp != nil {
			sdrstat.Buff = make([]byte, ini.Dtype[0]*sdrstat.BuffSize)
			if sdrstat.Buff == nil {
				SDRPRINTF("error: failed to allocate memory for the buffer\n")
				return err
			}
		}
	default:
		return err
	}
	return nil
}

// SDR receiver quit
func RcvQuit(ini *SdrIni, sdrstat *SdrStat) error {
	switch ini.Fend {
	case FEND_FILE:
		if ini.fp != nil {
			ini.fp.Close()
			ini.fp = nil
		}
	default:
		return err
	}
	if sdrstat.Buff != nil {
		sdrstat.Buff = nil
	}
	if sdrstat.TmpBuff != nil {
		sdrstat.TmpBuff = nil
	}
	return nil
}

// Start grabber of front end (tu: tylko plik, zwraca 0)
func RcvGrabStart(ini *SdrIni) int {
	switch ini.Fend {
	default:
		return 0
	}
	return 0
}

// Push data to memory buffer from front end (tu: tylko plik)
func RcvGrabData(ini *SdrIni, sdrstat *SdrStat) int {
	switch ini.Fend {
	case FEND_FILE:
		FilePushToMemBuf(ini, sdrstat)
		Sleepms(5)
	default:
		return -1
	}
	return 0
}

// Get current data buffer from memory buffer
func RcvGetBuff(ini *SdrIni, buffloc uint64, n, ftype, dtype int, sdrstat *SdrStat, expbuf []byte) int {
	switch ini.Fend {
	case FEND_FILE:
		FileGetBuff(buffloc, n, ftype, dtype, expbuf)
	default:
		return -1
	}
	return 0
}

func FilePushToMemBuf(ini *SdrIni, sdrstat *SdrStat) {
	var nread int
	buffmtx.Lock()
	if ini.fp != nil {
		buf := make([]byte, ini.Dtype[0]*FILE_BUFFSIZE)
		n, err := ini.fp.Read(buf)
		nread = n
		if err != nil && err.Error() != "EOF" {
			// handle error
		}
		copy(sdrstat.Buff[int((sdrstat.BuffCnt%uint64(MEMBUFFLEN))*uint64(ini.Dtype[0])*uint64(FILE_BUFFSIZE)):], buf[:nread])
	}
	buffmtx.Unlock()
	if ini.fp != nil && nread < ini.Dtype[0]*FILE_BUFFSIZE {
		sdrstat.StopFlag = ON
		SDRPRINTF("end of file!\n")
	}
	readmtx.Lock()
	sdrstat.BuffCnt++
	readmtx.Unlock()
}

func FileGetBuff(buffloc uint64, n, ftype, dtype int, expbuf []byte) {
	membuffloc := dtype*int(buffloc) % (MEMBUFFLEN*dtype*FILE_BUFFSIZE)
	n = dtype * n
	nout := (membuffloc + n) - (MEMBUFFLEN * dtype * FILE_BUFFSIZE)
	buffmtx.Lock()
	if nout > 0 {
		copy(expbuf, sdrstat.Buff[membuffloc:membuffloc+n-nout])
		copy(expbuf[n-nout:], sdrstat.Buff[:nout])
	} else {
		copy(expbuf, sdrstat.Buff[membuffloc:membuffloc+n])
	}
	buffmtx.Unlock()
}
