package main

import (
    "math"
)

// Assume all required types, constants, and helper functions are defined elsewhere
// Extract unsigned bits from byte data
func getbitu(buff []uint8, pos, len int) uint32 {
    var bits uint32 = 0
    for i := pos; i < pos+len; i++ {
        bits = (bits << 1) + uint32((buff[i/8]>>(7-i%8))&1)
    }
    return bits
}

// Extract signed bits from byte data
func getbits(buff []uint8, pos, len int) int32 {
    bits := getbitu(buff, pos, len)
    if len <= 0 || 32 <= len || (bits&(1<<(len-1))) == 0 {
        return int32(bits)
    }
    return int32(bits) | (int32(-1) << len) // sign extension
}

func sdrnavigation(sdr *SdrCh, buffloc, cnt uint64) {
    // sfn := 0 // unused variable, commented out
    sdr.Nav.BitI = int(cnt) % sdr.Nav.Rate
    sdr.Nav.OCodeI = sdr.Nav.BitI - sdr.Nav.SyncI - 1
    if sdr.Nav.OCodeI < 0 {
        sdr.Nav.OCodeI += sdr.Nav.Rate
    }
    // Navigation bit synchronization
    if sdr.Nav.Rate == 1 && cnt > uint64(2000/(sdr.CTime*1000)) {
        sdr.Nav.SyncI = 0
        sdr.Nav.FlagSync = ON
    }
    if sdr.Nav.FlagSync == 0 && cnt > uint64(2000/(sdr.CTime*1000)) {
        sdr.Nav.FlagSync = checksync(sdr.Trk.II[0], sdr.Trk.OldI[0], &sdr.Nav)
    }
    if sdr.Nav.FlagSync != 0 {
        if checkbit(sdr.Trk.II[0], sdr.Trk.LoopMs, &sdr.Nav) == OFF {
            // Navigation sync error
        }
        if sdr.Nav.SwSync != 0 {
            if sdr.Nav.FlagTow == 0 {
                predecodefec(&sdr.Nav)
            }
            if sdr.Nav.FlagTow == 0 {
                sdr.Nav.FlagSyncF = findpreamble(&sdr.Nav)
            }
            if sdr.Nav.FlagSyncF != 0 && sdr.Nav.FlagTow == 0 {
                sdr.Nav.FirstSf = buffloc
                sdr.Nav.FirstSfCnt = cnt
                sdr.Nav.FlagTow = ON
            }
        }
        if sdr.Nav.FlagTow != 0 && sdr.Nav.SwSync != 0 {
            if int(cnt-sdr.Nav.FirstSfCnt)%sdr.Nav.Update == 0 {
                predecodefec(&sdr.Nav)
                // sfn = decodenav(&sdr.Nav)
                if sdr.Nav.SdrEph.TowGpst == 0 {
                    sdr.Nav.FlagSyncF = OFF
                    sdr.Nav.FlagTow = OFF
                } else if cnt-sdr.Nav.FirstSfCnt == 0 {
                    sdr.Nav.FlagDec = ON
                    // SdrEph.Eph is interface{}, so we need a type assertion if Sat is needed
                    if eph, ok := sdr.Nav.SdrEph.Eph.(interface{ SetSat(int) }); ok {
                        eph.SetSat(sdr.Sat)
                    }
                    sdr.Nav.FirstSfTow = sdr.Nav.SdrEph.TowGpst
                }
            }
        }
    }
}

func getbitu2(buff []uint8, p1, l1, p2, l2 int) uint32 {
    return (getbitu(buff, p1, l1) << l2) + getbitu(buff, p2, l2)
}
func getbits2(buff []uint8, p1, l1, p2, l2 int) int32 {
    if getbitu(buff, p1, 1) != 0 {
        return int32((uint32(getbits(buff, p1, l1)) << l2) + getbitu(buff, p2, l2))
    }
    return int32(getbitu2(buff, p1, l1, p2, l2))
}
func getbitu3(buff []uint8, p1, l1, p2, l2, p3, l3 int) uint32 {
    return (getbitu(buff, p1, l1) << (l2 + l3)) + (getbitu(buff, p2, l2) << l3) + getbitu(buff, p3, l3)
}
func getbits3(buff []uint8, p1, l1, p2, l2, p3, l3 int) int32 {
    if getbitu(buff, p1, 1) != 0 {
        return int32((uint32(getbits(buff, p1, l1)) << (l2 + l3)) + (getbitu(buff, p2, l2) << l3) + getbitu(buff, p3, l3))
    }
    return int32(getbitu3(buff, p1, l1, p2, l2, p3, l3))
}
func merge_two_u(a, b uint32, n int) uint32 {
    return (a << n) + b
}
func merge_two_s(a int32, b uint32, n int) int32 {
    return int32((a << n) + int32(b))
}
func bits2byte(bits []int, nbits, nbin, right int, bin []uint8) {
    bitscpy := make([]int, MAXBITS)
    rem := 8*nbin - nbits
    copy(bitscpy[right*rem:], bits[:nbits])
    for i := 0; i < nbin; i++ {
        var b uint8
        for j := 0; j < 8; j++ {
            b <<= 1
            if bitscpy[i*8+j] < 0 {
                b |= 0x01
            }
        }
        bin[i] = b
    }
}
func interleave(in []int, row, col int, out []int) {
    tmp := make([]int, row*col)
    copy(tmp, in[:row*col])
    for r := 0; r < row; r++ {
        for c := 0; c < col; c++ {
            out[r*col+c] = tmp[c*row+r]
        }
    }
}
func checksync(IP, IPold float64, nav *SdrNav) int {
    var maxi int
    if IPold*IP < 0 {
        nav.BitSync[nav.BitI]++
        maxi = maxvi(nav.BitSync, nav.Rate, -1, -1, &nav.SyncI)
        if maxi > NAVSYNCTH {
            nav.SyncI--
            if nav.SyncI < 0 {
                nav.SyncI = nav.Rate - 1
            }
            return 1
        }
    }
    return 0
}
func checkbit(IP float64, loopms int, nav *SdrNav) int {
    diffi := nav.BitI - nav.SyncI
    syncflag := ON
    polarity := 1
    nav.SwReset = OFF
    nav.SwSync = OFF
    if diffi == 1 || diffi == -nav.Rate+1 {
        nav.BitIP = IP
        nav.SwReset = ON
        nav.Cnt = 1
    } else {
        nav.BitIP += IP
        if nav.BitIP*IP < 0 {
            syncflag = OFF
        }
    }
    if nav.Cnt%loopms == 0 {
        nav.SwLoop = ON
    } else {
        nav.SwLoop = OFF
    }
    if diffi == 0 {
        if nav.FlagPol != 0 {
            polarity = -1
        } else {
            polarity = 1
        }
        if nav.BitIP < 0 {
            nav.Bit = -polarity
        } else {
            nav.Bit = polarity
        }
        shiftdata(nav.FBits, nav.FBits[1:], intSize, nav.Flen+nav.AddFlen-1)
        nav.FBits[nav.Flen+nav.AddFlen-1] = nav.Bit
        nav.SwSync = ON
    }
    nav.Cnt++
    return syncflag
}
func predecodefec(nav *SdrNav) {
    if nav.Ctype == CTYPE_L1CA {
        copy(nav.FBitsDec, nav.FBits[:nav.Flen+nav.AddFlen])
    }
    if nav.Ctype == CTYPE_L1SBAS {
        init_viterbi27_port(nav.Fec, 0)
        enc := make([]uint8, NAVFLEN_SBAS+NAVADDFLEN_SBAS)
        dec := make([]uint8, 94)
        dec2 := make([]int, NAVFLEN_SBAS/2)
        for i := 0; i < NAVFLEN_SBAS+NAVADDFLEN_SBAS; i++ {
            if nav.FBits[i] == 1 {
                enc[i] = 0
            } else {
                enc[i] = 255
            }
        }
        update_viterbi27_blk_port(nav.Fec, enc, (nav.Flen+nav.AddFlen)/2)
        chainback_viterbi27_port(nav.Fec, dec, nav.Flen/2, 0)
        for i := 0; i < 94; i++ {
            for j := 0; j < 8; j++ {
                dec2[8*i+j] = int((dec[i]<<j)&0x80) >> 7
                if dec2[8*i+j] == 0 {
                    nav.FBitsDec[8*i+j] = 1
                } else {
                    nav.FBitsDec[8*i+j] = -1
                }
                if 8*i+j == NAVFLEN_SBAS/2-1 {
                    break
                }
            }
        }
    }
}
func paritycheck(nav *SdrNav) int {
    bits := make([]int, MAXBITS)
    bin := make([]uint8, 29)
    pbin := make([]uint8, 3)
    for i := 0; i < nav.Flen+nav.AddFlen; i++ {
        bits[i] = nav.Polarity * nav.FBitsDec[i]
    }
    if nav.Ctype == CTYPE_L1CA {
        stat := 0
        for i := 0; i < 10; i++ {
            if bits[i*30+1] == -1 {
                for j := 2; j < 26; j++ {
                    bits[i*30+j] *= -1
                }
            }
            stat += paritycheck_l1ca(bits[i*30:])
        }
        if stat == 10 {
            return 1
        }
    }
    if nav.Ctype == CTYPE_L1SBAS {
        bits2byte(bits, 226, 29, 1, bin)
        bits2byte(bits[226:], 24, 3, 0, pbin)
        crc := crc24q(bin, 29)
        if crc == getbitu(pbin, 0, 24) {
            return 1
        }
    }
    return 0
}
func findpreamble(nav *SdrNav) int {
    corr := 0
        if nav.Ctype == CTYPE_L1CA {
            for i := 0; i < nav.PreLen; i++ {
                corr += nav.FBitsDec[nav.AddFlen+i] * nav.PreBits[i]
            }
        }
        if nav.Ctype == CTYPE_L1SBAS {
            for i := 0; i < nav.PreLen/2; i++ {
                corr += nav.FBitsDec[i] * nav.PreBits[0+i]
                corr += nav.FBitsDec[i+250] * nav.PreBits[8+i]
            }
        }
        if int(math.Abs(float64(corr))) == nav.PreLen {
            nav.Polarity = 1
            if corr < 0 {
            nav.Polarity = -1
        }
        if paritycheck(nav) == 1 {
            return 1
        } else {
            if nav.Ctype == CTYPE_L1SBAS {
                if nav.Polarity == 1 {
                    nav.FlagPol = ON
                }
            }
        }
    }
    return 0
}
func decodenav(nav *SdrNav) int {
    switch nav.Ctype {
        case CTYPE_L1CA:
            return decode_l1ca(nav)
        case CTYPE_L1SBAS:
            return decode_l1sbas(nav)
    default:
        return -1
    }
}

// Helper: maximum value and index (int array)
func maxvi(data []int, n, exinds, exinde int, ind *int) int {
    max, idx := MaxVI(data, exinds, exinde)
    *ind = idx
    return max
}

// Helper: shiftdata for int slices
func shiftdata(dst, src []int, size, n int) {
    copy(dst[:n], src[:n])
}

// Helper: intSize constant (size of int in bytes)
const intSize = 4 // assuming 32-bit int

// Stubs for FEC functions (should be replaced with actual implementations)
func init_viterbi27_port(fec interface{}, val int) {
    // Viterbi 27 FEC initialization (stub)
    // In real code, allocate and initialize FEC state
    // Example: fec.(*Viterbi27).Init(val)
}
func update_viterbi27_blk_port(fec interface{}, enc []uint8, n int) {
    // Viterbi 27 FEC block update (stub)
    // In real code, update FEC state with encoded block
    // Example: fec.(*Viterbi27).UpdateBlock(enc, n)
}
func chainback_viterbi27_port(fec interface{}, dec []uint8, n int, val int) {
    // Viterbi 27 FEC chainback (stub)
    // In real code, perform chainback to recover decoded bits
    // Example: fec.(*Viterbi27).Chainback(dec, n, val)
}

// CRC24Q implementation based on RTKLIB
func crc24q(buff []byte, length int) uint32 {
    var tblCRC24Q = [256]uint32{
        0x000000,0x864CFB,0x8AD50D,0x0C99F6,0x93E6E1,0x15AA1A,0x1933EC,0x9F7F17,
        0xA18139,0x27CDC2,0x2B5434,0xAD18CF,0x3267D8,0xB42B23,0xB8B2D5,0x3EFE2E,
        0xC54E89,0x430272,0x4F9B84,0xC9D77F,0x56A868,0xD0E493,0xDC7D65,0x5A319E,
        0x64CFB0,0xE2834B,0xEE1ABD,0x685646,0xF72951,0x7165AA,0x7DFC5C,0xFBB0A7,
        0x0CD1E9,0x8A9D12,0x8604E4,0x00481F,0x9F3708,0x197BF3,0x15E205,0x93AEFE,
        0xAD50D0,0x2B1C2B,0x2785DD,0xA1C926,0x3EB631,0xB8FACA,0xB4633C,0x322FC7,
        0xC99F60,0x4FD39B,0x434A6D,0xC50696,0x5A7981,0xDC357A,0xD0AC8C,0x56E077,
        0x681E59,0xEE52A2,0xE2CB54,0x6487AF,0xFBF8B8,0x7DB443,0x712DB5,0xF7614E,
        0x19A3D2,0x9FEF29,0x9376DF,0x153A24,0x8A4533,0x0C09C8,0x00903E,0x86DCC5,
        0xB822EB,0x3E6E10,0x32F7E6,0xB4BB1D,0x2BC40A,0xAD88F1,0xA11107,0x275DFC,
        0xDCED5B,0x5AA1A0,0x563856,0xD074AD,0x4F0BBA,0xC94741,0xC5DEB7,0x43924C,
        0x7D6C62,0xFB2099,0xF7B96F,0x71F594,0xEE8A83,0x68C678,0x645F8E,0xE21375,
        0x15723B,0x933EC0,0x9FA736,0x19EBCD,0x8694DA,0x00D821,0x0C41D7,0x8A0D2C,
        0xB4F302,0x32BFF9,0x3E260F,0xB86AF4,0x2715E3,0xA15918,0xADC0EE,0x2B8C15,
        0xD03CB2,0x567049,0x5AE9BF,0xDCA544,0x43DA53,0xC596A8,0xC90F5E,0x4F43A5,
        0x71BD8B,0xF7F170,0xFB6886,0x7D247D,0xE25B6A,0x641791,0x688E67,0xEEC29C,
        0x3347A4,0xB50B5F,0xB992A9,0x3FDE52,0xA0A145,0x26EDBE,0x2A7448,0xAC38B3,
        0x92C69D,0x148A66,0x181390,0x9E5F6B,0x01207C,0x876C87,0x8BF571,0x0DB98A,
        0xF6092D,0x7045D6,0x7CDC20,0xFA90DB,0x65EFCC,0xE3A337,0xEF3AC1,0x69763A,
        0x578814,0xD1C4EF,0xDD5D19,0x5B11E2,0xC46EF5,0x42220E,0x4EBBF8,0xC8F703,
        0x3F964D,0xB9DAB6,0xB54340,0x330FBB,0xAC70AC,0x2A3C57,0x26A5A1,0xA0E95A,
        0x9E1774,0x185B8F,0x14C279,0x928E82,0x0DF195,0x8BBD6E,0x872498,0x016863,
        0xFAD8C4,0x7C943F,0x700DC9,0xF64132,0x693E25,0xEF72DE,0xE3EB28,0x65A7D3,
        0x5B59FD,0xDD1506,0xD18CF0,0x57C00B,0xC8BF1C,0x4EF3E7,0x426A11,0xC426EA,
        0x2AE476,0xACA88D,0xA0317B,0x267D80,0xB90297,0x3F4E6C,0x33D79A,0xB59B61,
        0x8B654F,0x0D29B4,0x01B042,0x87FCB9,0x1883AE,0x9ECF55,0x9256A3,0x141A58,
        0xEFAAFF,0x69E604,0x657FF2,0xE33309,0x7C4C1E,0xFA00E5,0xF69913,0x70D5E8,
        0x4E2BC6,0xC8673D,0xC4FECB,0x42B230,0xDDCD27,0x5B81DC,0x57182A,0xD154D1,
        0x26359F,0xA07964,0xACE092,0x2AAC69,0xB5D37E,0x339F85,0x3F0673,0xB94A88,
        0x87B4A6,0x01F85D,0x0D61AB,0x8B2D50,0x145247,0x921EBC,0x9E874A,0x18CBB1,
        0xE37B16,0x6537ED,0x69AE1B,0xEFE2E0,0x709DF7,0xF6D10C,0xFA48FA,0x7C0401,
        0x42FA2F,0xC4B6D4,0xC82F22,0x4E63D9,0xD11CCE,0x575035,0x5BC9C3,0xDD8538,
    }
    crc := uint32(0)
    for i := 0; i < length; i++ {
        crc = ((crc << 8) & 0xFFFFFF) ^ tblCRC24Q[(crc>>16)^uint32(buff[i])]
    }
    return crc
}