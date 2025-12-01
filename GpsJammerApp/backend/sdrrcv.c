#include "sdr.h"

extern int rcvinit(sdrini_t *ini) {
    fftwf_init_threads();

    sdrstat.buff = sdrstat.tmpbuff = NULL;

    if ((ini->fp = fopen(ini->file, "rb")) == NULL) {
        SDRPRINTF("error: failed to open file: %s\n", ini->file);
        return -1;
    }

    sdrstat.fendbuffsize = FILE_BUFFSIZE;
    sdrstat.buffsize = 2 * FILE_BUFFSIZE * MEMBUFFLEN;

    if (ini->fp != NULL) {
        sdrstat.buff = (uint8_t *)malloc(ini->dtype[0] * sdrstat.buffsize);
        if (NULL == sdrstat.buff) {
            SDRPRINTF("error: failed to allocate memory for the buffer\n");
            return -1;
        }
    }

    return 0;
}

extern int rcvquit(sdrini_t *ini) {

    if (ini->fp != NULL) {
        fclose(ini->fp);
        ini->fp = NULL;
    }

    if (NULL != sdrstat.buff) {
        free(sdrstat.buff);
        sdrstat.buff = NULL;
    }
    if (NULL != sdrstat.tmpbuff) {
        free(sdrstat.tmpbuff);
        sdrstat.tmpbuff = NULL;
    }
    return 0;
}

extern int rcvgrabstart(sdrini_t *ini) { return 0; }

extern int rcvgrabdata(sdrini_t *ini) {

    file_pushtomembuf();
    sleepms(5);

    return 0;
}

extern int rcvgetbuff(sdrini_t *ini, uint64_t buffloc, int n, int ftype,
                      int dtype, char *expbuf) {
    file_getbuff(buffloc, n, ftype, dtype, expbuf);
    return 0;
}

extern void file_pushtomembuf(void) {
    size_t nread = 0;

    mlock(hbuffmtx);
    if (sdrini.fp != NULL) {
        nread = fread(&sdrstat.buff[(sdrstat.buffcnt % MEMBUFFLEN) *
                                    sdrini.dtype[0] * FILE_BUFFSIZE],
                      1, sdrini.dtype[0] * FILE_BUFFSIZE, sdrini.fp);
    }
    unmlock(hbuffmtx);

    if ((sdrini.fp != NULL && (int)nread < sdrini.dtype[0] * FILE_BUFFSIZE)) {
        sdrstat.stopflag = ON;
        SDRPRINTF("end of file!\n");
    }

    mlock(hreadmtx);
    sdrstat.buffcnt++;
    unmlock(hreadmtx);
}

extern void file_getbuff(uint64_t buffloc, int n, int ftype, int dtype,
                         char *expbuf) {
    uint64_t membuffloc =
        dtype * buffloc % (MEMBUFFLEN * dtype * FILE_BUFFSIZE);
    int nout;
    int i;
    int total_bytes;

    n = dtype * n;
    total_bytes = n;
    nout = (int)((membuffloc + n) - (MEMBUFFLEN * dtype * FILE_BUFFSIZE));

    mlock(hbuffmtx);

    if (nout > 0) {
        memcpy(expbuf, &sdrstat.buff[membuffloc], n - nout);
        memcpy(&expbuf[(n - nout)], &sdrstat.buff[0], nout);
    } else {
        memcpy(expbuf, &sdrstat.buff[membuffloc], n);
    }
    unmlock(hbuffmtx);

    for (i = 0; i < total_bytes; i++) {
        expbuf[i] = (char)((unsigned char)expbuf[i] - 128);
    }
}

