// sdrcmn.go : SDR common functions (Go version)
// Translated from sdrcmn.c
package main

import (
	"math"
	"path/filepath"
	"time"
	"fmt"

 	"gonum.org/v1/gonum/dsp/fourier"
)

func Log2(x float64) float64 {
	return math.Log(x) / math.Log(2.0)
}

func CalcFFTNum(x float64, next int) int {
	nn := int(Log2(x)+0.5) + next
	return int(math.Pow(2.0, float64(nn)))
}

func MeanVD(data []float64, n int, exinds, exinde int) float64 {
	if n <= 0 || n > len(data) {
		n = len(data)
	}
	ne := 0
	mean := 0.0
	for i := 0; i < n; i++ {
		if (exinds <= exinde && (i < exinds || i > exinde)) ||
			(exinds > exinde && (i < exinds && i > exinde)) {
			mean += data[i]
		} else {
			ne++
		}
	}
	return mean / float64(n-ne)
}

func MaxVI(data []int, exinds, exinde int) (max int, ind int) {
	max = data[0]
	ind = 0
	for i := 0; i < len(data); i++ {
		// exinds=exinde=-1 means use all data (no exclusions)
		shouldCheck := true
		if exinds != -1 || exinde != -1 {
			if exinds <= exinde {
				// Normal range exclusion: exclude indices between exinds and exinde
				shouldCheck = (i < exinds || i > exinde)
			} else {
				// Wraparound range exclusion
				shouldCheck = (i < exinds && i > exinde)
			}
		}
		if shouldCheck && max < data[i] {
			max = data[i]
			ind = i
		}
	}
	return
}

func MaxVF(data []float32, exinds, exinde int) (max float32, ind int) {
	max = data[0]
	ind = 0
	for i := 1; i < len(data); i++ {
		if (exinds <= exinde && (i < exinds || i > exinde)) ||
			(exinds > exinde && (i < exinds && i > exinde)) {
			if max < data[i] {
				max = data[i]
				ind = i
			}
		}
	}
	return
}

func MaxVD(data []float64, n int, exinds, exinde int) (max float64, ind int) {
	if n <= 0 || n > len(data) {
		n = len(data)
	}
	max = data[0]
	ind = 0
	for i := 1; i < n; i++ {
		if (exinds <= exinde && (i < exinds || i > exinde)) ||
			(exinds > exinde && (i < exinds && i > exinde)) {
			if max < data[i] {
				max = data[i]
				ind = i
			}
		}
	}
	return
}

func MulVCS(data1 []byte, data2 []int16, out []int16) {
	n := len(data1)
	for i := 0; i < n; i++ {
		out[i] = int16(data1[i]) * data2[i]
	}
}

func SumVF(data1, data2, out []float32) {
	n := len(data1)
	for i := 0; i < n; i++ {
		out[i] = data1[i] + data2[i]
	}
}

func SumVD(data1, data2, out []float64) {
	n := len(data1)
	for i := 0; i < n; i++ {
		out[i] = data1[i] + data2[i]
	}
}

func Interp1(x, y []float64, t float64) float64 {
	n := len(x)
	if n < 1 {
		return 0.0
	}
	if n == 1 {
		return y[0]
	}
	if n == 2 {
		return (y[0]*(t-x[1]) - y[1]*(t-x[0])) / (x[0]-x[1])
	}
	// Kopiowanie i ewentualne odwrócenie tablicy
	xx := make([]float64, n)
	yy := make([]float64, n)
	if x[0] > x[n-1] {
		for j, k := n-1, 0; j >= 0; j, k = j-1, k+1 {
			xx[k] = x[j]
			yy[k] = y[j]
		}
	} else {
		copy(xx, x)
		copy(yy, y)
	}
	var k, m int
	if t <= xx[1] {
		k = 0
		m = 2
	} else if t >= xx[n-2] {
		k = n - 3
		m = n - 1
	} else {
		k = 1
		m = n
		for m-k != 1 {
			i := (k + m) / 2
			if t < xx[i-1] {
				m = i
			} else {
				k = i
			}
		}
		k = k - 1
		m = m - 1
		if math.Abs(t-xx[k]) < math.Abs(t-xx[m]) {
			k = k - 1
		} else {
			m = m + 1
		}
	}
	z := 0.0
	for i := k; i <= m; i++ {
		s := 1.0
		for j := k; j <= m; j++ {
			if j != i {
				s *= (t - xx[j]) / (xx[i] - xx[j])
			}
		}
		z += s * yy[i]
	}
	return z
}

func UInt64ToDouble(data []uint64, base uint64, out []float64) {
	n := len(data)
	for i := 0; i < n; i++ {
		out[i] = float64(data[i] - base)
	}
}

func Ind2Sub(ind, nx, ny int) (subx, suby int) {
	subx = ind % nx
	suby = ny * ind / (nx * ny)
	return
}

func ShiftData(dst, src []byte) {
	copy(dst, src)
}

// TickGetUS: Zwraca czas w mikrosekundach od uruchomienia programu
// W Go można użyć time.Now().UnixMicro()
func TickGetUS() uint64 {
	return uint64(time.Now().UnixMicro())
}

// GetFullPath: Zwraca absolutną ścieżkę pliku
func GetFullPath(relpath string) (string, error) {
	return filepath.Abs(relpath)
}

// Resample code (dokładnie jak w C) z debuggingiem
func ResCode(code []int16, len int, coff float64, smax int, ci float64, n int, rcode []int16) float64 {
	// Simplified debug for first few calls only
	if len < 50 {
		fmt.Printf("ResCode: len=%d, n=%d\n", len, n)
	}
	
	coff -= float64(smax) * ci
	coff -= math.Floor(coff/float64(len)) * float64(len)
	
	// Safety check: nie pisz poza bounds
	maxElements := min(n+2*smax, len_of_slice(rcode))
	
	for i := 0; i < maxElements; i++ {
		if coff >= float64(len) {
			coff -= float64(len)
		}
		idx := int(coff)
		rcode[i] = code[idx]
		coff += ci
	}
	
	return coff - float64(smax)*ci
}

func len_of_slice(slice []int16) int {
	return len(slice)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Mix local carrier (dokładnie zgodnie z kodem C)
func MixCarr(data []byte, dtype int, ti float64, n int, freq, phi0 float64, II, QQ []int16) float64 {
	DPI := 2.0 * math.Pi
	CDIV := 16        // SSE2 version uses 16, not 32
	CMASK := 0x0F     // Mask for 16 (0x0F instead of 0x1F)
	CSCALE := 1.0 / 32.0
	
	// SSE2 version uses int8 table, not int16
	cost := make([]int8, CDIV)
	sint := make([]int8, CDIV)
	for i := 0; i < CDIV; i++ {
		cost[i] = int8(math.Floor(math.Cos(DPI/float64(CDIV)*float64(i))/CSCALE + 0.5))
		sint[i] = int8(math.Floor(math.Sin(DPI/float64(CDIV)*float64(i))/CSCALE + 0.5))
	}
	
	phi := phi0 * float64(CDIV) / DPI
	ps := freq * float64(CDIV) * ti // phase step
	
	if dtype == DTYPEIQ { // complex
		for i := 0; i < n; i++ {
			index := int(phi) & CMASK
			// Treat bytes as signed chars like C code does
			dataI := int8(data[2*i])
			dataQ := int8(data[2*i+1])
			// Use int8 table values like SSE2 version
			II[i] = int16(cost[index])*int16(dataI) - int16(sint[index])*int16(dataQ)
			QQ[i] = int16(sint[index])*int16(dataI) + int16(cost[index])*int16(dataQ)
			phi += ps
		}
	} else if dtype == DTYPEI { // real
		for i := 0; i < n; i++ {
			index := int(phi) & CMASK
			// Treat bytes as signed chars like C code does
			dataVal := int8(data[i])
			// Use int8 table values like SSE2 version
			II[i] = int16(cost[index]) * int16(dataVal)
			QQ[i] = int16(sint[index]) * int16(dataVal)
			phi += ps
		}
	}
	
	prem := phi * DPI / float64(CDIV)
	for prem > DPI {
		prem -= DPI
	}
	return prem
}

// Leap second table (rok, miesiąc, dzień, liczba sekund)
var leapTable = []struct {
	year, month, day, leap int
}{
	{1981, 6, 30, 1}, {1982, 6, 30, 1}, {1983, 6, 30, 1}, {1985, 6, 30, 1},
	{1987, 12, 31, 1}, {1989, 12, 31, 1}, {1990, 12, 31, 1}, {1992, 6, 30, 1},
	{1993, 6, 30, 1}, {1994, 6, 30, 1}, {1995, 12, 31, 1}, {1997, 6, 30, 1},
	{1998, 12, 31, 1}, {2005, 12, 31, 1}, {2008, 12, 31, 1}, {2012, 6, 30, 1},
	{2015, 6, 30, 1}, {2016, 12, 31, 1},
}

// Funkcja obliczająca liczbę sekund przestępnych od epoki GPS
func LeapSeconds(gpsSeconds int64) int {
	gpsEpoch := int64(315964800) // GPS_EPOCH_SECONDS
	gpsTime := time.Unix(gpsSeconds+gpsEpoch, 0).UTC()
	numLeaps := 0
	for _, entry := range leapTable {
		leapTime := time.Date(entry.year, time.Month(entry.month), entry.day, 0, 0, 0, 0, time.UTC)
		if gpsTime.After(leapTime) {
			numLeaps += entry.leap
		}
	}
	return numLeaps
}

// Funkcja konwertująca GPS week i TOW na UTC
func GpsToUtc(gpsWeek int, gpsTow float64) time.Time {
	gpsSeconds := int64(gpsWeek)*604800 + int64(gpsTow)
	leaps := LeapSeconds(gpsSeconds)
	utcSeconds := gpsSeconds + 315964800 - int64(leaps)
	return time.Unix(utcSeconds, 0).UTC()
}

// Advanced DSP functions (FFT, correlator, pcorrelator)

func FFTReal(input []float64) []complex128 {
	fft := fourier.NewFFT(len(input))
	return fft.Coefficients(nil, input)
}

func FFTComplex(input []complex128) []complex128 {
	fft := fourier.NewCmplxFFT(len(input))
	return fft.Coefficients(nil, input)
}

// Example correlator using FFT (skeleton)
func CorrelatorFFT(x, y []float64) []float64 {
	// Convert to complex
	xc := make([]complex128, len(x))
	yc := make([]complex128, len(y))
	for i := range x {
		xc[i] = complex(x[i], 0)
		yc[i] = complex(y[i], 0)
	}
	// FFT
	X := FFTComplex(xc)
	Y := FFTComplex(yc)
	// Multiply in frequency domain
	R := make([]complex128, len(X))
	for i := range X {
		R[i] = X[i] * Y[i]
	}
	// Inverse FFT
	fft := fourier.NewCmplxFFT(len(R))
	corr := fft.Sequence(nil, R)
	// Take real part
	result := make([]float64, len(corr))
	for i := range corr {
		result[i] = real(corr[i])
	}
	return result
}

// CSCALE constant for carrier lookup table scale
const CSCALE = 1.0 / 32.0

// dot_23: d1={dot(a1,b1),dot(a1,b2),dot(a1,b3)}, d2={dot(a2,b1),dot(a2,b2),dot(a2,b3)}
func dot_23(a1, a2, b1, b2, b3 []float64, n int, d1, d2 []float64) {
	d1[0], d1[1], d1[2] = 0, 0, 0
	d2[0], d2[1], d2[2] = 0, 0, 0
	for i := 0; i < n; i++ {
		d1[0] += a1[i] * b1[i]
		d1[1] += a1[i] * b2[i]
		d1[2] += a1[i] * b3[i]
		d2[0] += a2[i] * b1[i]
		d2[1] += a2[i] * b2[i]
		d2[2] += a2[i] * b3[i]
	}
}

// dot_22: d1={dot(a1,b1),dot(a1,b2)}, d2={dot(a2,b1),dot(a2,b2)}
func dot_22(a1, a2, b1, b2 []float64, n int, d1, d2 []float64) {
	d1[0], d1[1] = 0, 0
	d2[0], d2[1] = 0, 0
	for i := 0; i < n; i++ {
		d1[0] += a1[i] * b1[i]
		d1[1] += a1[i] * b2[i]
		d2[0] += a2[i] * b1[i]
		d2[1] += a2[i] * b2[i]
	}
}

// dot_23_int16: d1={dot(a1,b1),dot(a1,b2),dot(a1,b3)}, d2={dot(a2,b1),dot(a2,b2),dot(a2,b3)} for int16
func dot_23_int16(a1, a2, b1, b2, b3 []int16, n int, d1, d2 []float64) {
	d1[0], d1[1], d1[2] = 0, 0, 0
	d2[0], d2[1], d2[2] = 0, 0, 0
	for i := 0; i < n; i++ {
		d1[0] += float64(a1[i]) * float64(b1[i])
		d1[1] += float64(a1[i]) * float64(b2[i])
		d1[2] += float64(a1[i]) * float64(b3[i])
		d2[0] += float64(a2[i]) * float64(b1[i])
		d2[1] += float64(a2[i]) * float64(b2[i])
		d2[2] += float64(a2[i]) * float64(b3[i])
	}
}

// dot_22_int16: d1={dot(a1,b1),dot(a1,b2)}, d2={dot(a2,b1),dot(a2,b2)} for int16
func dot_22_int16(a1, a2, b1, b2 []int16, n int, d1, d2 []float64) {
	d1[0], d1[1] = 0, 0
	d2[0], d2[1] = 0, 0
	for i := 0; i < n; i++ {
		d1[0] += float64(a1[i]) * float64(b1[i])
		d1[1] += float64(a1[i]) * float64(b2[i])
		d2[0] += float64(a2[i]) * float64(b1[i])
		d2[1] += float64(a2[i]) * float64(b2[i])
	}
}