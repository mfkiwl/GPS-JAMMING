// sdr.go : constants, types and function prototypes (Go version)
// Translated from sdr.h (C) 2014-2025
package main

import (
	"math"
)

// Constants ------------------------------------------------------------------
const (
	PI      = 3.1415926535897932
	DPI     = 2.0 * PI
	D2R     = PI / 180.0
	R2D     = 180.0 / PI
	CLIGHT  = 299792458.0
	ON      = 1
	OFF     = 0
	MAXBITS = 3000

	GPS_PI    = 3.1415926535897932
	MU        = 3.986005e14
	OMEGAEDOT = 7.2921151467e-5
	CTIME     = 2.99792458e8

	FEND_RTLSDR  = 3
	FEND_FRTLSDR = 8
	FEND_FILE    = 10
	FTYPE1       = 1
	DTYPEI       = 1
	DTYPEIQ      = 2

	MEMBUFFLEN    = 5000
	FILE_BUFFSIZE = 65536

	NFFTTHREAD    = 4
	ACQINTG_L1CA  = 10
	ACQINTG_G1    = 10
	ACQINTG_E1B   = 4
	ACQINTG_B1I   = 10
	ACQINTG_SBAS  = 10
	ACQHBAND      = 7000
	ACQSTEP       = 200
	ACQTH         = 3.0
	ACQSLEEP      = 2000

	LOOP_L1CA     = 10
	LOOP_G1       = 10
	LOOP_E1B      = 1
	LOOP_B1I      = 10
	LOOP_B1IG     = 2
	LOOP_SBAS     = 2
	LOOP_LEX      = 4

	NAVSYNCTH     = 50

	NAVRATE_L1CA    = 20
	NAVFLEN_L1CA    = 300
	NAVADDFLEN_L1CA = 2
	NAVPRELEN_L1CA  = 8
	NAVEPHCNT_L1CA  = 3

	NAVRATE_SBAS    = 2
	NAVFLEN_SBAS    = 1500
	NAVADDFLEN_SBAS = 12
	NAVPRELEN_SBAS  = 16
	NAVEPHCNT_SBAS  = 3

	PTIMING       = 68.802
	OBSINTERPN    = 80
	SNSMOOTHMS    = 100

	MAXGPSSATNO   = 210
	MAXGALSATNO   = 50
	MAXCMPSATNO   = 37

	CTYPE_L1CA    = 1
	CTYPE_L1CP    = 2
	CTYPE_L1CD    = 3
	CTYPE_L1CO    = 4
	CTYPE_L1SBAS  = 27
	CTYPE_NH10    = 28
	CTYPE_NH20    = 29

	LENSBASMSG    = 32
	LENSBASNOV    = 80

	LOW_PR        = 0.060 * CTIME
	HIGH_PR       = 0.092 * CTIME

	SNR_RESET_THRES = 15
	SNR_PVT_THRES   = 19

	GPS_WEEK      = 2360
	GPS_EPOCH_SECONDS = 315964800

	ET_TIMER   = 60.0
	SV_EL_PVT_MASK    = 15.0
	SV_EL_RESET_MASK  = 12.0

	MAX_MESSAGES = 100
	MSG_LENGTH   = 128
	PUR1         = 1
	PUR2         = 2
)

// Helper functions ----------------------------------------------------------
func Round(x float64) int {
	return int(math.Floor(x + 0.5))
}

// ...structs and types will follow...
