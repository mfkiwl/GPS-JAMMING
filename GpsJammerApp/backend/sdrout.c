//----------------------------------------------------------------------------
//
//  sdrgui.c
//
//  Copyright 2025, Don Kelly, don.kelly@mac.com
//----------------------------------------------------------------------------

#include "sdr.h"

void init_sdrgui_messages() {
  for (int i = 0; i < MAX_MESSAGES; i++) {
    sdrgui.messages[i] = NULL;
  }
  sdrgui.message_count = 0;
}

extern void add_message(const char *msg)
{
  pthread_mutex_lock(&hmsgmtx);
  if (sdrgui.message_count < MAX_MESSAGES) {
      sdrgui.messages[sdrgui.message_count++] = strdup(msg);
  } else {
    if (sdrgui.messages[0] != NULL) {
      free(sdrgui.messages[0]);
      sdrgui.messages[0] = NULL;
    }
    for (int i = 1; i < MAX_MESSAGES; i++) {
      sdrgui.messages[i - 1] = sdrgui.messages[i];
    }
    sdrgui.messages[MAX_MESSAGES - 1] = strdup(msg);
  }
  pthread_mutex_unlock(&hmsgmtx);
}

extern void updateNavStatusWin(int counter)
{
  // Pull data from sdrstat and sdrch
  int prn[32] = {0};
  int flagacq[32] = {0};
  int flagsync[32] = {0};
  int flagdec[32] = {0};
  int nsat = 0;
  double lat = 0.0;
  double lon = 0.0;
  double hgt = 0.0;
  double gdop = 0.0;
  double clkBias = 0.0;
  double obs_v[MAXSAT*11] = {0.0};
  double vk1_v[MAXSAT] = {0.0};
  double rk1_v[MAXSAT] = {0.0};
  int gps_week;
  double gps_tow;
  char bufferNav[256];
  char str1[10];

  // Load in data to display
  mlock(hobsvecmtx);
  for (int i=0; i<32; i++) {
    prn[i] = sdrch[i].prn;
    flagacq[i] = sdrch[i].flagacq;
    flagsync[i] = sdrch[i].nav.flagsync;
    flagdec[i] = sdrch[i].nav.flagdec;
  }
  nsat = sdrstat.nsatValid;
  lat = sdrstat.lat;
  lon = sdrstat.lon;
  hgt = sdrstat.hgt;
  gdop = sdrstat.gdop;
  clkBias = sdrstat.xyzdt[3];
  int numIter = 32*11;
  for (int m=0; m<numIter; m++) {
    obs_v[m] = sdrstat.obs_v[m];
  }
  for (int n=0; n<32; n++) {
    vk1_v[n] = sdrstat.vk1_v[n];
    rk1_v[n] = sdrekf.rk1_v[n];
  }
  gps_tow = sdrstat.obs_v[(sdrstat.obsValidList[0]-1)*11+6] ;
  gps_week = (int)sdrstat.obs_v[(sdrstat.obsValidList[0]-1)*11+7];
  unmlock(hobsvecmtx);

  // Correct rcvr TOW with rcvr clock bias for precise UTC
  time_t utc_time_seconds = gps_to_utc(gps_week, gps_tow+clkBias/CTIME);
  struct tm utc_tm;
  gmtime_r(&utc_time_seconds, &utc_tm);
  sprintf(bufferNav,"%04d-%02d-%02d %02d:%02d:%02d.%03d",
     utc_tm.tm_year + 1900, utc_tm.tm_mon + 1, utc_tm.tm_mday,
     utc_tm.tm_hour, utc_tm.tm_min, utc_tm.tm_sec, (int)(gps_tow * 1000) % 1000);

  printf("ETIME|%.3f\n", sdrstat.elapsedTime);
  printf("TIME|%s\n", bufferNav);

  // Update filter mode
  if (sdrini.ekfFilterOn) {
    printf("FILTER|EKF\n");
  } else {
    printf("FILTER|WLS\n");
  }

  // Update acquired SVs
  sprintf(bufferNav, "");
  for (int i=0; i<32; i++) {
    if (flagacq[i] ==1) {
      sprintf(str1, "%02d ", prn[i]);
      strcat(bufferNav, str1);
    }
  }
  printf("ACQSV|%s\n", bufferNav);

  // Update tracked SVs
  sprintf(bufferNav, "");
  for (int i=0; i<32; i++) {
    if (flagsync[i] ==1) {
      sprintf(str1, "%02d ", prn[i]);
      strcat(bufferNav, str1);
    }
  }
  printf("TRACKED|%s\n", bufferNav);

  // Update nav decoded SVs
  sprintf(bufferNav, "");
  for (int i=0; i<32; i++) {
    if (flagdec[i] ==1) {
      sprintf(str1, "%02d ", prn[i]);
      strcat(bufferNav, str1);
    }
  }
  printf("DECODED|%s\n", bufferNav);

  // Update LLA data
  // LAT, LON, ALT, GDOP, CB
  sprintf(bufferNav, "%.7f|%.7f|%.1f|%.2f|%.5e",
    lat, lon, hgt, gdop, clkBias/CTIME);
  printf("LLA|%02d|%s\n", nsat, bufferNav);

  // Display Obs data for all valid SVs once it is calculated
  for (int i=0; i<nsat; i++) {
    int prn = sdrstat.obsValidList[i];
    // G, TOW, WEEK, SNR, PR, AZ, EL, RK1, VK1
    sprintf(bufferNav, "%02d|%.1f|%d|%.1f|%.1f|%05.1f|%04.1f|%05.1f|%7.1f",
      (int)obs_v[(prn-1)*11+0],
      obs_v[(prn-1)*11+6],
      (int)obs_v[(prn-1)*11+7],
      obs_v[(prn-1)*11+8],
      obs_v[(prn-1)*11+5],
      obs_v[(prn-1)*11+9],
      obs_v[(prn-1)*11+10],
      rk1_v[(prn-1)],
      vk1_v[(prn-1)]);
    printf("OBS|%s\n", bufferNav);
  }
}

