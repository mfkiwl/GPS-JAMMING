package main

// Constants for bit scaling (from RTKLIB)
const (
    P2_31 = 4.656612873077393E-10 // 2^-31
    P2_43 = 1.136868377216160E-13 // 2^-43
    P2_55 = 2.775557561562891E-17 // 2^-55
    P2_5  = 0.03125                // 2^-5
    P2_19 = 1.907348632812500E-06  // 2^-19
    P2_29 = 1.862645149230957E-09  // 2^-29
    P2_33 = 1.164153218269348E-10  // 2^-33
    SC2RAD = 3.1415926535898       // semi-circle to radian (IS-GPS)
)

// Adjust GPS week (from RTKLIB)
func adjgpsweek(week int) int {
    // Use 2009/12/1 if time is earlier than 2009/12/1
    w := 1560 // GPS week for 2009/12/1
    return week + (w-week+512)/1024*1024
}

// Convert GPS week and seconds to time (simplified)
type GpsTime struct {
    Week int
    Sec  float64
}
func gpst2time(week int, sec float64) GpsTime {
    // This is a simplified version; original uses epoch2time(gpst0)
    return GpsTime{Week: week, Sec: sec}
}

// Define EphGps struct for GPS ephemeris fields (minimal, expand as needed)
type EphGps struct {
    code   uint32
    sva    uint32
    svh    uint32
    iodc   uint32
    flag   uint32
    tgd    [2]float64
    f2     float64
    f1     float64
    f0     float64
    week   int
    ttr    GpsTime
    toc    GpsTime
    iode   uint32
    crs    float64
    deln   float64
    M0     float64
    cuc    float64
    e      float64
    cus    float64
    toes   float64
    fit    uint32
    A      float64
    cic    float64
    OMG0   float64
    cis    float64
    i0     float64
    crc    float64
    omg    float64
    OMGd   float64
    idot   float64
    // Add other fields as needed for full GPS ephemeris
}

func decode_subfrm1(buff []uint8, eph *SdrEph) {
    eph.TowGpst = float64(getbitu(buff, 30, 17)) * 6.0
    week := getbitu(buff, 60, 10) + 1024
    // The following assumes Eph is a struct with these fields. Type assertion may be needed if Eph is interface{}.
    ephStruct, ok := eph.Eph.(*EphGps)
    if !ok {
        return // or handle error
    }
    ephStruct.code = getbitu(buff, 70, 2)
    ephStruct.sva = getbitu(buff, 72, 4)
    ephStruct.svh = getbitu(buff, 76, 6)
    ephStruct.iodc = getbitu2(buff, 82, 2, 210, 8)
    ephStruct.flag = getbitu(buff, 90, 1)
    ephStruct.tgd[0] = float64(getbits(buff, 196, 8)) * P2_31
    toc := float64(getbitu(buff, 218, 16)) * 16.0
    ephStruct.f2 = float64(getbits(buff, 240, 8)) * P2_55
    ephStruct.f1 = float64(getbits(buff, 248, 16)) * P2_43
    ephStruct.f0 = float64(getbits(buff, 270, 22)) * P2_31
    ephStruct.week = adjgpsweek(int(week))
    eph.WeekGpst = ephStruct.week
    ephStruct.ttr = gpst2time(ephStruct.week, eph.TowGpst)
    ephStruct.toc = gpst2time(ephStruct.week, toc)
    eph.Cnt++
}

func decode_subfrm2(buff []uint8, eph *SdrEph) {
    ephStruct, ok := eph.Eph.(*EphGps)
    if !ok {
        return // or handle error
    }
    oldiode := ephStruct.iode
    eph.TowGpst = float64(getbitu(buff, 30, 17)) * 6.0
    ephStruct.iode = getbitu(buff, 60, 8)
    ephStruct.crs = float64(getbits(buff, 68, 16)) * P2_5
    ephStruct.deln = float64(getbits(buff, 90, 16)) * P2_43 * SC2RAD
    ephStruct.M0 = float64(getbits2(buff, 106, 8, 120, 24)) * P2_31 * SC2RAD
    ephStruct.cuc = float64(getbits(buff, 150, 16)) * P2_29
    ephStruct.e = float64(getbitu2(buff, 166, 8, 180, 24)) * P2_33
    ephStruct.cus = float64(getbits(buff, 210, 16)) * P2_29
    sqrtA := float64(getbitu2(buff, 226, 8, 240, 24)) * P2_19
    ephStruct.toes = float64(getbitu(buff, 270, 16)) * 16.0
    ephStruct.fit = getbitu(buff, 286, 1)
    ephStruct.A = sqrtA * sqrtA
    if oldiode-ephStruct.iode != 0 {
        eph.Update = ON
    }
    eph.Cnt++
}

func decode_subfrm3(buff []uint8, eph *SdrEph) {
    ephStruct, ok := eph.Eph.(*EphGps)
    if !ok {
        return // or handle error
    }
    oldiode := ephStruct.iode
    eph.TowGpst = float64(getbitu(buff, 30, 17)) * 6.0
    ephStruct.cic = float64(getbits(buff, 60, 16)) * P2_29
    ephStruct.OMG0 = float64(getbits2(buff, 76, 8, 90, 24)) * P2_31 * SC2RAD
    ephStruct.cis = float64(getbits(buff, 120, 16)) * P2_29
    ephStruct.i0 = float64(getbits2(buff, 136, 8, 150, 24)) * P2_31 * SC2RAD
    ephStruct.crc = float64(getbits(buff, 180, 16)) * P2_5
    ephStruct.omg = float64(getbits2(buff, 196, 8, 210, 24)) * P2_31 * SC2RAD
    ephStruct.OMGd = float64(getbits(buff, 240, 24)) * P2_43 * SC2RAD
    ephStruct.iode = getbitu(buff, 270, 8)
    ephStruct.idot = float64(getbits(buff, 278, 14)) * P2_43 * SC2RAD
    if oldiode-ephStruct.iode != 0 {
        eph.Update = ON
    }
    eph.Cnt++
}

func decode_subfrm4(buff []uint8, eph *SdrEph) {
    eph.TowGpst = float64(getbitu(buff, 30, 17)) * 6.0
}

func decode_subfrm5(buff []uint8, eph *SdrEph) {
    eph.TowGpst = float64(getbitu(buff, 30, 17)) * 6.0
}

func decode_frame_l1ca(buff []uint8, eph *SdrEph) int {
    id := getbitu(buff, 49, 3)
    switch id {
    case 1:
        decode_subfrm1(buff, eph)
    case 2:
        decode_subfrm2(buff, eph)
    case 3:
        decode_subfrm3(buff, eph)
    case 4:
        decode_subfrm4(buff, eph)
    case 5:
        decode_subfrm5(buff, eph)
    }
    return int(id)
}

func paritycheck_l1ca(bits []int) int {
    stat := 0
    pbits := make([]int, 6)
    pbits[0] = bits[0] * bits[2] * bits[3] * bits[4] * bits[6] * bits[7] * bits[11] * bits[12] * bits[13] * bits[14] * bits[15] * bits[18] * bits[19] * bits[21] * bits[24]
    pbits[1] = bits[1] * bits[3] * bits[4] * bits[5] * bits[7] * bits[8] * bits[12] * bits[13] * bits[14] * bits[15] * bits[16] * bits[19] * bits[20] * bits[22] * bits[25]
    pbits[2] = bits[0] * bits[2] * bits[4] * bits[5] * bits[6] * bits[8] * bits[9] * bits[13] * bits[14] * bits[15] * bits[16] * bits[17] * bits[20] * bits[21] * bits[23]
    pbits[3] = bits[1] * bits[3] * bits[5] * bits[6] * bits[7] * bits[9] * bits[10] * bits[14] * bits[15] * bits[16] * bits[17] * bits[18] * bits[21] * bits[22] * bits[24]
    pbits[4] = bits[1] * bits[2] * bits[4] * bits[6] * bits[7] * bits[8] * bits[10] * bits[11] * bits[15] * bits[16] * bits[17] * bits[18] * bits[19] * bits[22] * bits[23] * bits[25]
    pbits[5] = bits[0] * bits[4] * bits[6] * bits[7] * bits[9] * bits[10] * bits[11] * bits[12] * bits[14] * bits[16] * bits[20] * bits[23] * bits[24] * bits[25]
    for i := 0; i < 6; i++ {
        stat += pbits[i] - bits[26+i]
    }
    if stat == 0 {
        return 1
    }
    return 0
}

func decode_l1ca(nav *SdrNav) int {
    id := 0
    bin := make([]uint8, 38)
    for i := 0; i < 10; i++ {
        if nav.FBitsDec[i*30+1] == -1 {
            for j := 2; j < 26; j++ {
                nav.FBitsDec[i*30+j] *= -1
            }
        }
    }
    bits2byte(nav.FBitsDec[nav.AddFlen:], nav.Flen, 38, 0, bin)
    id = decode_frame_l1ca(bin, &nav.SdrEph)
    return id
}
