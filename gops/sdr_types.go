// sdr_types.go : types and structs translated from sdr.h (C)
package main

import "os"

// Typy i struktury z sdr.h

// Kompleksowy typ dla FFT
// W Go używamy wbudowanego complex64

// SDR Initialization struct
// Uwaga: MAXSAT, SYS_* muszą być zdefiniowane w innym pliku/stałej

type SdrIni struct {
	Fend        int
	FGain       [2]int
	FBias       [2]int
	FClock      [2]int
	FCf         [2]float64
	FSf         [2]float64
	FIf         [2]float64
	Dtype       [2]int
	File        string
	fp          *os.File // file pointer for FEND_FILE
	UseIf       int
	Nch         int
	NchL1       int
	NchL2       int
	NchL5       int
	NchL6       int
	Prn         []int // MAXSAT
	Sys         []int // MAXSAT
	Ctype       []int // MAXSAT
	Ftype       []int // MAXSAT
	PltAcq      int
	PltTrk      int
	PltSpec     int
	OutMs       int
	Sbas        int
	SnrThreshold int
	Xu0V        [3]int
	SbasPort    int
	TrkCorrN    int
	TrkCorrD    int
	TrkCorrP    int
	TrkDllB     [2]float64
	TrkPllB     [2]float64
	TrkFllB     [2]float64
	RtlsdrPpmErr int
	EkfFilterOn  int
}

type SdrStat struct {
	StopFlag    int
	ResetSyncThreadFlag int
	SpecFlag    int
	BuffSize    int
	FendBuffSize int
	Buff        []byte
	TmpBuff     []byte
	BuffCnt     uint64
	PrintFlag   int
	Lat         float64
	Lon         float64
	Hgt         float64
	Gdop        float64
	Nsat        int
	NsatRaw     int
	NsatValid   int
	SatList     []int // MAXSAT
	ObsValidList []int // MAXSAT
	ObsV        []float64 // 11*MAXSAT
	Vk1V        []float64 // MAXSAT
	ClkBias     float64
	Xyzdt       [4]float64
	ElapsedTime float64
	AzElCalculatedFlag int
}

type SdrObs struct {
	Prn   int
	Sys   int
	Tow   float64
	Week  int
	P     float64
	L     float64
	D     float64
	S     float64
}

type SdrAcq struct {
	Intg      int
	HBand     float64
	Step      float64
	Nfreq     int
	Freq      []float64
	AcqCodeI  int
	FreqI     int
	AcqFreq   float64
	Nfft      int
	Cn0       float64
	PeakR     float64
}

type SdrTrkPrm struct {
	PllB   float64
	DllB   float64
	FllB   float64
	DllW2  float64
	DllAw  float64
	PllW2  float64
	PllAw  float64
	FllW   float64
}

type SdrTrk struct {
	CodeFreq    float64
	CarrFreq    float64
	RemCode     float64
	RemCarr     float64
	OldRemCode  float64
	OldRemCarr  float64
	CodeNco     float64
	CodeErr     float64
	CarrNco     float64
	CarrErr     float64
	FreqErr     float64
	BuffLoc     uint64
	Tow         []float64 // OBSINTERPN
	CodeI       []uint64  // OBSINTERPN
	CodeISum    []uint64  // OBSINTERPN
	CntOut      []uint64  // OBSINTERPN
	RemCOut     []float64 // OBSINTERPN
	L           []float64 // OBSINTERPN
	D           []float64 // OBSINTERPN
	S           []float64 // OBSINTERPN
	II          []float64
	QQ          []float64
	OldI        []float64
	OldQ        []float64
	SumI        []float64
	SumQ        []float64
	OldSumI     []float64
	OldSumQ     []float64
	ISum        float64
	Loop        int
	LoopMs      int
	FlagPolarityAdd int
	FlagRemCarrAdd  int
	FlagLoopFilter  int
	CorrN       int
	CorrP       []int
	CorrX       []float64
	Ne          int
	Nl          int
	Prm1        SdrTrkPrm
	Prm2        SdrTrkPrm
}

type SdrEph struct {
	// eph_t z rtklib.h, tu jako interface{} do czasu implementacji
	Eph        interface{}
	Ctype      int
	TowGpst    float64
	WeekGpst   int
	Cnt        int
	CntTh      int
	Update     int
	Prn        int
	Tk         [3]int
	Nt         int
	N4         int
	S1Cnt      int
	TocGst     float64
	WeekGst    int
}

type SdrSbas struct {
	Msg         [LENSBASMSG]byte
	NovatelMsg  [LENSBASNOV]byte
	Id          int
	Week        int
	Tow         float64
}

type SdrNav struct {
	FpNav       interface{} // FILE* w C, tu jako interface{}
	Ctype       int
	Rate        int
	Flen        int
	AddFlen     int
	PreBits     [32]int
	PreLen      int
	Bit         int
	BitI        int
	Cnt         int
	BitIP       float64
	FBits       []int
	FBitsDec    []int
	Update      int
	BitSync     []int
	SyncI       int
	FirstSf     uint64
	FirstSfCnt  uint64
	FirstSfTow  float64
	Polarity    int
	FlagPol     int
	Fec         interface{} // FEC
	OCode       []int16
	OCodeI      int
	SwSync      int
	SwReset     int
	SwLoop      int
	FlagSync    int
	FlagSyncF   int
	FlagTow     int
	FlagDec     int
	SdrEph      SdrEph
	Sbas        SdrSbas
}

type SdrCh struct {
	No          int
	Sat         int
	Sys         int
	Prn         int
	SatStr      string
	Ctype       int
	Dtype       int
	Ftype       int
	FCf         float64
	FSf         float64
	FIf         float64
	FGain       int
	FBias       int
	FClock      int
	FOffset     float64
	Code        []int16
	XCode       []complex64
	Clen        int
	CRate       float64
	CTime       float64
	Ti          float64
	Ci          float64
	NSamp       int
	CurrNSamp   int
	NSampChip   int
	Acq         SdrAcq
	Trk         SdrTrk
	Nav         SdrNav
	FlagAcq     int
	FlagTrk     int
	ElapsedTimeSnr float64
	ElapsedTimeNav float64
}

type SdrEkf struct {
	Rk1V []float64 // MAXSAT
	VarR float64
}

type SdrGui struct {
	MessageCount int
	Messages     [MAX_MESSAGES]string
}

// ...kolejne struktury będą dodawane w tym pliku...
