package main

// import "fmt"  // Disabled for clean output

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

func time2gpst(t GpsTime) float64 {
    // Convert to GPS seconds from epoch
    return float64(t.Week) * 604800.0 + t.Sec
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
    // Direct access to Eph structure fields
    eph.Eph.Code = int(getbitu(buff, 70, 2))
    eph.Eph.Sva = int(getbitu(buff, 72, 4))
    eph.Eph.Svh = int(getbitu(buff, 76, 6))
    eph.Eph.Iodc = int(getbitu2(buff, 82, 2, 210, 8))
    eph.Eph.Flag = int(getbitu(buff, 90, 1))
    eph.Eph.Tgd[0] = float64(getbits(buff, 196, 8)) * P2_31
    toc := float64(getbitu(buff, 218, 16)) * 16.0
    eph.Eph.F2 = float64(getbits(buff, 240, 8)) * P2_55
    eph.Eph.F1 = float64(getbits(buff, 248, 16)) * P2_43
    eph.Eph.F0 = float64(getbits(buff, 270, 22)) * P2_31
    eph.Eph.Week = adjgpsweek(int(week))
    eph.WeekGpst = eph.Eph.Week
    ttr := gpst2time(eph.Eph.Week, eph.TowGpst)
    toc_time := gpst2time(eph.Eph.Week, toc)
    eph.Eph.Ttr = time2gpst(ttr)
    eph.Eph.Toc = time2gpst(toc_time)
    eph.Cnt++
}

func decode_subfrm2(buff []uint8, eph *SdrEph) {
    oldiode := eph.Eph.Iode
    eph.TowGpst = float64(getbitu(buff, 30, 17)) * 6.0
    eph.Eph.Iode = int(getbitu(buff, 60, 8))
    eph.Eph.Crs = float64(getbits(buff, 68, 16)) * P2_5
    eph.Eph.Deln = float64(getbits(buff, 90, 16)) * P2_43 * SC2RAD
    eph.Eph.M0 = float64(getbits2(buff, 106, 8, 120, 24)) * P2_31 * SC2RAD
    eph.Eph.Cuc = float64(getbits(buff, 150, 16)) * P2_29
    eph.Eph.E = float64(getbitu2(buff, 166, 8, 180, 24)) * P2_33
    eph.Eph.Cus = float64(getbits(buff, 210, 16)) * P2_29
    sqrtA := float64(getbitu2(buff, 226, 8, 240, 24)) * P2_19
    eph.Eph.Toes = float64(getbitu(buff, 270, 16)) * 16.0
    eph.Eph.Fit = float64(getbitu(buff, 286, 1))
    eph.Eph.A = sqrtA * sqrtA
    if oldiode-eph.Eph.Iode != 0 {
        eph.Update = ON
    }
    eph.Cnt++
}

func decode_subfrm3(buff []uint8, eph *SdrEph) {
    oldiode := eph.Eph.Iode
    eph.TowGpst = float64(getbitu(buff, 30, 17)) * 6.0
    eph.Eph.Cic = float64(getbits(buff, 60, 16)) * P2_29
    eph.Eph.OMG0 = float64(getbits2(buff, 76, 8, 90, 24)) * P2_31 * SC2RAD
    eph.Eph.Cis = float64(getbits(buff, 120, 16)) * P2_29
    eph.Eph.I0 = float64(getbits2(buff, 136, 8, 150, 24)) * P2_31 * SC2RAD
    eph.Eph.Crc = float64(getbits(buff, 180, 16)) * P2_5
    eph.Eph.Omg = float64(getbits2(buff, 196, 8, 210, 24)) * P2_31 * SC2RAD
    eph.Eph.OMGd = float64(getbits(buff, 240, 24)) * P2_43 * SC2RAD
    eph.Eph.Iode = int(getbitu(buff, 270, 8))
    eph.Eph.Idot = float64(getbits(buff, 278, 14)) * P2_43 * SC2RAD
    if oldiode-eph.Eph.Iode != 0 {
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
    // fmt.Printf("DEBUG decode_frame_l1ca: subframe id=%d\n", id)  // Disabled for cleaner output
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
    
    // Disable debug for cleaner output
    // fmt.Printf("DEBUG decode_l1ca: Flen=%d, AddFlen=%d, TotalBitCollections=%d\n", 
    //     nav.Flen, nav.AddFlen, nav.TotalBitCollections)
    
    // Disable debug for cleaner output
    // if nav.Flen > 0 {
    //     fmt.Printf("DEBUG decode_l1ca: First few FBits values: ")
    //     maxShow := nav.Flen
    //     if maxShow > 10 {
    //         maxShow = 10
    //     }
    //     for i := 0; i < maxShow; i++ {
    //         fmt.Printf("%d ", nav.FBits[i])
    //     }
    //     fmt.Printf("\n")
    //     
    //     // Check middle of buffer too
    //     midStart := nav.Flen / 2
    //     fmt.Printf("DEBUG decode_l1ca: Middle FBits values [%d-%d]: ", midStart, midStart+9)
    //     for i := midStart; i < midStart+10 && i < nav.Flen; i++ {
    //         fmt.Printf("%d ", nav.FBits[i])
    //     }
    //     fmt.Printf("\n")
    //     
    //     fmt.Printf("DEBUG decode_l1ca: First few FBitsDec values: ")
    //     for i := 0; i < maxShow; i++ {
    //         fmt.Printf("%d ", nav.FBitsDec[i])
    //     }
    //     fmt.Printf("\n")
    // }
    
    for i := 0; i < 10; i++ {
        if nav.FBitsDec[i*30+1] == -1 {
            for j := 2; j < 26; j++ {
                nav.FBitsDec[i*30+j] *= -1
            }
        }
    }
    bits2byte(nav.FBitsDec[nav.AddFlen:], nav.Flen, 38, 0, bin)
    id = decode_frame_l1ca(bin, &nav.SdrEph)
    
    // Disable debug for cleaner output  
    // fmt.Printf("DEBUG decode_l1ca: decode_frame_l1ca returned id=%d\n", id)
    return id
}
