#include "sdr.h"
#define P2_34 5.820766091346741E-11
#define P2_46 1.421085471520200E-14
#define P2_59 1.734723475976807E-18
#define OFFSET1 2
#define OFFSET2 122
void decode_word1(const uint8_t *buff, sdreph_t *eph) {
    int oldiodc = eph->eph.iodc;
    double sqrtA;
    eph->eph.iodc = getbitu(buff, OFFSET1 + 6, 10);
    eph->eph.toes = getbitu(buff, OFFSET1 + 16, 14) * 60;
    eph->eph.M0 = getbits(buff, OFFSET1 + 30, 32) * P2_31 * SC2RAD;
    eph->eph.e = getbitu(buff, OFFSET1 + 62, 32) * P2_33;
    sqrtA = getbitu2(buff, OFFSET1 + 94, 18, OFFSET2 + 0, 14) * P2_19;
    eph->eph.A = sqrtA * sqrtA;
    eph->eph.iode = eph->eph.iodc;
    eph->eph.code = 513;
    if (eph->week_gst != 0)
        eph->eph.toe = gst2time(eph->week_gst, eph->eph.toes);
    if (oldiodc - eph->eph.iodc != 0)
        eph->update = ON;
    eph->cnt++;
}

void decode_word2(const uint8_t *buff, sdreph_t *eph) {
    int oldiodc = eph->eph.iodc;
    eph->eph.iodc = getbitu(buff, OFFSET1 + 6, 10);
    eph->eph.OMG0 = getbits(buff, OFFSET1 + 16, 32) * P2_31 * SC2RAD;
    eph->eph.i0 = getbits(buff, OFFSET1 + 48, 32) * P2_31 * SC2RAD;
    eph->eph.omg = getbits(buff, OFFSET1 + 80, 32) * P2_31 * SC2RAD;
    eph->eph.idot = getbits(buff, OFFSET2 + 0, 14) * P2_43 * SC2RAD;
    eph->eph.iode = eph->eph.iodc;
    if (oldiodc - eph->eph.iodc != 0)
        eph->update = ON;
    eph->cnt++;
}

void decode_word3(const uint8_t *buff, sdreph_t *eph) {
    int oldiodc = eph->eph.iodc;
    eph->eph.iodc = getbitu(buff, OFFSET1 + 6, 10);
    eph->eph.OMGd = getbits(buff, OFFSET1 + 16, 24) * P2_43 * SC2RAD;
    eph->eph.deln = getbits(buff, OFFSET1 + 40, 16) * P2_43 * SC2RAD;
    eph->eph.cuc = getbits(buff, OFFSET1 + 56, 16) * P2_29;
    eph->eph.cus = getbits(buff, OFFSET1 + 72, 16) * P2_29;
    eph->eph.crc = getbits(buff, OFFSET1 + 88, 16) * P2_5;
    eph->eph.crs = getbits2(buff, OFFSET1 + 104, 8, OFFSET2 + 0, 8) * P2_5;
    eph->eph.sva = 0;
    eph->eph.iode = eph->eph.iodc;
    if (oldiodc - eph->eph.iodc != 0)
        eph->update = ON;
    eph->cnt++;
}

void decode_word4(const uint8_t *buff, sdreph_t *eph) {
    int oldiodc = eph->eph.iodc;
    eph->eph.iodc = getbitu(buff, OFFSET1 + 6, 10);
    eph->eph.cic = getbits(buff, OFFSET1 + 22, 16) * P2_29;
    eph->eph.cis = getbits(buff, OFFSET1 + 38, 16) * P2_29;
    eph->toc_gst = getbitu(buff, OFFSET1 + 54, 14) * 60;
    eph->eph.f0 = getbits(buff, OFFSET1 + 68, 31) * P2_34;
    eph->eph.f1 = getbits2(buff, OFFSET1 + 99, 13, OFFSET2 + 0, 8) * P2_46;
    eph->eph.f2 = getbits(buff, OFFSET2 + 8, 6) * P2_59;
    eph->eph.iode = eph->eph.iodc;
    if (eph->week_gst != 0)
        eph->eph.toc = gst2time(eph->week_gst, eph->toc_gst);
    if (oldiodc - eph->eph.iodc != 0)
        eph->update = ON;
    eph->cnt++;
}

void decode_word5(const uint8_t *buff, sdreph_t *eph) {
    unsigned int e1bdvs, e1bhs, e5bdvs, e5bhs;
    double tow_gst;
    eph->eph.tgd[0] = getbits(buff, OFFSET1 + 47, 10) * P2_32;
    eph->eph.tgd[1] = getbits(buff, OFFSET1 + 57, 10) * P2_32;
    e5bhs = getbitu(buff, OFFSET1 + 67, 2);
    e1bhs = getbitu(buff, OFFSET1 + 69, 2);
    e5bdvs = getbitu(buff, OFFSET1 + 71, 1);
    e1bdvs = getbitu(buff, OFFSET1 + 72, 1);
    eph->week_gst = getbitu(buff, OFFSET1 + 73, 12);
    eph->eph.week = eph->week_gst + 1024;
    tow_gst = getbitu(buff, OFFSET1 + 85, 20) + 2.0;
    eph->tow_gpst =
        time2gpst(gst2time(eph->week_gst, tow_gst), &eph->week_gpst);
    eph->eph.ttr = gst2time(eph->week_gst, tow_gst);
    if (eph->toc_gst != 0)
        eph->eph.toc = gst2time(eph->week_gst, eph->toc_gst);
    if (eph->eph.toes != 0)
        eph->eph.toe = gst2time(eph->week_gst, eph->eph.toes);
    eph->eph.svh = (e5bhs << 7) + (e5bdvs << 6) + (e1bhs << 1) + (e1bdvs);
    eph->cnt++;
}

void decode_word6(const uint8_t *buff, sdreph_t *eph) {
    double tow_gst;
    tow_gst = getbitu2(buff, OFFSET1 + 105, 7, OFFSET2 + 0, 13) + 2.0;
    if (eph->week_gst != 0) {
        eph->tow_gpst =
            time2gpst(gst2time(eph->week_gst, tow_gst), &eph->week_gpst);
        eph->eph.ttr = gst2time(eph->week_gst, tow_gst);
    }
}

void decode_word0(const uint8_t *buff, sdreph_t *eph) {
    double tow_gst;
    eph->week_gst = getbitu(buff, OFFSET1 + 96, 12);
    eph->eph.week = eph->week_gst + 1024;
    tow_gst = getbitu2(buff, OFFSET1 + 108, 4, OFFSET2 + 0, 16) + 2.0;
    eph->tow_gpst =
        time2gpst(gst2time(eph->week_gst, tow_gst), &eph->week_gpst);
    eph->eph.ttr = gst2time(eph->week_gst, tow_gst);
}

extern int checkcrc_e1b(uint8_t *data1, uint8_t *data2) {
    uint8_t crcbins[25] = {0};
    int i, j, crcbits[196], crc, crcmsg;
    for (i = 0; i < 15; i++) {
        for (j = 0; j < 8; j++) {
            if (8 * i + j == 114)
                break;
            crcbits[8 * i + j] = -2 * (((data1[i] << j) & 0x80) >> 7) + 1;
        }
    }
    for (i = 0; i < 11; i++) {
        for (j = 0; j < 8; j++) {
            if (8 * i + j == 82)
                break;
            crcbits[114 + 8 * i + j] = -2 * (((data2[i] << j) & 0x80) >> 7) + 1;
        }
    }
    bits2byte(crcbits, 196, 25, 1, crcbins);
    crc = crc24q(crcbins, 25);
    crcmsg = getbitu(data2, 82, 24);
    if (crc == crcmsg)
        return 0;
    else
        return -1;
}

extern int decode_page_e1b(const uint8_t *buff1, const uint8_t *buff2,
                           sdreph_t *eph) {
    int id;
    uint8_t buff[30];
    memcpy(buff, buff1, 15);
    memcpy(&buff[15], buff2, 15);
    id = getbitu(buff, 2, 6);
    switch (id) {
    case 0:
        decode_word0(buff, eph);
        break;
    case 1:
        decode_word1(buff, eph);
        break;
    case 2:
        decode_word2(buff, eph);
        break;
    case 3:
        decode_word3(buff, eph);
        break;
    case 4:
        decode_word4(buff, eph);
        break;
    case 5:
        decode_word5(buff, eph);
        break;
    case 6:
        decode_word6(buff, eph);
        break;
    default:
        break;
    }
    return id;
}

extern int decode_e1b(sdrnav_t *nav) {
    int i, id = 0, bits[500], bits_e1b[240];
    uint8_t enc_e1b[240], dec_e1b1[15], dec_e1b2[15];
    for (i = 0; i < nav->flen; i++)
        bits[i] = nav->polarity * nav->fbits[i];
    init_viterbi27_port(nav->fec, 0);
    interleave(&bits[10], 30, 8, bits_e1b);
    for (i = 0; i < 240; i++) {
        if (i % 2 == 0)
            enc_e1b[i] = (bits_e1b[i] == 1) ? 0 : 255;
        else
            enc_e1b[i] = (bits_e1b[i] == 1) ? 255 : 0;
    }
    update_viterbi27_blk_port(nav->fec, enc_e1b, 120);
    chainback_viterbi27_port(nav->fec, dec_e1b1, 120 - 6, 0);
    init_viterbi27_port(nav->fec, 0);
    interleave(&bits[260], 30, 8, bits_e1b);
    for (i = 0; i < 240; i++) {
        if (i % 2 == 0)
            enc_e1b[i] = (bits_e1b[i] == 1) ? 0 : 255;
        else
            enc_e1b[i] = (bits_e1b[i] == 1) ? 255 : 0;
    }
    update_viterbi27_blk_port(nav->fec, enc_e1b, 120);
    chainback_viterbi27_port(nav->fec, dec_e1b2, 120 - 6, 0);
    if (getbitu(dec_e1b1, 0, 1)) {
        nav->flagsyncf = OFF;
        nav->flagtow = OFF;
        return -1;
    }
    if (checkcrc_e1b(dec_e1b1, dec_e1b2) < 0) {
        SDRPRINTF("error: E1B CRC mismatch\n");
        return -1;
    }
    id = decode_page_e1b(dec_e1b1, dec_e1b2, &nav->sdreph);
    return id;
}
