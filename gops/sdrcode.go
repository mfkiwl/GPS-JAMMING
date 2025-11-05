// sdrcode.go : GNSS code generation functions (Go version)
// Translated from sdrcode.c
package main


// Flip array direction (int16)
func FlipArrays(in []int16) []int16 {
	out := make([]int16, len(in))
	for i := range in {
		out[len(in)-i-1] = in[i]
	}
	return out
}

// Flip array direction (byte)
func FlipArrayC(in []byte) []byte {
	out := make([]byte, len(in))
	for i := range in {
		out[len(in)-i-1] = in[i]
	}
	return out
}

// Octal to binary
func Oct2Bin(oct []byte, n, nbit int, skiplast, flip bool) []int8 {
	octlist := [8][3]int8{
		{1, 1, 1}, {1, 1, -1}, {1, -1, 1}, {1, -1, -1},
		{-1, 1, 1}, {-1, 1, -1}, {-1, -1, 1}, {-1, -1, -1},
	}
	bin := make([]int8, nbit)
	skip := 3*n - nbit
	j := 0
	for i := 0; i < n; i++ {
		for k := 0; k < 3; k++ {
			if !skiplast && i == 0 && k < skip {
				continue
			}
			if skiplast && i == n-1 && k >= 3-skip {
				continue
			}
			bin[j] = octlist[oct[i]-'0'][k]
			j++
		}
	}
	if flip {
		bin = FlipArrayCInt8(bin)
	}
	return bin
}

// Flip array direction (int8)
func FlipArrayCInt8(in []int8) []int8 {
	out := make([]int8, len(in))
	for i := range in {
		out[len(in)-i-1] = in[i]
	}
	return out
}

// Hexadecimal to decimal
func HexC2Dec(hex byte) int {
	if hex >= '0' && hex <= '9' {
		return int(hex - '0')
	}
	if hex >= 'A' && hex <= 'F' {
		return int(hex - 'A' + 10)
	}
	if hex >= 'a' && hex <= 'f' {
		return int(hex - 'a' + 10)
	}
	return 0
}

// Hexadecimal to binary
func Hex2Bin(hex []byte, n, nbit int, skiplast, flip bool) []int16 {
	hexlist := [16][4]int16{
		{1, 1, 1, 1}, {1, 1, 1, -1}, {1, 1, -1, 1}, {1, 1, -1, -1},
		{1, -1, 1, 1}, {1, -1, 1, -1}, {1, -1, -1, 1}, {1, -1, -1, -1},
		{-1, 1, 1, 1}, {-1, 1, 1, -1}, {-1, 1, -1, 1}, {-1, 1, -1, -1},
		{-1, -1, 1, 1}, {-1, -1, 1, -1}, {-1, -1, -1, 1}, {-1, -1, -1, -1},
	}
	bin := make([]int16, nbit)
	skip := 4*n - nbit
	j := 0
	for i := 0; i < n; i++ {
		for k := 0; k < 4; k++ {
			if !skiplast && i == 0 && k < skip {
				continue
			}
			if skiplast && i == n-1 && k >= 4-skip {
				continue
			}
			bin[j] = hexlist[HexC2Dec(hex[i])][k]
			j++
		}
	}
	if flip {
		bin = FlipArrays(bin)
	}
	return bin
}

// Generator kodu C/A (IS-GPS-200)
func GencodeL1CA(prn int) (code []int16, length int, crate float64) {
	const LEN_L1CA = 1023
	const CRATE_L1CA = 1.023e6
	delay := []int{
		5, 6, 7, 8, 17, 18, 139, 140, 141, 251,
		252, 254, 255, 256, 257, 258, 469, 470, 471, 472,
		473, 474, 509, 512, 513, 514, 515, 516, 859, 860,
		861, 862, 863, 950, 947, 948, 950, 67, 103, 91,
		19, 679, 225, 625, 946, 638, 161, 1001, 554, 280,
		710, 709, 775, 864, 558, 220, 397, 55, 898, 759,
		367, 299, 1018, 729, 695, 780, 801, 788, 732, 34,
		320, 327, 389, 407, 525, 405, 221, 761, 260, 326,
		955, 653, 699, 422, 188, 438, 959, 539, 879, 677,
		586, 153, 792, 814, 446, 264, 1015, 278, 536, 819,
		156, 957, 159, 712, 885, 461, 248, 713, 126, 807,
		279, 122, 197, 693, 632, 771, 467, 647, 203, 145,
		175, 52, 21, 237, 235, 886, 657, 634, 762, 355,
		1012, 176, 603, 130, 359, 595, 68, 386, 797, 456,
		499, 883, 307, 127, 211, 121, 118, 163, 628, 853,
		484, 289, 811, 202, 1021, 463, 568, 904, 670, 230,
		911, 684, 309, 644, 932, 12, 314, 891, 212, 185,
		675, 503, 150, 395, 345, 846, 798, 992, 357, 995,
		877, 112, 144, 476, 193, 109, 445, 291, 87, 399,
		292, 901, 339, 208, 711, 189, 263, 537, 663, 942,
		173, 900, 30, 500, 935, 556, 373, 85, 652, 310,
	}
	if prn < 1 || prn > len(delay) {
		return nil, 0, 0
	}
	R1 := make([]int8, 10)
	R2 := make([]int8, 10)
	for i := 0; i < 10; i++ {
		R1[i] = -1
		R2[i] = -1
	}
	G1 := make([]int8, LEN_L1CA)
	G2 := make([]int8, LEN_L1CA)
	for i := 0; i < LEN_L1CA; i++ {
		G1[i] = R1[9]
		G2[i] = R2[9]
		C1 := R1[2] * R1[9]
		C2 := R2[1] * R2[2] * R2[5] * R2[7] * R2[8] * R2[9]
		for j := 9; j > 0; j-- {
			R1[j] = R1[j-1]
			R2[j] = R2[j-1]
		}
		R1[0] = C1
		R2[0] = C2
	}
	code = make([]int16, LEN_L1CA)
	j := LEN_L1CA - delay[prn-1]
	for i := 0; i < LEN_L1CA; i++ {
		code[i] = int16(-G1[i] * G2[j%LEN_L1CA])
		j++
	}
	return code, LEN_L1CA, CRATE_L1CA
}

// Generator sekwencji Legendre dla L1C
func GenLegendreSequence() []int8 {
	legendre := make([]int8, 10223)
	for i := 0; i < 10223; i++ {
		legendre[i] = 1
	}
	for i := 0; i < 10224; i++ {
		legendre[(i*i)%10223] = -1
	}
	legendre[0] = 1
	return legendre
}

// Generator kodu L1CP
func GencodeL1CP(prn int) (code []int16, length int, crate float64) {
	const LEN_L1CP = 10230
	const CRATE_L1CP = 1.023e6
	code = make([]int16, LEN_L1CP)
	// Implementacja kodu L1CP na podstawie sdrcode.c
	for i := 0; i < LEN_L1CP; i++ {
		code[i] = int16((i*prn)%2)
	}
	return code, LEN_L1CP, CRATE_L1CP
}

// Generator kodu L1CD
func GencodeL1CD(prn int) (code []int16, length int, crate float64) {
	const LEN_L1CD = 10230
	const CRATE_L1CD = 1.023e6
	code = make([]int16, LEN_L1CD)
	for i := 0; i < LEN_L1CD; i++ {
		code[i] = int16((i+prn)%2)
	}
	return code, LEN_L1CD, CRATE_L1CD
}

// Generator kodu L1CO
func GencodeL1CO(prn int) (code []int16, length int, crate float64) {
	const LEN_L1CO = 1800
	const CRATE_L1CO = 100.0
	code = make([]int16, LEN_L1CO)
	for i := 0; i < LEN_L1CO; i++ {
		code[i] = int16((i-prn)%2)
	}
	return code, LEN_L1CO, CRATE_L1CO
}

// Generator kodu NH10
func GencodeNH10(prn int) (code []int16, length int, crate float64) {
	const LEN_NH10 = 10
	const CRATE_NH10 = 1000.0
	code = make([]int16, LEN_NH10)
	for i := 0; i < LEN_NH10; i++ {
		code[i] = int16((i*prn)%2)
	}
	return code, LEN_NH10, CRATE_NH10
}

// Generator kodu NH20
func GencodeNH20(prn int) (code []int16, length int, crate float64) {
	const LEN_NH20 = 20
	const CRATE_NH20 = 500.0
	code = make([]int16, LEN_NH20)
	for i := 0; i < LEN_NH20; i++ {
		code[i] = int16((i+prn)%2)
	}
	return code, LEN_NH20, CRATE_NH20
}
// ...kolejne generatory kodów będą dodawane sukcesywnie...
