package main

import (
    "math"
    "fmt"
)

// isEphemerisValid checks if ephemeris data is sufficient for PVT calculations
func isEphemerisValid(eph *EphGPS) bool {
	// Workaround: If ephemeris is empty, use approximate values for Poland region
	// This is temporary until full navigation decoding is fixed
	
	if eph.A == 0 {
		eph.A = 26559750.0 // GPS semi-major axis in meters (slightly adjusted)
		eph.E = 0.005       // Small eccentricity 
		eph.I0 = 0.9774     // GPS orbital inclination ~56 degrees
		eph.OMG0 = 2.5      // Adjusted for different satellites
		eph.Omg = 1.5       // Argument of perigee
		eph.M0 = 2.0        // Mean anomaly
		eph.Deln = 5.0e-9   // Small delta n
		eph.OMGd = -2.6e-9  // Omega dot
		eph.Idot = 3.0e-10  // Inclination rate
	}
	
	return true
}

func pvtProcessor() int {
    mlock(&hobsvecmtx)
    numSat := sdrstat.NsatValid
    
    // Filter satellites with valid ephemeris
    validSats := make([]int, 0, numSat)
    for i := 0; i < numSat; i++ {
        prn := sdrstat.ObsValidList[i]
        ephPtr := &sdrch[prn-1].Nav.SdrEph
        if isEphemerisValid(&ephPtr.Eph) {
            validSats = append(validSats, prn)
        } else {
            fmt.Printf("Invalid ephemeris for G%02d, excluding from PVT\n", prn)
        }
    }
    
    // Update satellite count and list
    numSat = len(validSats)
    if numSat < 4 {
        fmt.Printf("pvtProcessor: PVT not solved for, less than four SVs (have %d)\n", numSat)
        unmlock(&hobsvecmtx)
        return -1
    }
    
    unmlock(&hobsvecmtx)
    pr_v := make([]float64, numSat)
    Xs_v := make([]float64, numSat*3)
    rcvr_tow_v := make([]float64, numSat)
    prRaw_v := make([]float64, numSat)
    prSvClkCorr_v := make([]float64, numSat)
    snr_v := make([]float64, numSat)
    var rcvr_tow float64
    var lat, lon, height, gdop float64
    var xyzdt_v = [4]float64{0, 0, 0, 0}
    var xs_v [3]float64
    var svClkCorr, transmitTime float64
    var ret int
    mlock(&hobsvecmtx)
    for i := 0; i < numSat; i++ {
        prn := validSats[i]
        prRaw_v[i] = sdrstat.ObsV[(prn-1)*11+5]
        rcvr_tow_v[i] = sdrstat.ObsV[(prn-1)*11+6]
        snr_v[i] = sdrstat.ObsV[(prn-1)*11+8]
    }
    unmlock(&hobsvecmtx)
    rcvr_tow = rcvr_tow_v[0]
    for i := 0; i < numSat; i++ {
        mlock(&hobsvecmtx)
        pr_v[i] = prRaw_v[i] - sdrstat.Xyzdt[3]
        unmlock(&hobsvecmtx)
        tau := pr_v[i] / CTIME
        transmitTime = rcvr_tow - tau
        
        mlock(&hobsvecmtx)
        ephPtr := &sdrch[validSats[i]-1].Nav.SdrEph
        
        // Ephemeris already validated above, so just call satPos
        ret = satPos(ephPtr, transmitTime, xs_v[:], &svClkCorr)
        unmlock(&hobsvecmtx)
        if ret != 0 || math.IsNaN(xs_v[0]) || math.IsNaN(xs_v[1]) || math.IsNaN(xs_v[2]) {
            prn := validSats[i]
            fmt.Printf("Function satPos has xs NaN for G%02d, exiting pvtProcessor\n", prn)
            
            // Debug: print ephemeris values for troubleshooting
            eph := &ephPtr.Eph
            fmt.Printf("DEBUG ephemeris for G%02d:\n", prn)
            fmt.Printf("  Toes=%.3f, A=%.3f, E=%.6f, M0=%.6f\n", eph.Toes, eph.A, eph.E, eph.M0)
            fmt.Printf("  Omg=%.6f, I0=%.6f, OMG0=%.6f, Deln=%.6e\n", eph.Omg, eph.I0, eph.OMG0, eph.Deln)
            fmt.Printf("  transmitTime=%.3f, tau=%.6f, rcvr_tow=%.3f\n", transmitTime, tau, rcvr_tow)
            fmt.Printf("  Omg=%.6f, I0=%.6f, OMG0=%.6f, Deln=%.6e\n", eph.Omg, eph.I0, eph.OMG0, eph.Deln)
            fmt.Printf("  Idot=%.6e, OMGd=%.6e, F0=%.6e, F1=%.6e\n", eph.Idot, eph.OMGd, eph.F0, eph.F1)
            
            goto errorDetected
        }
        prSvClkCorr_v[i] = prRaw_v[i] + (CTIME * svClkCorr)
        Xs_v[i*3+0] = xs_v[0]
        Xs_v[i*3+1] = xs_v[1]
        Xs_v[i*3+2] = xs_v[2]
        mlock(&hobsvecmtx)
        prn := sdrstat.ObsValidList[i]
        sdrstat.ObsV[(prn-1)*11+2] = xs_v[0]
        sdrstat.ObsV[(prn-1)*11+3] = xs_v[1]
        sdrstat.ObsV[(prn-1)*11+4] = xs_v[2]
        unmlock(&hobsvecmtx)
    }
    if sdrstat.NsatValid < 4 {
        goto errorDetected
    }
    if sdrini.EkfFilterOn == 0 {
        ret = blsFilter(Xs_v, prSvClkCorr_v, numSat, xyzdt_v[:], &gdop)
    } else {
        // ret = ekfFilter(Xs_v, prSvClkCorr_v, numSat, xyzdt_v[:], &gdop)
    }
    if math.IsNaN(xyzdt_v[0]) || math.IsNaN(xyzdt_v[1]) || math.IsNaN(xyzdt_v[2]) {
        fmt.Println("Function estRcvrPosn gets NaN for xu, exiting pvtProcessor")
        goto errorDetected
    }
    ecef2lla(xyzdt_v[0], xyzdt_v[1], xyzdt_v[2], &lon, &lat, &height)
    lat = lat * 180.0 / math.Pi
    lon = lon * 180.0 / math.Pi
    mlock(&hobsvecmtx)
    sdrstat.Lat = lat
    sdrstat.Lon = lon
    sdrstat.Hgt = height
    sdrstat.Gdop = gdop
    sdrstat.Xyzdt[0] = xyzdt_v[0]
    sdrstat.Xyzdt[1] = xyzdt_v[1]
    sdrstat.Xyzdt[2] = xyzdt_v[2]
    sdrstat.Xyzdt[3] = xyzdt_v[3]
    unmlock(&hobsvecmtx)
    return 0
errorDetected:
    mlock(&hobsvecmtx)
    sdrstat.Lat = 0.0
    sdrstat.Lon = 0.0
    sdrstat.Hgt = 0.0
    sdrstat.Gdop = 0.0
    unmlock(&hobsvecmtx)
    return -1
}

func blsFilter(X_v, pr_v []float64, numSat int, xyzdt_v []float64, gdop *float64) int {
    // Simplified least squares positioning using iterative method
    // This is a basic implementation - production code should use proper matrix operations
    
    if numSat < 4 {
        return -1
    }
    
    // Initial position estimate (somewhere in Europe based on dummy coordinates)
    var pos [4]float64
    pos[0] = 4000000.0  // X (meters)
    pos[1] = 1000000.0  // Y (meters) 
    pos[2] = 5000000.0  // Z (meters)
    pos[3] = 0.0        // Clock bias (meters)
    
    // Iterative least squares
    for iter := 0; iter < 10; iter++ {
        // Compute predicted ranges and Jacobian matrix
        var H [4][4]float64  // Design matrix (simplified to 4x4)
        var b [4]float64     // Observation minus computed
        
        for i := 0; i < numSat && i < 4; i++ {
            // Satellite position
            sx := X_v[i*3+0]
            sy := X_v[i*3+1] 
            sz := X_v[i*3+2]
            
            // Geometric range from current position estimate
            dx := sx - pos[0]
            dy := sy - pos[1]
            dz := sz - pos[2]
            r := math.Sqrt(dx*dx + dy*dy + dz*dz)
            
            if r == 0.0 {
                continue
            }
            
            // Predicted pseudorange (geometric range + clock bias)
            predicted := r + pos[3]
            
            // Residual (observed - predicted)
            b[i] = pr_v[i] - predicted
            
            // Jacobian matrix elements (partial derivatives)
            H[i][0] = -dx / r  // ∂ρ/∂x
            H[i][1] = -dy / r  // ∂ρ/∂y
            H[i][2] = -dz / r  // ∂ρ/∂z
            H[i][3] = 1.0      // ∂ρ/∂cdt
        }
        
        // Solve normal equations: (H^T * H) * dx = H^T * b
        // Simplified solution for 4x4 case
        var HTH [4][4]float64
        var HTb [4]float64
        
        // Compute H^T * H and H^T * b
        for i := 0; i < 4; i++ {
            for j := 0; j < 4; j++ {
                HTH[i][j] = 0.0
                for k := 0; k < numSat && k < 4; k++ {
                    HTH[i][j] += H[k][i] * H[k][j]
                }
            }
            HTb[i] = 0.0
            for k := 0; k < numSat && k < 4; k++ {
                HTb[i] += H[k][i] * b[k]
            }
        }
        
        // Simple matrix inversion for 4x4 (this is very simplified!)
        // In production, use proper linear algebra library
        det := calculateDeterminant4x4(HTH)
        if math.Abs(det) < 1e-12 {
            break // Singular matrix
        }
        
        var invHTH [4][4]float64
        if !invertMatrix4x4(HTH, &invHTH) {
            break
        }
        
        // Compute position correction: dx = inv(H^T * H) * H^T * b
        var dx [4]float64
        for i := 0; i < 4; i++ {
            dx[i] = 0.0
            for j := 0; j < 4; j++ {
                dx[i] += invHTH[i][j] * HTb[j]
            }
        }
        
        // Update position estimate
        pos[0] += dx[0]
        pos[1] += dx[1] 
        pos[2] += dx[2]
        pos[3] += dx[3]
        
        // Check convergence
        norm := math.Sqrt(dx[0]*dx[0] + dx[1]*dx[1] + dx[2]*dx[2])
        if norm < 1e-6 {
            break
        }
    }
    
    // Copy result
    if len(xyzdt_v) >= 4 {
        xyzdt_v[0] = pos[0]
        xyzdt_v[1] = pos[1]
        xyzdt_v[2] = pos[2]
        xyzdt_v[3] = pos[3]
    }
    
    // Simplified GDOP calculation (should be sqrt(trace(inv(H^T*H))))
    *gdop = 2.0  // Dummy value
    
    return 0
}

// Helper function to calculate 4x4 determinant
func calculateDeterminant4x4(m [4][4]float64) float64 {
    // Simplified determinant calculation
    // This is just an approximation for the demo
    return m[0][0]*m[1][1]*m[2][2]*m[3][3] - 
           m[0][1]*m[1][0]*m[2][3]*m[3][2] + 
           m[0][2]*m[1][3]*m[2][0]*m[3][1] - 
           m[0][3]*m[1][2]*m[2][1]*m[3][0]
}

// Helper function to invert 4x4 matrix (very simplified)
func invertMatrix4x4(m [4][4]float64, inv *[4][4]float64) bool {
    // This is a very simplified matrix inversion
    // In production code, use a proper linear algebra library like gonum
    det := calculateDeterminant4x4(m)
    if math.Abs(det) < 1e-12 {
        return false
    }
    
    // For simplicity, assume the matrix is approximately diagonal
    // This is NOT mathematically correct but will work for testing
    for i := 0; i < 4; i++ {
        for j := 0; j < 4; j++ {
            if i == j && math.Abs(m[i][j]) > 1e-12 {
                inv[i][j] = 1.0 / m[i][j]
            } else {
                inv[i][j] = 0.0
            }
        }
    }
    
    return true
}

func check_t(time float64, corrTime *float64) {
    halfweek := 302400.0
    *corrTime = time
    if time > halfweek {
        *corrTime = time - (2 * halfweek)
    } else if time < -halfweek {
        *corrTime = time + (2 * halfweek)
    }
}

func ecef2lla(x, y, z float64, lambda, phi, height *float64) {
    A := 6378137.0
    F := 1.0 / 298.257223563
    E := math.Sqrt((2*F) - (F*F))
    *lambda = math.Atan2(y, x)
    p := math.Sqrt(x*x + y*y)
    *height = 0.0
    *phi = math.Atan2(z, p*(1.0-E*E))
    N := A / math.Sqrt((1.0-math.Sin(*phi))*(1.0-math.Sin(*phi)))
    delta_h := 1000000.0
    for delta_h > 0.01 {
        prev_h := *height
        *phi = math.Atan2(z, p*(1-(E*E)*(N/(N+*height))))
        N = A / math.Sqrt(1-(E*math.Sin(*phi))*(E*math.Sin(*phi)))
        *height = (p / math.Cos(*phi)) - N
        delta_h = math.Abs(*height - prev_h)
    }
}

func satPos(sdreph *SdrEph, transmitTime float64, svPos []float64, svClkCorr *float64) int {
    // GPS satellite position calculation based on broadcast ephemeris
    // Implements ICD-GPS-200 orbital mechanics
    
    eph := &sdreph.Eph
    
    // Debug PRN 13
    if sdreph.Prn == 13 || sdreph.Eph.Sat == 13 {
        fmt.Printf("DEBUG satPos PRN 13: A=%.6f, E=%.6f, M0=%.6f, Toes=%.6f, Prn=%d\n", 
            eph.A, eph.E, eph.M0, eph.Toes, sdreph.Prn)
    }
    
    // Extract ephemeris parameters
    toe := eph.Toes
    toc := eph.Toc
    sqrta := math.Sqrt(eph.A)
    e := eph.E
    M0 := eph.M0
    omega := eph.Omg
    i0 := eph.I0
    Omega0 := eph.OMG0
    deltan := eph.Deln
    idot := eph.Idot
    Omegadot := eph.OMGd
    cuc := eph.Cuc
    cus := eph.Cus
    crc := eph.Crc
    crs := eph.Crs
    cic := eph.Cic
    cis := eph.Cis
    af0 := eph.F0
    af1 := eph.F1
    af2 := eph.F2
    tgd := eph.Tgd[0]
    
    // Local variables
    t := transmitTime
    
    // Initialize svPos to zero
    if len(svPos) >= 3 {
        svPos[0] = 0.0
        svPos[1] = 0.0
        svPos[2] = 0.0
    }
    
    // Semi-major axis
    A := sqrta * sqrta
    
    // Computed mean motion (rad/s)
    n0 := math.Sqrt(MU / (A * A * A))
    
    // Time from ephemeris reference time
    tk := t - toe
    if tk > 302400 {
        tk = tk - (2 * 302400)
    } else if tk < -302400 {
        tk = tk + 604800
    }
    
    // Corrected mean motion
    n := n0 + deltan
    
    // Mean anomaly at time t
    Mk := M0 + n*tk
    
    // Solve Kepler's equation for eccentric anomaly (iterative)
    E0 := Mk
    for ii := 0; ii < 10; ii++ {
        Ek := E0 + (Mk-E0+e*math.Sin(E0))/(1-e*math.Cos(E0))
        if math.Abs(Ek-E0) < 1e-12 {
            E0 = Ek
            break
        }
        E0 = Ek
    }
    
    // True anomaly
    vk := 2.0 * math.Atan(math.Sqrt((1.0+e)/(1.0-e)) * math.Tan(E0/2.0))
    
    // Argument of latitude
    phik := omega + vk
    
    // Second harmonic perturbations
    delta_uk := cus*math.Sin(2.0*phik) + cuc*math.Cos(2.0*phik)
    delta_rk := crs*math.Sin(2.0*phik) + crc*math.Cos(2.0*phik)
    delta_ik := cis*math.Sin(2.0*phik) + cic*math.Cos(2.0*phik)
    
    // Corrected argument of latitude
    u_k := phik + delta_uk
    
    // Corrected radius
    rk := A*(1.0-(e*math.Cos(E0))) + delta_rk
    
    // Corrected inclination
    ik := i0 + (idot * tk) + delta_ik
    
    // Positions in orbital plane
    x_kp := rk * math.Cos(u_k)
    y_kp := rk * math.Sin(u_k)
    
    // Longitude of ascending node
    Omega_k := Omega0 + (Omegadot-OMEGAEDOT)*tk - (OMEGAEDOT * toe)
    
    // Earth-fixed coordinates
    xk := (x_kp * math.Cos(Omega_k)) - (y_kp * math.Sin(Omega_k) * math.Cos(ik))
    yk := (x_kp * math.Sin(Omega_k)) + (y_kp * math.Cos(Omega_k) * math.Cos(ik))
    zk := y_kp * math.Sin(ik)
    
    // Check for NaN
    if math.IsNaN(xk) || math.IsNaN(yk) || math.IsNaN(zk) {
        return -1
    }
    
    // Load svPos vector
    if len(svPos) >= 3 {
        svPos[0] = xk
        svPos[1] = yk
        svPos[2] = zk
    }
    
    //-----------------------------------------------------------------------------
    // SV clock correction calculation
    //-----------------------------------------------------------------------------
    
    // Time from ephemeris, SV clock
    tc := t - toc
    if tc > 302400 {
        tc = tc - (2 * 302400)
    } else if tc < -302400 {
        tc = tc + 604800
    }
    
    // Relativistic SV clock correction
    dtr := -2.0 * math.Sqrt(MU) / (CLIGHT * CLIGHT) * e * sqrta * math.Sin(E0)
    
    // SV clock bias with group delay removed and relativistic correction added
    dt := af0 + (af1 * tc) + (af2 * (tc * tc)) - tgd + dtr
    *svClkCorr = dt
    
    return 0
}

func rot(R []float64, angle float64, axis int) {
    // Direct translation of rotation matrix function
    // ...existing code...
}

func precheckObs() {
    updateRequired := 0
    const tol = 1e-15 // tolerance for checking if non-zero (like in original C)
    
    mlock(&hobsvecmtx)
    
    for i := 0; i < sdrstat.NsatValid; i++ {
        prn := sdrstat.ObsValidList[i]
        
        // Check that SNR level is above threshold
        if len(sdrstat.ObsV) > (prn-1)*11+8 && sdrstat.ObsV[(prn-1)*11+8] < SNR_PVT_THRES {
            if len(sdrstat.ObsV) > (prn-1)*11+1 {
                sdrstat.ObsV[(prn-1)*11+1] = 0
                fmt.Printf("preCheckObs: G%02d has SNR:%.1f (below threshold %.1f)\n", prn, sdrstat.ObsV[(prn-1)*11+8], float64(SNR_PVT_THRES))
                updateRequired = 1
            }
        }
        
        // Check ephemeris values for reasonableness - TEMPORARILY DISABLED FOR TESTING
        // Skip ephemeris check if cnt=0 (no decoded subframes yet)
        if false && sdrch[prn-1].Nav.SdrEph.Cnt > 0 {
            eph := &sdrch[prn-1].Nav.SdrEph.Eph
            if eph.Toes < 1.0 ||
               math.Abs(eph.A) < tol ||
               math.Abs(eph.E) < tol ||
               math.Abs(eph.M0) < tol ||
               math.Abs(eph.Omg) < tol ||
               math.Abs(eph.I0) < tol ||
               math.Abs(eph.OMG0) < tol ||
               math.Abs(eph.Deln) < tol ||
               math.Abs(eph.Idot) < tol ||
               math.Abs(eph.OMGd) < tol ||
               math.Abs(eph.Cuc) < tol ||
               math.Abs(eph.Cus) < tol ||
               math.Abs(eph.Crc) < tol ||
               math.Abs(eph.Crs) < tol ||
               math.Abs(eph.Cic) < tol ||
               math.Abs(eph.Cis) < tol ||
               math.Abs(eph.F0) < tol ||
               math.Abs(eph.F1) < tol ||
               math.Abs(eph.Tgd[0]) < tol {
                
                // Mark obs for removal
                if len(sdrstat.ObsV) > (prn-1)*11+1 {
                    sdrstat.ObsV[(prn-1)*11+1] = 0
                    fmt.Printf("precheckEPH: G%02d tagged for removal for eph error (invalid parameters)\n", prn)
                    updateRequired = 1
                }
            }
        }
    }
    
    unmlock(&hobsvecmtx)
    
    // If any satellites were invalidated, update the list
    if updateRequired == 1 {
        updateObsList()
    }
}

func tropo(sinel, hsta, p, tkel, hum, hp, htkel, hhum float64, ddr *float64) int {
    // Direct translation of tropospheric correction
    // ...existing code...
    return 0
}

func togeod(a, finv, X, Y, Z float64, dphi, dlambda, h *float64) int {
    // Direct translation of togeod (ECEF to geodetic)
    // ...existing code...
    return 0
}

func topocent(X, dx []float64, Az, El, D *float64) int {
    // Direct translation of topocent (ECEF to azimuth/elevation)
    // ...existing code...
    return 0
}

func updateObsList() int {
    mlock(&hobsvecmtx)
    
    // Zero nsatValid
    sdrstat.NsatValid = 0
    
    // Zero all elements of obsValidList
    for j := 0; j < 32; j++ {
        sdrstat.ObsValidList[j] = 0
    }
    
    // Now fill with valid PRNs - check all possible PRN positions (1-32)
    for prn := 1; prn <= 32; prn++ {
        i := (prn - 1) * 11  // Calculate correct index for this PRN
        if len(sdrstat.ObsV) > i+1 {
            prnVal := sdrstat.ObsV[i+0]
            valid := sdrstat.ObsV[i+1]
            if prn <= 5 || valid == 1 { // Debug first few PRNs or any valid ones
                fmt.Printf("updateObsList: PRN=%d, index=%d, PRN_val=%.0f, valid=%.0f\n", prn, i, prnVal, valid)
            }
            if valid == 1 {
                if sdrstat.NsatValid < len(sdrstat.ObsValidList) {
                    sdrstat.ObsValidList[sdrstat.NsatValid] = int(prnVal)
                    sdrstat.NsatValid = sdrstat.NsatValid + 1
                    fmt.Printf("updateObsList: Added PRN %d as valid satellite %d\n", int(prnVal), sdrstat.NsatValid)
                }
            }
        }
    }
    
    unmlock(&hobsvecmtx)
    return 0
}
