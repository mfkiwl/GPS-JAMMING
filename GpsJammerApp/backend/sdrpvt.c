#include "nml.h"
#include "sdr.h"
#define HALF_WEEK_SEC 302400.0
#define FULL_WEEK_SEC 604800.0
#define WGS84_A 6378137.0
#define WGS84_FINV 298.257223563
#define CONVERGENCE_TOL 1e-10
#define DETERMINANT_TOL 1e-12
#define MAX_ITER_POSITION 10
#define MAX_ITER_GEODETIC 50
#define COORD_DIM 3
#define STATE_DIM 4
#define SQ(x) ((x) * (x))
#define CUBE(x) ((x) * (x) * (x))
#define OBS_IDX(prn, field) (((prn) - 1) * 14 + (field))
typedef struct {
    double ecef[COORD_DIM];
    double clkBias;
} ReceiverState;
typedef struct {
    double pos[COORD_DIM];
    double clkCorr;
} SatelliteData;
typedef struct {
    double lat;
    double lon;
    double alt;
} GeodeticCoord;
typedef struct {
    double azimuth;
    double elevation;
    double range;
} TopocentricCoord;
static void wrapTimeToHalfWeek(double rawTime, double *wrapped);
static double computeEccentricAnomaly(double meanAnomaly, double eccentricity,
                                      int iterations);
static void applyEarthRotation(const double satPos[], double travelTime,
                               double rotated[]);
static int computeKeplerianPosition(const eph_t *eph, double txTime,
                                    SatelliteData *sat);
static int computeGlonassPosition(const geph_t *geph, double txTime,
                                  SatelliteData *sat);
static void initWeightMatrix(int numSats, double *weights);
static void ecefToGeodetic(const double ecef[], GeodeticCoord *geo);
static int ecefToTopocentric(const double refEcef[], const double deltaXyz[],
                             TopocentricCoord *topo);
static int validateEphemerisGps(int prn);
static int validateEphemerisGlo(int prn);
static void markSatelliteInvalid(int prn, const char *reason);
typedef struct {
    double *pseudoranges;
    double *satPositions;
    double *rcvrTow;
    double *rawPr;
    double *corrPr;
    double *snr;
} PvtBuffers;
static PvtBuffers *allocatePvtBuffers(int numSat) {
    PvtBuffers *buf = (PvtBuffers *)malloc(sizeof(PvtBuffers));
    if (!buf)
        return NULL;
    buf->pseudoranges = (double *)malloc(numSat * sizeof(double));
    buf->satPositions = (double *)malloc(numSat * COORD_DIM * sizeof(double));
    buf->rcvrTow = (double *)malloc(numSat * sizeof(double));
    buf->rawPr = (double *)malloc(numSat * sizeof(double));
    buf->corrPr = (double *)malloc(numSat * sizeof(double));
    buf->snr = (double *)malloc(numSat * sizeof(double));
    return buf;
}

static void freePvtBuffers(PvtBuffers *buf) {
    if (!buf)
        return;
    free(buf->pseudoranges);
    free(buf->satPositions);
    free(buf->rcvrTow);
    free(buf->rawPr);
    free(buf->corrPr);
    free(buf->snr);
    free(buf);
}

typedef struct {
    nml_mat *design;
    nml_mat *weight;
    nml_mat *residual;
    nml_mat *solution;
    nml_mat *position;
    nml_mat *designT;
    nml_mat *ata;
    nml_mat *atw;
    nml_mat *atwa;
    nml_mat *invAta;
    nml_mat *invAtwa;
    nml_mat *gain1;
    nml_mat *gain2;
    nml_mat *tempState;
} LsqMatrices;
static LsqMatrices *initLsqMatrices(int numObs) {
    LsqMatrices *m = (LsqMatrices *)malloc(sizeof(LsqMatrices));
    if (!m)
        return NULL;
    m->solution = nml_mat_new(STATE_DIM, 1);
    m->position = nml_mat_new(STATE_DIM, 1);
    m->weight = nml_mat_new(numObs, numObs);
    m->residual = nml_mat_new(numObs, 1);
    m->designT = nml_mat_new(STATE_DIM, numObs);
    m->ata = nml_mat_new(STATE_DIM, STATE_DIM);
    m->atw = nml_mat_new(STATE_DIM, numObs);
    m->atwa = nml_mat_new(STATE_DIM, STATE_DIM);
    m->invAta = nml_mat_new(STATE_DIM, STATE_DIM);
    m->invAtwa = nml_mat_new(STATE_DIM, STATE_DIM);
    m->gain1 = nml_mat_new(STATE_DIM, numObs);
    m->gain2 = nml_mat_new(STATE_DIM, numObs);
    m->tempState = nml_mat_new(STATE_DIM, 1);
    m->design = NULL;
    return m;
}

static void freeLsqMatrices(LsqMatrices *m) {
    if (!m)
        return;
    nml_mat_free(m->solution);
    nml_mat_free(m->design);
    nml_mat_free(m->weight);
    nml_mat_free(m->position);
    nml_mat_free(m->residual);
    nml_mat_free(m->designT);
    nml_mat_free(m->ata);
    nml_mat_free(m->atw);
    nml_mat_free(m->atwa);
    nml_mat_free(m->invAta);
    nml_mat_free(m->invAtwa);
    nml_mat_free(m->gain1);
    nml_mat_free(m->gain2);
    nml_mat_free(m->tempState);
    free(m);
}

static void wrapTimeToHalfWeek(double rawTime, double *wrapped) {
    *wrapped = rawTime;
    if (rawTime > HALF_WEEK_SEC) {
        *wrapped = rawTime - FULL_WEEK_SEC;
    } else if (rawTime < -HALF_WEEK_SEC) {
        *wrapped = rawTime + FULL_WEEK_SEC;
    }
}

extern void normalizeGpsTime(double time, double *corrTime) {
    wrapTimeToHalfWeek(time, corrTime);
}

static double computeEccentricAnomaly(double M, double e, int maxIter) {
    double E = M;
    for (int k = 0; k < maxIter; k++) {
        double dE = (M - E + e * sin(E)) / (1.0 - e * cos(E));
        E += dE;
    }
    return E;
}

static int computeKeplerianPosition(const eph_t *eph, double txTime,
                                    SatelliteData *sat) {
    double sqrtA = sqrt(eph->A);
    double semiMajor = eph->A;
    double meanMotion0 = sqrt(MU / CUBE(semiMajor));
    double tk = txTime - eph->toes;
    if (tk > HALF_WEEK_SEC)
        tk -= FULL_WEEK_SEC;
    else if (tk < -HALF_WEEK_SEC)
        tk += FULL_WEEK_SEC;
    double n = meanMotion0 + eph->deln;
    double Mk = eph->M0 + n * tk;
    double Ek = computeEccentricAnomaly(Mk, eph->e, 3);
    double sinE = sin(Ek);
    double cosE = cos(Ek);
    double sqrtOneMinusE2 = sqrt(1.0 - SQ(eph->e));
    double vk = atan2(sqrtOneMinusE2 * sinE, cosE - eph->e);
    double phik = eph->omg + vk;
    double sin2phi = sin(2.0 * phik);
    double cos2phi = cos(2.0 * phik);
    double duk = eph->cus * sin2phi + eph->cuc * cos2phi;
    double drk = eph->crs * sin2phi + eph->crc * cos2phi;
    double dik = eph->cis * sin2phi + eph->cic * cos2phi;
    double uk = phik + duk;
    double rk = semiMajor * (1.0 - eph->e * cosE) + drk;
    double ik = eph->i0 + eph->idot * tk + dik;
    double xkp = rk * cos(uk);
    double ykp = rk * sin(uk);
    double Omegak =
        eph->OMG0 + (eph->OMGd - OMEGAEDOT) * tk - OMEGAEDOT * eph->toes;
    double cosOmega = cos(Omegak);
    double sinOmega = sin(Omegak);
    double cosInc = cos(ik);
    double sinInc = sin(ik);
    sat->pos[0] = xkp * cosOmega - ykp * sinOmega * cosInc;
    sat->pos[1] = xkp * sinOmega + ykp * cosOmega * cosInc;
    sat->pos[2] = ykp * sinInc;
    if (isnan(sat->pos[0]) || isnan(sat->pos[1]) || isnan(sat->pos[2])) {
        return -1;
    }
    double tc = txTime - eph->toes;
    if (tc > HALF_WEEK_SEC)
        tc -= FULL_WEEK_SEC;
    else if (tc < -HALF_WEEK_SEC)
        tc += FULL_WEEK_SEC;
    double relativistic = -2.0 * sqrt(MU) / SQ(CTIME) * eph->e * sqrtA * sinE;
    sat->clkCorr =
        eph->f0 + eph->f1 * tc + eph->f2 * SQ(tc) - eph->tgd[0] + relativistic;
    return 0;
}

static int computeGlonassPosition(const geph_t *geph, double txTime,
                                  SatelliteData *sat) {
    gtime_t t = gpst2time(0, txTime);
    double dt = timediff(t, geph->toe);
    double dt2 = SQ(dt);
    sat->pos[0] = geph->pos[0] + geph->vel[0] * dt + geph->acc[0] * dt2 * 0.5;
    sat->pos[1] = geph->pos[1] + geph->vel[1] * dt + geph->acc[1] * dt2 * 0.5;
    sat->pos[2] = geph->pos[2] + geph->vel[2] * dt + geph->acc[2] * dt2 * 0.5;
    sat->clkCorr = -geph->taun + geph->gamn * dt;
    if (isnan(sat->pos[0]) || isnan(sat->pos[1]) || isnan(sat->pos[2])) {
        return -1;
    }
    return 0;
}

extern int computeSvCoordinates(sdreph_t *sdreph, double transmitTime,
                                double svPos[3], double *svClkCorr) {
    SatelliteData satData = {{0}, 0};
    int status;
    if (sdreph->ctype == CTYPE_G1) {
        status = computeGlonassPosition(&sdreph->geph, transmitTime, &satData);
    } else {
        status = computeKeplerianPosition(&sdreph->eph, transmitTime, &satData);
    }
    if (status == 0) {
        svPos[0] = satData.pos[0];
        svPos[1] = satData.pos[1];
        svPos[2] = satData.pos[2];
        *svClkCorr = satData.clkCorr;
    }
    return status;
}

static void applyEarthRotation(const double satPos[], double travelTime,
                               double rotated[]) {
    double omega_tau = OMEGAEDOT * travelTime;
    double cosOmega = cos(omega_tau);
    double sinOmega = sin(omega_tau);
    rotated[0] = cosOmega * satPos[0] + sinOmega * satPos[1];
    rotated[1] = -sinOmega * satPos[0] + cosOmega * satPos[1];
    rotated[2] = satPos[2];
}

extern void buildAxisRotation(double R[9], double angleDeg, int axis) {
    memset(R, 0, 9 * sizeof(double));
    R[0] = R[4] = R[8] = 1.0;
    double rad = angleDeg * M_PI / 180.0;
    double c = cos(rad);
    double s = sin(rad);
    switch (axis) {
    case 1:
        R[4] = c;
        R[5] = s;
        R[7] = -s;
        R[8] = c;
        break;
    case 2:
        R[0] = c;
        R[2] = -s;
        R[6] = s;
        R[8] = c;
        break;
    case 3:
        R[0] = c;
        R[1] = s;
        R[3] = -s;
        R[4] = c;
        break;
    }
}

static void ecefToGeodetic(const double ecef[], GeodeticCoord *geo) {
    double f = 1.0 / WGS84_FINV;
    double e2 = (2.0 * f) - SQ(f);
    geo->lon = atan2(ecef[1], ecef[0]);
    double p = sqrt(SQ(ecef[0]) + SQ(ecef[1]));
    geo->alt = 0.0;
    geo->lat = atan2(ecef[2], p * (1.0 - e2));
    double N = WGS84_A / sqrt(1.0 - SQ(sin(geo->lat)));
    double deltaH = 1e6;
    while (deltaH > 0.01) {
        double prevH = geo->alt;
        geo->lat = atan2(ecef[2], p * (1.0 - e2 * N / (N + geo->alt)));
        N = WGS84_A / sqrt(1.0 - SQ(e2 * sin(geo->lat)));
        geo->alt = p / cos(geo->lat) - N;
        deltaH = fabs(geo->alt - prevH);
    }
}

extern void cartesianToGeodetic(double x, double y, double z, double *lambda,
                                double *phi, double *height) {
    double ecef[3] = {x, y, z};
    GeodeticCoord geo;
    ecefToGeodetic(ecef, &geo);
    *lambda = geo.lon;
    *phi = geo.lat;
    *height = geo.alt;
}

extern int xyz2GeodeticDeg(double a, double finv, double X, double Y, double Z,
                           double *dphi, double *dlambda, double *h) {
    *h = 0.0;
    double esq = (finv < 1e-20) ? 0.0 : (2.0 - 1.0 / finv) / finv;
    double oneMinusE2 = 1.0 - esq;
    double p = sqrt(SQ(X) + SQ(Y));
    *dlambda = (p > 1e-20) ? atan2(Y, X) * R2D : 0.0;
    if (*dlambda < 0)
        *dlambda += 360.0;
    double r = sqrt(SQ(p) + SQ(Z));
    double sinPhi = (r > 1e-20) ? Z / r : 0.0;
    *dphi = asin(sinPhi);
    if (r < 1e-20) {
        printf("xyz2GeodeticDeg--- errorDetected\n");
        return -1;
    }
    *h = r - a * (1.0 - SQ(sinPhi) / finv);
    for (int iter = 0; iter < MAX_ITER_GEODETIC; iter++) {
        double sp = sin(*dphi);
        double cp = cos(*dphi);
        double Nphi = a / sqrt(1.0 - esq * SQ(sp));
        double dP = p - (Nphi + *h) * cp;
        double dZ = Z - (Nphi * oneMinusE2 + *h) * sp;
        *h += sp * dZ + cp * dP;
        *dphi += (cp * dZ - sp * dP) / (Nphi + *h);
        if (SQ(dP) + SQ(dZ) < 1e-10)
            break;
        if (iter == MAX_ITER_GEODETIC - 1) {
            printf("xyz2GeodeticDeg: convergence failed after %d iterations\n",
                   iter);
        }
    }
    *dphi *= R2D;
    return 0;
}

static int ecefToTopocentric(const double refEcef[], const double deltaXyz[],
                             TopocentricCoord *topo) {
    double phi, lambda, h;
    int ret = xyz2GeodeticDeg(WGS84_A, WGS84_FINV, refEcef[0], refEcef[1],
                              refEcef[2], &phi, &lambda, &h);
    if (ret != 0)
        return ret;
    double cl = cos(lambda * D2R);
    double sl = sin(lambda * D2R);
    double cb = cos(phi * D2R);
    double sb = sin(phi * D2R);
    double E = -sl * deltaXyz[0] + cl * deltaXyz[1];
    double N =
        -sb * cl * deltaXyz[0] - sb * sl * deltaXyz[1] + cb * deltaXyz[2];
    double U = cb * cl * deltaXyz[0] + cb * sl * deltaXyz[1] + sb * deltaXyz[2];
    double horDist = sqrt(SQ(E) + SQ(N));
    if (horDist < 1e-20) {
        topo->azimuth = 0.0;
        topo->elevation = 90.0;
    } else {
        topo->azimuth = atan2(E, N) / D2R;
        topo->elevation = atan2(U, horDist) / D2R;
    }
    if (topo->azimuth < 0)
        topo->azimuth += 360.0;
    topo->range = sqrt(SQ(deltaXyz[0]) + SQ(deltaXyz[1]) + SQ(deltaXyz[2]));
    return 0;
}

extern int calculateLocalAngles(double X[], double dx[], double *Az, double *El,
                                double *D) {
    TopocentricCoord topo = {0, 0, 0};
    int ret = ecefToTopocentric(X, dx, &topo);
    *Az = topo.azimuth;
    *El = topo.elevation;
    *D = topo.range;
    return ret;
}

extern int estimateAtmosphericDelay(double sinel, double hsta, double p,
                                    double tkel, double hum, double hp,
                                    double htkel, double hhum, double *ddr) {
    double a_e = 6378.137;
    double b0 = 7.839257e-5;
    double tlapse = -6.5;
    double tkhum = tkel + tlapse * (hhum - htkel);
    double atkel = 7.5 * (tkhum - 273.15) / (237.3 + tkhum - 273.15);
    double e0 = 0.0611 * hum * pow(10, atkel);
    double tksea = tkel - tlapse * htkel;
    double em = -978.77 / (2.8704e6 * tlapse * 1.0e-5);
    double tkelh = tksea + tlapse * hhum;
    double e0sea = e0 * pow((tksea / tkelh), (4 * em));
    double tkelp = tksea + tlapse * hp;
    double psea = p * pow((tksea / tkelp), em);
    double alpha[8] = {0};
    double rn[8] = {0};
    if (sinel < 0)
        sinel = 0;
    double tropoVal = 0;
    int done = 0;
    double refsea = 77.624e-6 / tksea;
    double htop = 1.1385e-5 / refsea;
    refsea = refsea * psea;
    double ref = refsea * pow(((htop - hsta) / htop), 4);
    while (1) {
        double rtop = SQ(a_e + htop) - SQ(a_e + hsta) * (1 - SQ(sinel));
        rtop = (rtop < 0) ? 0 : sqrt(rtop) - (a_e + hsta) * sinel;
        double a = -sinel / (htop - hsta);
        double b = -b0 * (1 - SQ(sinel)) / (htop - hsta);
        for (int i = 0; i < 7; i++)
            rn[i] = pow(rtop, i + 2);
        alpha[0] = 2 * a;
        alpha[1] = 2 * SQ(a) + 4 * b / 3;
        alpha[2] = a * (SQ(a) + 3 * b);
        alpha[3] = SQ(a) * SQ(a) / 5 + 2.4 * SQ(a) * b + 1.2 * SQ(b);
        alpha[4] = 2 * a * b * (SQ(a) + 3 * b) / 3;
        alpha[5] = SQ(b) * (6 * SQ(a) + 4 * b) * 1.428571e-1;
        alpha[6] = (SQ(b) > 1e-35) ? a * CUBE(b) / 2 : 0;
        alpha[7] = (SQ(b) > 1e-35) ? SQ(b) * SQ(b) / 9 : 0;
        double dr = rtop;
        for (int i = 0; i < 7; i++)
            dr += alpha[i] * rn[i];
        tropoVal += dr * ref * 1000;
        if (done) {
            *ddr = tropoVal;
            break;
        }
        done = 1;
        refsea = (371900.0e-6 / tksea - 12.92e-6) / tksea;
        htop = 1.1385e-5 * (1255 / tksea + 0.05) / refsea;
        ref = refsea * e0sea * pow(((htop - hsta) / htop), 4);
    }
    return 0;
}

static void initWeightMatrix(int numSats, double *weights) {
    double baseVariance = 25.0;
    mlock(hobsvecmtx);
    sdrekf.varR = baseVariance;
    memset(sdrekf.rk1_v, 0, MAXSAT * sizeof(double));
    for (int k = 0; k < numSats; k++) {
        int prn = sdrstat.obsValidList[k];
        if (prn < 1 || prn > MAXSAT)
            continue;
        sdrekf.rk1_v[prn - 1] = baseVariance;
        double elev = sdrstat.obs_v[OBS_IDX(prn, 10)];
        if (sdrstat.azElCalculatedflag && elev < 30.0) {
            double elevFactor = 25.0 - (25.0 / 15.0) * (elev - 15.0);
            sdrekf.rk1_v[prn - 1] = baseVariance + SQ(elevFactor);
        }
        weights[k + k * numSats] = 1.0 / sdrekf.rk1_v[prn - 1];
    }
    unmlock(hobsvecmtx);
}

static int solveWlsIteration(const double *satPos, const double *pr, int numSat,
                             double *state, const double *weights, int iterNum,
                             double *normDx, LsqMatrices *mat) {
    double *designArr = (double *)calloc(numSat * STATE_DIM, sizeof(double));
    double *omcArr = (double *)calloc(numSat, sizeof(double));
    for (int i = 0; i < numSat; i++) {
        double rotSat[3];
        double trop = 2.0;
        if (iterNum == 0) {
            rotSat[0] = satPos[i * 3 + 0];
            rotSat[1] = satPos[i * 3 + 1];
            rotSat[2] = satPos[i * 3 + 2];
        } else {
            double dx = satPos[i * 3 + 0] - state[0];
            double dy = satPos[i * 3 + 1] - state[1];
            double dz = satPos[i * 3 + 2] - state[2];
            double rho = sqrt(SQ(dx) + SQ(dy) + SQ(dz));
            applyEarthRotation(&satPos[i * 3], rho / CTIME, rotSat);
            double refPos[3] = {state[0], state[1], state[2]};
            double delta[3] = {rotSat[0] - state[0], rotSat[1] - state[1],
                               rotSat[2] - state[2]};
            TopocentricCoord topo = {0, 0, 0};
            ecefToTopocentric(refPos, delta, &topo);
            sdrstat.azElCalculatedflag = 1;
            mlock(hobsvecmtx);
            int prn = sdrstat.obsValidList[i];
            if (prn >= 1 && prn <= MAXSAT) {
                sdrstat.obs_v[OBS_IDX(prn, 2)] = rotSat[0];
                sdrstat.obs_v[OBS_IDX(prn, 3)] = rotSat[1];
                sdrstat.obs_v[OBS_IDX(prn, 4)] = rotSat[2];
                sdrstat.obs_v[OBS_IDX(prn, 9)] = topo.azimuth;
                sdrstat.obs_v[OBS_IDX(prn, 10)] = topo.elevation;
            }
            unmlock(hobsvecmtx);
            estimateAtmosphericDelay(sin(topo.elevation * D2R), 0.0, 1013.0,
                                     293.0, 50.0, 0.0, 0.0, 0.0, &trop);
        }
        double dx = rotSat[0] - state[0];
        double dy = rotSat[1] - state[1];
        double dz = rotSat[2] - state[2];
        double range = sqrt(SQ(dx) + SQ(dy) + SQ(dz));
        omcArr[i] = pr[i] - range - state[3] - trop;
        designArr[i * 4 + 0] = -dx / range;
        designArr[i * 4 + 1] = -dy / range;
        designArr[i * 4 + 2] = -dz / range;
        designArr[i * 4 + 3] = 1.0;
    }
    mat->weight =
        nml_mat_from(numSat, numSat, numSat * numSat, (double *)weights);
    mat->design =
        nml_mat_from(numSat, STATE_DIM, numSat * STATE_DIM, designArr);
    mat->designT = nml_mat_transp(mat->design);
    mat->ata = nml_mat_dot(mat->designT, mat->design);
    mat->atw = nml_mat_dot(mat->designT, mat->weight);
    mat->atwa = nml_mat_dot(mat->atw, mat->design);
    nml_mat_lup *lupAta = nml_mat_lup_solve(mat->ata);
    nml_mat_lup *lupAtwa = nml_mat_lup_solve(mat->atwa);
    double det = nml_mat_det(lupAta);
    if (fabs(det) < DETERMINANT_TOL) {
        printf("WLS solver abort: singular matrix (det=%lf)\n", det);
        nml_mat_lup_free(lupAta);
        nml_mat_lup_free(lupAtwa);
        free(designArr);
        free(omcArr);
        return -1;
    }
    mat->invAta = nml_mat_inv(lupAta);
    mat->invAtwa = nml_mat_inv(lupAtwa);
    nml_mat_lup_free(lupAta);
    nml_mat_lup_free(lupAtwa);
    mat->gain1 = nml_mat_dot(mat->invAtwa, mat->designT);
    mat->gain2 = nml_mat_dot(mat->gain1, mat->weight);
    mat->residual = nml_mat_from(numSat, 1, numSat, omcArr);
    mat->solution = nml_mat_dot(mat->gain2, mat->residual);
    state[0] += mat->solution->data[0][0];
    state[1] += mat->solution->data[1][0];
    state[2] += mat->solution->data[2][0];
    state[3] += mat->solution->data[3][0];
    *normDx =
        sqrt(SQ(mat->solution->data[0][0]) + SQ(mat->solution->data[1][0]) +
             SQ(mat->solution->data[2][0]) + SQ(mat->solution->data[3][0]));
    mlock(hobsvecmtx);
    memset(sdrstat.vk1_v, 0, MAXSAT * sizeof(double));
    for (int j = 0; j < numSat; j++) {
        int prn = sdrstat.obsValidList[j];
        if (prn >= 1 && prn <= MAXSAT) {
            sdrstat.vk1_v[prn - 1] = omcArr[j];
        }
    }
    unmlock(hobsvecmtx);
    free(designArr);
    free(omcArr);
    return 0;
}

extern int runWeightedLeastSquares(double *X_v, double *pr_v, int numSat,
                                   double xyzdt_v[], double *gdop) {
    numSat = sdrstat.nsatValid;
    double *weightArr = (double *)calloc(numSat * numSat, sizeof(double));
    initWeightMatrix(numSat, weightArr);
    double state[STATE_DIM] = {(double)sdrini.xu0_v[0], (double)sdrini.xu0_v[1],
                               (double)sdrini.xu0_v[2], 0.0};
    LsqMatrices *matrices = initLsqMatrices(numSat);
    if (!matrices) {
        free(weightArr);
        return -1;
    }
    double normDx = 100.0;
    int status = 0;
    for (int iter = 0; iter < MAX_ITER_POSITION && normDx > CONVERGENCE_TOL;
         iter++) {
        status = solveWlsIteration(X_v, pr_v, numSat, state, weightArr, iter,
                                   &normDx, matrices);
        if (status != 0)
            break;
    }
    if (status == 0) {
        xyzdt_v[0] = state[0];
        xyzdt_v[1] = state[1];
        xyzdt_v[2] = state[2];
        xyzdt_v[3] = state[3];
        *gdop =
            sqrt(matrices->invAta->data[0][0] + matrices->invAta->data[1][1] +
                 matrices->invAta->data[2][2] + matrices->invAta->data[3][3]);
    } else {
        memset(xyzdt_v, 0, STATE_DIM * sizeof(double));
        *gdop = 0.0;
    }
    freeLsqMatrices(matrices);
    free(weightArr);
    return status;
}

static int validateEphemerisGps(int prn) {
    double tol = 1e-15;
    eph_t *e = &sdrch[prn - 1].nav.sdreph.eph;
    return (e->toes >= 1.0) && (fabs(e->A) >= tol) && (fabs(e->e) >= tol) &&
           (fabs(e->M0) >= tol) && (fabs(e->omg) >= tol) &&
           (fabs(e->i0) >= tol) && (fabs(e->OMG0) >= tol) &&
           (fabs(e->deln) >= tol) && (fabs(e->idot) >= tol) &&
           (fabs(e->OMGd) >= tol) && (fabs(e->cuc) >= tol) &&
           (fabs(e->cus) >= tol) && (fabs(e->crc) >= tol) &&
           (fabs(e->crs) >= tol) && (fabs(e->cic) >= tol) &&
           (fabs(e->cis) >= tol) && (fabs(e->f0) >= tol) &&
           (fabs(e->f1) >= tol) && (fabs(e->tgd[0]) >= tol);
}

static int validateEphemerisGlo(int prn) {
    double tol = 1e-15;
    geph_t *g = &sdrch[prn - 1].nav.sdreph.geph;
    return (g->toe.time != 0) && (fabs(g->pos[0]) >= tol) &&
           (fabs(g->pos[1]) >= tol) && (fabs(g->pos[2]) >= tol) &&
           (fabs(g->vel[0]) >= tol) && (fabs(g->vel[1]) >= tol) &&
           (fabs(g->vel[2]) >= tol);
}

static void markSatelliteInvalid(int prn, const char *reason) {
    char buffer[MSG_LENGTH];
    char satId[8] = {0};
    snprintf(satId, sizeof(satId), "PRN%02d", prn);
    if (prn >= 1 && prn <= sdrini.nch && sdrch[prn - 1].sat > 0) {
        satno2id(sdrch[prn - 1].sat, satId);
    }
    sdrstat.obs_v[OBS_IDX(prn, 1)] = 0;
    snprintf(buffer, sizeof(buffer), "%.3f  validateMeas: %s %s\n",
             sdrstat.elapsedTime, satId, reason);
    add_message(buffer);
}

extern void validateMeasurements(void) {
    int needUpdate = 0;
    char reasonBuf[64];
    mlock(hobsvecmtx);
    for (int i = 0; i < sdrstat.nsatValid; i++) {
        int prn = sdrstat.obsValidList[i];
        if (prn < 1 || prn > MAXSAT || prn > sdrini.nch)
            continue;
        double snr = sdrstat.obs_v[OBS_IDX(prn, 8)];
        double week = sdrstat.obs_v[OBS_IDX(prn, 7)];
        double tow = sdrstat.obs_v[OBS_IDX(prn, 6)];
        double pr = sdrstat.obs_v[OBS_IDX(prn, 5)];
        double elev = sdrstat.obs_v[OBS_IDX(prn, 10)];
        int ctype = sdrch[prn - 1].nav.sdreph.ctype;
        if (snr < SNR_PVT_THRES) {
            snprintf(reasonBuf, sizeof(reasonBuf), "has SNR:%.1f", snr);
            markSatelliteInvalid(prn, reasonBuf);
            needUpdate = 1;
            continue;
        }
        if (week < GPS_WEEK) {
            snprintf(reasonBuf, sizeof(reasonBuf), "has Week:%d", (int)week);
            markSatelliteInvalid(prn, reasonBuf);
            needUpdate = 1;
            continue;
        }
        if (tow < 1.0) {
            snprintf(reasonBuf, sizeof(reasonBuf), "has ToW:%.1f", tow);
            markSatelliteInvalid(prn, reasonBuf);
            needUpdate = 1;
            continue;
        }
        if (pr < LOW_PR) {
            snprintf(reasonBuf, sizeof(reasonBuf), "has Low PR:%.1f", pr);
            markSatelliteInvalid(prn, reasonBuf);
            needUpdate = 1;
            continue;
        }
        if (pr > HIGH_PR) {
            snprintf(reasonBuf, sizeof(reasonBuf), "has High PR:%.1f", pr);
            markSatelliteInvalid(prn, reasonBuf);
            needUpdate = 1;
            continue;
        }
        if ((ctype == CTYPE_L1CA || ctype == CTYPE_E1B) &&
            !validateEphemerisGps(prn)) {
            markSatelliteInvalid(prn, "tagged for removal for eph error");
            needUpdate = 1;
            continue;
        }
        if (ctype == CTYPE_G1 && !validateEphemerisGlo(prn)) {
            char buf[64];
            snprintf(buf, sizeof(buf), "R%02d tagged for removal for eph error",
                     prn);
            sdrstat.obs_v[OBS_IDX(prn, 1)] = 0;
            char msg[MSG_LENGTH];
            snprintf(msg, sizeof(msg), "%.3f  ephValidation: %s\n",
                     sdrstat.elapsedTime, buf);
            add_message(msg);
            needUpdate = 1;
            continue;
        }
        if (sdrstat.azElCalculatedflag && elev < SV_EL_PVT_MASK) {
            snprintf(reasonBuf, sizeof(reasonBuf),
                     "tagged for removal with el of %.1f", elev);
            markSatelliteInvalid(prn, reasonBuf);
            needUpdate = 1;
        }
    }
    unmlock(hobsvecmtx);
    if (needUpdate) {
        if (refreshValidSatellites() != 0) {
            printf("refreshValidSatellites: error\n");
        }
    }
}

extern int refreshValidSatellites(void) {
    mlock(hobsvecmtx);
    sdrstat.nsatValid = 0;
    memset(sdrstat.obsValidList, 0, MAXSAT * sizeof(int));
    for (int i = 0; i < MAXSAT; i++) {
        if (sdrstat.obs_v[OBS_IDX(i + 1, 1)] != 1.0)
            continue;
        int prn = (int)sdrstat.obs_v[OBS_IDX(i + 1, 0)];
        if (prn >= 1 && prn <= MAXSAT && prn <= sdrini.nch &&
            sdrstat.nsatValid < MAXSAT) {
            sdrstat.obsValidList[sdrstat.nsatValid++] = prn;
        }
    }
    unmlock(hobsvecmtx);
    return 0;
}

static int computeSatellitePositions(PvtBuffers *buf, int numSat,
                                     double rcvrTow) {
    SatelliteData satData;
    for (int i = 0; i < numSat; i++) {
        mlock(hobsvecmtx);
        double prEstimate = buf->rawPr[i] - sdrstat.xyzdt[3];
        unmlock(hobsvecmtx);
        double tau = prEstimate / CTIME;
        double txTime = rcvrTow - tau;
        mlock(hobsvecmtx);
        int prn = sdrstat.obsValidList[i];
        sdreph_t *eph = &sdrch[prn - 1].nav.sdreph;
        unmlock(hobsvecmtx);
        int ret;
        if (eph->ctype == CTYPE_G1) {
            ret = computeGlonassPosition(&eph->geph, txTime, &satData);
        } else {
            ret = computeKeplerianPosition(&eph->eph, txTime, &satData);
        }
        if (ret != 0 || isnan(satData.pos[0])) {
            printf("SV coordinate computation failed (NaN) for G%02d\n", prn);
            return -1;
        }
        buf->corrPr[i] = buf->rawPr[i] + CTIME * satData.clkCorr;
        buf->satPositions[i * 3 + 0] = satData.pos[0];
        buf->satPositions[i * 3 + 1] = satData.pos[1];
        buf->satPositions[i * 3 + 2] = satData.pos[2];
        mlock(hobsvecmtx);
        sdrstat.obs_v[OBS_IDX(prn, 2)] = satData.pos[0];
        sdrstat.obs_v[OBS_IDX(prn, 3)] = satData.pos[1];
        sdrstat.obs_v[OBS_IDX(prn, 4)] = satData.pos[2];
        unmlock(hobsvecmtx);
    }
    return 0;
}

static void storePositionResult(double x, double y, double z, double clkBias,
                                double gdop) {
    GeodeticCoord geo;
    double ecef[3] = {x, y, z};
    ecefToGeodetic(ecef, &geo);
    mlock(hobsvecmtx);
    sdrstat.lat = geo.lat * 180.0 / M_PI;
    sdrstat.lon = geo.lon * 180.0 / M_PI;
    sdrstat.hgt = geo.alt;
    sdrstat.gdop = gdop;
    sdrstat.xyzdt[0] = x;
    sdrstat.xyzdt[1] = y;
    sdrstat.xyzdt[2] = z;
    sdrstat.xyzdt[3] = clkBias;
    unmlock(hobsvecmtx);
}

static void clearPositionResult(void) {
    mlock(hobsvecmtx);
    sdrstat.lat = 0.0;
    sdrstat.lon = 0.0;
    sdrstat.hgt = 0.0;
    sdrstat.gdop = 0.0;
    unmlock(hobsvecmtx);
}

extern int executeNavigationSolution(void) {
    mlock(hobsvecmtx);
    int numSat = sdrstat.nsatValid;
    unmlock(hobsvecmtx);
    PvtBuffers *buf = allocatePvtBuffers(numSat);
    if (!buf)
        return -1;
    mlock(hobsvecmtx);
    for (int i = 0; i < numSat; i++) {
        int prn = sdrstat.obsValidList[i];
        buf->rawPr[i] = sdrstat.obs_v[OBS_IDX(prn, 5)];
        buf->rcvrTow[i] = sdrstat.obs_v[OBS_IDX(prn, 6)];
        buf->snr[i] = sdrstat.obs_v[OBS_IDX(prn, 8)];
    }
    unmlock(hobsvecmtx);
    double rcvrTow = buf->rcvrTow[0];
    if (computeSatellitePositions(buf, numSat, rcvrTow) != 0) {
        clearPositionResult();
        freePvtBuffers(buf);
        return -1;
    }
    if (sdrstat.nsatValid < 4) {
        clearPositionResult();
        freePvtBuffers(buf);
        return -1;
    }
    double state[STATE_DIM] = {0};
    double gdop = 0.0;
    int ret = runWeightedLeastSquares(buf->satPositions, buf->corrPr, numSat,
                                      state, &gdop);
    if (ret != 0 || isnan(state[0]) || isnan(state[1]) || isnan(state[2])) {
        printf("Navigation solution diverged (NaN in receiver position)\n");
        clearPositionResult();
        freePvtBuffers(buf);
        return -1;
    }
    storePositionResult(state[0], state[1], state[2], state[3], gdop);
    freePvtBuffers(buf);
    return 0;
}
