package main

// Assume all required types, constants, and helper functions are defined elsewhere

const (
    OEMSYNC1   = 0xAA
    OEMSYNC2   = 0x44
    OEMSYNC3   = 0x12
    OEMHLEN    = 28
    OEMSBASLEN = 48
    ID_RAWSBASFRAME = 973
)

func setU2(p []uint8, u uint16) {
    pp := make([]uint8, 2)
    setbitu(pp, 0, 16, uint32(u))
    p[0] = pp[1]
    p[1] = pp[0]
}
func setU4(p []uint8, u uint32) {
    pp := make([]uint8, 4)
    setbitu(pp, 0, 32, u)
    p[0] = pp[3]
    p[1] = pp[2]
    p[2] = pp[1]
    p[3] = pp[0]
}
func gen_novatel_sbasmsg(sbas *SdrSbas) {
    for i := range sbas.NovatelMsg {
        sbas.NovatelMsg[i] = 0
    }
    sbas.NovatelMsg[0] = OEMSYNC1
    sbas.NovatelMsg[1] = OEMSYNC2
    sbas.NovatelMsg[2] = OEMSYNC3
    setU2(sbas.NovatelMsg[4:], ID_RAWSBASFRAME)
    setU2(sbas.NovatelMsg[8:], OEMSBASLEN)
    setU2(sbas.NovatelMsg[14:], uint16(sbas.Week))
    setU4(sbas.NovatelMsg[16:], uint32(sbas.Tow*1000))
    setU4(sbas.NovatelMsg[OEMHLEN+4:], 183)
    setU4(sbas.NovatelMsg[OEMHLEN+8:], uint32(sbas.Id))
    for i := 0; i < 29; i++ {
        sbas.NovatelMsg[OEMHLEN+12+i] = sbas.Msg[i]
    }
    setU4(sbas.NovatelMsg[OEMHLEN+48:], crc32(sbas.NovatelMsg[:OEMHLEN+48], OEMHLEN+48))
}
func decode_MT12(buff []uint8, sbas *SdrSbas) {
    sbas.Tow = float64(getbitu(buff, 107, 20)) + 1.0
    sbas.Week = int(getbitu(buff, 127, 10)) + 1024
}
func decode_msg_sbas(buff []uint8, sbas *SdrSbas) {
    sbas.Id = int(getbitu(buff, 8, 6))
    switch sbas.Id {
    case 12:
        decode_MT12(buff, sbas)
    default:
        sbas.Tow += 1.0
    }
}
func decode_l1sbas(nav *SdrNav) int {
    bits := make([]int, 250)
    bin := make([]uint8, 29)
    pbin := make([]uint8, 3)
    for i := 0; i < 250; i++ {
        bits[i] = nav.Polarity * nav.FBitsDec[i]
    }
    bits2byte(bits[:226], 226, 29, 1, bin)
    bits2byte(bits[226:], 24, 3, 0, pbin)
    crc := crc24q(bin, 29)
    crcmsg := getbitu(pbin, 0, 24)
    if crc != crcmsg {
        SDRPRINTF("error: parity mismatch crc=%d msg=%d\n", crc, crcmsg)
    }
    msg := make([]uint8, 32)
    bits2byte(bits, 250, 32, 0, msg)
    decode_msg_sbas(msg, &nav.Sbas)
    if sdrini.Nch > 1 && sdrch[sdrini.Nch-2].Nav.SdrEph.WeekGpst != 0 {
        nav.Sbas.Tow = sdrch[sdrini.Nch-2].Trk.Tow[0]
        nav.Sbas.Week = sdrch[sdrini.Nch-2].Nav.SdrEph.WeekGpst
    }
    if nav.Sbas.Week != 0 {
        gen_novatel_sbasmsg(&nav.Sbas)
        nav.SdrEph.TowGpst = nav.Sbas.Tow
        nav.SdrEph.WeekGpst = nav.Sbas.Week
    }
    return nav.Sbas.Id
}

// Set unsigned bits in byte buffer (from RTKLIB)
func setbitu(buff []uint8, pos, length int, data uint32) {
    for i := 0; i < length; i++ {
        if ((data >> uint(length-i-1)) & 1) != 0 {
            buff[(pos+i)/8] |= 1 << uint(7-(pos+i)%8)
        } else {
            buff[(pos+i)/8] &^= 1 << uint(7-(pos+i)%8)
        }
    }
}

// CRC32 implementation (from RTKLIB)
func crc32(buff []byte, length int) uint32 {
    const poly = 0xEDB88320
    crc := uint32(0xFFFFFFFF)
    for i := 0; i < length; i++ {
        crc ^= uint32(buff[i])
        for j := 0; j < 8; j++ {
            if crc&1 != 0 {
                crc = (crc >> 1) ^ poly
            } else {
                crc >>= 1
            }
        }
    }
    return crc ^ 0xFFFFFFFF
}
