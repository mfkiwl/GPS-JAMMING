//-----------------------------------------------------------------------------
// sdrrcv.c : SDR receiver functions
//
// Copyright (C) 2014 Taro Suzuki <gnsssdrlib@gmail.com>
// Edits from Don Kelly, don.kelly@mac.com, 2025
//-----------------------------------------------------------------------------
#include "sdr.h"

/* sdr receiver initialization -------------------------------------------------
* receiver initialization, memory allocation, file open
* args   : sdrini_t *ini    I   sdr initialization struct
* return : int                  status 0:okay -1:failure
*-----------------------------------------------------------------------------*/
extern int rcvinit(sdrini_t *ini)
{
        /* FFT initialization */
        fftwf_init_threads();

        sdrstat.buff=sdrstat.tmpbuff=NULL;

        switch (ini->fend) {

        /* frontend support removed */

        #ifdef RTLSDR
        /* RTL-SDR */
        case FEND_RTLSDR:
                if (rtlsdr_init()<0) return -1; /* rtlsdr initialization */

                /* frontend buffer size */
                sdrstat.fendbuffsize=RTLSDR_DATABUFF_SIZE; /* frontend buff size */
                sdrstat.buffsize=2*RTLSDR_DATABUFF_SIZE*MEMBUFFLEN; /* total */

                /* memory allocation */
                sdrstat.buff=(uint8_t*)malloc(sdrstat.buffsize);
                if (NULL==sdrstat.buff) {
                        SDRPRINTF("error: failed to allocate memory for the buffer\n");
                        return -1;
                }
                break;

        /* RTL-SDR Binary File */
        case FEND_FRTLSDR:
                /* IF file open */
                if ((ini->fp = fopen(ini->file,"rb"))==NULL) {
                        SDRPRINTF("error: failed to open file : %s\n",ini->file);
                        return -1;
                }

                /* frontend buffer size */
                sdrstat.fendbuffsize=RTLSDR_DATABUFF_SIZE; /* frontend buff size */
                sdrstat.buffsize=2*RTLSDR_DATABUFF_SIZE*MEMBUFFLEN; /* total */

                /* memory allocation */
                sdrstat.buff=(uint8_t*)malloc(sdrstat.buffsize);
                if (NULL==sdrstat.buff) {
                        SDRPRINTF("error: failed to allocate memory for the buffer\n");
                        return -1;
                }
                break;
        #endif

        /* File */
        case FEND_FILE:
                /* IF file open (FILE1) */
                if ((ini->fp = fopen(ini->file,"rb"))==NULL) {
                        SDRPRINTF("error: failed to open file: %s\n",ini->file);
                        return -1;
                }

                /* frontend buffer size */
                sdrstat.fendbuffsize=FILE_BUFFSIZE; /* frontend buff size */
                sdrstat.buffsize=FILE_BUFFSIZE*MEMBUFFLEN; /* total */

                /* memory allocation */
                if (ini->fp!=NULL) {
                        sdrstat.buff=(uint8_t*)malloc(ini->dtype[0]*sdrstat.buffsize);
                        if (NULL==sdrstat.buff) {
                                SDRPRINTF("error: failed to allocate memory for the buffer\n");
                                return -1;
                        }
                }
                break;
        default:
                return -1;
        }
        return 0;
}

/* stop front-end --------------------------------------------------------------
* stop grabber of front end
* args   : sdrini_t *ini    I   sdr initialization struct
* return : int                  status 0:okay -1:failure
*-----------------------------------------------------------------------------*/
extern int rcvquit(sdrini_t *ini)
{
        switch (ini->fend) {

        /* frontend support removed */

        #ifdef RTLSDR
        /* RTL-SDR */
        case FEND_RTLSDR:
                rtlsdr_quit();
                break;
        #endif

        /* Front End Binary File */
        case FEND_FRTLSDR:
        case FEND_FILE:
                if (ini->fp != NULL) {
                        fclose(ini->fp);
                        ini->fp = NULL;
                }
                break;
        default:
                return -1;
        }

        /* free memory */
        if (NULL!=sdrstat.buff) {
          free(sdrstat.buff);
          sdrstat.buff=NULL;
        }
                                if (NULL!=sdrstat.tmpbuff) {
                                        free(sdrstat.tmpbuff);
                                        sdrstat.tmpbuff=NULL;
                                }
        return 0;
}

/* start grabber ---------------------------------------------------------------
* start grabber of front end
* args   : sdrini_t *ini    I   sdr initialization struct
* return : int                  status 0:okay -1:failure
*-----------------------------------------------------------------------------*/
extern int rcvgrabstart(sdrini_t *ini)
{
        switch (ini->fend) {

        default:
                return 0;
        }
        return 0;
}

/* grab current data -----------------------------------------------------------
* push data to memory buffer from front end
* args   : sdrini_t *ini    I   sdr initialization struct
* return : int                  status 0:okay -1:failure
*-----------------------------------------------------------------------------*/
extern int rcvgrabdata(sdrini_t *ini)
{
        //unsigned long buffcnt=0; // DK commented out

        switch (ini->fend) {

        /* frontend support removed */

        #ifdef RTLSDR
        /* RTL-SDR */
        case FEND_RTLSDR:
                if (rtlsdr_start()<0) {
                        SDRPRINTF("error: rtlsdr...\n");
                        return -1;
                }
                break;

        /* RTL-SDR Binary File */
        case FEND_FRTLSDR:
                frtlsdr_pushtomembuf(); /* copy to membuffer */
                sleepms(5);
                break;/* File */
        #endif
        case FEND_FILE:
                file_pushtomembuf(); /* copy to membuffer */
                sleepms(5);
                break;
        default:
                return -1;
        }
        return 0;
}

/* grab current buffer ---------------------------------------------------------
* get current data buffer from memory buffer
* args   : sdrini_t *ini    I   sdr initialization struct
*          uint64_t buffloc I   buffer location
*          int    n         I   number of grab data
*          int    ftype     I   front end type (FTYPE1)
*          int    dtype     I   data type (DTYPEI or DTYPEIQ)
*          char   *expbuff  O   extracted data buffer
* return : int                  status 0:okay -1:failure
*-----------------------------------------------------------------------------*/
extern int rcvgetbuff(sdrini_t *ini, uint64_t buffloc, int n, int ftype,
                      int dtype, char *expbuf)
{
        switch (ini->fend) {

        /* frontend support removed */

        #ifdef RTLSDR
        /* RTL-SDR */
        case FEND_RTLSDR:
                rtlsdr_getbuff(buffloc,n,expbuf);
                break;

        /* RTL-SDR Binary File */
        case FEND_FRTLSDR:
                rtlsdr_getbuff(buffloc,n,expbuf);
                break;
        #endif

        /* File */
        case FEND_FILE:
                file_getbuff(buffloc,n,ftype,dtype,expbuf);
                break;
        default:
                return -1;
        }
        return 0;
}

/* push data to memory buffer --------------------------------------------------
* post-processing function: push data to memory buffer
* args   : none
* return : none
*-----------------------------------------------------------------------------*/
extern void file_pushtomembuf(void)
{
        size_t nread=0;

        mlock(hbuffmtx);
        if(sdrini.fp!=NULL) {
                nread=fread(&sdrstat.buff[(sdrstat.buffcnt%MEMBUFFLEN)*
                                           sdrini.dtype[0]*FILE_BUFFSIZE],1,sdrini.dtype[0]*FILE_BUFFSIZE,
                             sdrini.fp);
        }
        unmlock(hbuffmtx);

        if ((sdrini.fp!=NULL&&(int)nread<sdrini.dtype[0]*FILE_BUFFSIZE)) {
                sdrstat.stopflag=ON;
                SDRPRINTF("end of file!\n");
        }

        mlock(hreadmtx);
        sdrstat.buffcnt++;
        unmlock(hreadmtx);
}

/* get current data buffer from IF file ----------------------------------------
* post-processing function: get current data buffer from memory buffer
* args   : uint64_t buffloc I   buffer location
*          int    n         I   number of grab data
*          int    ftype     I   front end type (FTYPE1)
*          int    dtype     I   data type (DTYPEI or DTYPEIQ)
*          char   *expbuff  O   extracted data buffer
* return : none
*-----------------------------------------------------------------------------*/
extern void file_getbuff(uint64_t buffloc, int n, int ftype, int dtype,
                         char *expbuf)
{
        uint64_t membuffloc=dtype*buffloc%(MEMBUFFLEN*dtype*FILE_BUFFSIZE);
        int nout;

        n=dtype*n;
        nout=(int)((membuffloc+n)-(MEMBUFFLEN*dtype*FILE_BUFFSIZE));

        mlock(hbuffmtx);
        /* Only one frontend/file buffer is supported in simplified mode */
        if (nout>0) {
                memcpy(expbuf,&sdrstat.buff[membuffloc],n-nout);
                memcpy(&expbuf[(n-nout)],&sdrstat.buff[0],nout);
        } else {
                memcpy(expbuf,&sdrstat.buff[membuffloc],n);
        }
        unmlock(hbuffmtx);
}
