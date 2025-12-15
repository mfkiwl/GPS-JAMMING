#!/usr/bin/env python3

import json
import time
import numpy as np
from collections import deque
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass, field
from datetime import datetime

CTIME = 299792458.0
GPS_L1_FREQ = 1575.42e6
GPS_L1_WAVELENGTH = CTIME / GPS_L1_FREQ
GALILEO_E1_FREQ = 1575.42e6
GALILEO_E1_WAVELENGTH = CTIME / GALILEO_E1_FREQ
GLONASS_G1_FREQ_CENTER = 1602.0e6
GLONASS_G1_WAVELENGTH = CTIME / GLONASS_G1_FREQ_CENTER
GPS_ORBIT_RADIUS = 26559.7e3
GPS_SAT_VELOCITY = 3874.0
GPS_ORBIT_PERIOD = 11.967 * 3600
WGS84_A = 6378137.0
WGS84_E2 = 0.00669437999014


@dataclass
class SatelliteObservation:
    prn: int
    tow: float
    week: int
    snr: float
    pseudorange: float
    az: float
    el: float
    doppler: Optional[float] = None
    carrier_phase: Optional[float] = None
    sat_pos_xyz: Optional[Tuple[float, float, float]] = None
    codei_diff: Optional[float] = None
    residual: float = 0.0
    innovation: float = 0.0
    timestamp: float = field(default_factory=time.time)


@dataclass
class ReceiverState:
    elapsed_time: float
    time: str
    lat: float
    lon: float
    hgt: float
    gdop: float
    clk_bias: float
    nsat: int
    observations: List[SatelliteObservation] = field(default_factory=list)
    timestamp: float = field(default_factory=time.time)


@dataclass
class SpoofingDetectionResult:
    timestamp: float
    suspicious_satellites: List[int] = field(default_factory=list)
    detection_methods: Dict[str, bool] = field(default_factory=dict)
    confidence: float = 0.0
    details: Dict = field(default_factory=dict)


class HistoryBuffer:
    
    def __init__(self, max_size: int = 100):
        self.max_size = max_size
        self.history: deque = deque(maxlen=max_size)
    
    def add(self, state: ReceiverState):
        self.history.append(state)
    
    def get_recent(self, n: int) -> List[ReceiverState]:
        return list(self.history)[-n:] if n <= len(self.history) else list(self.history)
    
    def get_satellite_history(self, prn: int, n: int) -> List[SatelliteObservation]:
        history = []
        for state in self.get_recent(n):
            for obs in state.observations:
                if obs.prn == prn:
                    history.append(obs)
                    break
        return history


def get_wavelength(system: str) -> float:
    system_lower = system.lower()
    if system_lower == 'g':
        return GPS_L1_WAVELENGTH
    elif system_lower == 'a':
        return GALILEO_E1_WAVELENGTH
    elif system_lower == 'l':
        return GLONASS_G1_WAVELENGTH
    else:
        return GPS_L1_WAVELENGTH


def lla_to_ecef(lat: float, lon: float, hgt: float) -> Tuple[float, float, float]:
    from math import cos, sin, radians, sqrt
    lat_rad = radians(lat)
    lon_rad = radians(lon)
    N = WGS84_A / sqrt(1 - WGS84_E2 * sin(lat_rad)**2)
    x = (N + hgt) * cos(lat_rad) * cos(lon_rad)
    y = (N + hgt) * cos(lat_rad) * sin(lon_rad)
    z = (N * (1 - WGS84_E2) + hgt) * sin(lat_rad)
    return (x, y, z)



class PseudorangeDopplerDetector:
    
    def __init__(self, system: str = 'g', debug: bool = False):
        self.debug = debug
        self.system = system
        self.wavelength = get_wavelength(system)
        self.threshold = 70.0
    
    def detect(self, state: ReceiverState, history: HistoryBuffer) -> SpoofingDetectionResult:
        result = SpoofingDetectionResult(
            timestamp=state.timestamp,
            detection_methods={'pseudorange_doppler': False}
        )
        
        if len(history.history) < 2:
            return result
        
        obs_valid = [obs for obs in state.observations 
                    if obs.pseudorange > 0 and obs.doppler is not None]
        
        if len(obs_valid) < 2:
            if self.debug:
                print(f"  [P-Doppler DEBUG] Brak danych (mamy {len(obs_valid)} obs z P i Doppler)")
            return result
        
        suspicious_prns = []
        max_deviation = 0.0
        
        prev_state = history.get_recent(2)[-2]
        dt = state.elapsed_time - prev_state.elapsed_time
        if dt <= 0 or dt > 1.0:
            dt = state.timestamp - prev_state.timestamp
        if dt <= 0 or dt > 1.0:
            dt = 0.1
        
        if self.debug:
            print(f"  [P-Doppler DEBUG] dt={dt:.3f}s, analizuje {len(obs_valid)} satelitow:")
        
        for obs in obs_valid:
            sat_history = history.get_satellite_history(obs.prn, 2)
            if len(sat_history) < 2:
                continue
            
            prev_obs = sat_history[-2]
            if prev_obs.pseudorange <= 0:
                continue
            
            dP_actual = obs.pseudorange - prev_obs.pseudorange
            
            range_rate = -self.wavelength * obs.doppler
            dP_expected = range_rate * dt
            
            deviation = abs(dP_actual - dP_expected)
            
            if self.debug:
                flag = "!!!" if deviation > self.threshold else "   "
                print(f"  {flag} PRN {obs.prn}: dP={dP_actual:.1f}m | Doppler={obs.doppler:.0f}Hz -> expected_dP={dP_expected:.1f}m | dev={deviation:.1f}m")
            
            if deviation > self.threshold:
                suspicious_prns.append(obs.prn)
                max_deviation = max(max_deviation, deviation)
        
        if suspicious_prns:
            result.detection_methods['pseudorange_doppler'] = True
            result.suspicious_satellites = list(set(suspicious_prns))
            result.confidence = min(max_deviation / (self.threshold * 10), 1.0)
            result.details['pseudorange_doppler'] = {
                'max_deviation_m': max_deviation,
                'threshold_m': self.threshold,
                'num_anomalies': len(suspicious_prns)
            }
        
        return result


class DopplerConsistencyDetector:
    
    def __init__(self, doppler_threshold: float = 15.0):
        self.doppler_threshold = doppler_threshold
    
    def detect(self, state: ReceiverState, history: HistoryBuffer) -> SpoofingDetectionResult:
        result = SpoofingDetectionResult(
            timestamp=state.timestamp,
            detection_methods={'doppler_consistency': False}
        )
        
        obs_with_doppler = [obs for obs in state.observations if obs.doppler is not None]
        if len(obs_with_doppler) < 3:
            return result
        
        suspicious_prns = []
        
        dopplers = [obs.doppler for obs in obs_with_doppler]
        mean_doppler = np.mean(dopplers)
        std_doppler = np.std(dopplers)
        
        if std_doppler < 80.0 and len(dopplers) >= 4:
            suspicious_prns = [obs.prn for obs in obs_with_doppler]
            result.detection_methods['doppler_consistency'] = True
            result.suspicious_satellites = suspicious_prns
            result.confidence = min(1.0 - (std_doppler / 100.0), 1.0)
            result.details['doppler_consistency'] = {
                'mean_doppler_hz': mean_doppler,
                'std_doppler_hz': std_doppler,
                'reason': 'all_similar_doppler'
            }
            return result
        
        if len(history.history) >= 2:
            for obs in obs_with_doppler:
                sat_history = history.get_satellite_history(obs.prn, 5)
                if len(sat_history) >= 2:
                    prev_obs = sat_history[-2]
                    if prev_obs.doppler is not None:
                        doppler_change = abs(obs.doppler - prev_obs.doppler)
                        if doppler_change > self.doppler_threshold:
                            suspicious_prns.append(obs.prn)
        
        if suspicious_prns:
            result.detection_methods['doppler_consistency'] = True
            result.suspicious_satellites = list(set(suspicious_prns))
            result.confidence = min(len(set(suspicious_prns)) / len(obs_with_doppler), 1.0)
            result.details['doppler_consistency'] = {
                'mean_doppler_hz': mean_doppler,
                'std_doppler_hz': std_doppler,
                'reason': 'sudden_doppler_change'
            }
        
        return result


class SNRMonitoringDetector:
    
    def __init__(self, snr_high_threshold: float = 48.0, snr_uniformity_threshold: float = 3.0):
        self.snr_high_threshold = snr_high_threshold
        self.snr_uniformity_threshold = snr_uniformity_threshold
    
    def detect(self, state: ReceiverState, history: HistoryBuffer) -> SpoofingDetectionResult:
        result = SpoofingDetectionResult(
            timestamp=state.timestamp,
            detection_methods={'snr_monitoring': False}
        )
        
        if state.nsat < 3:
            return result
        
        snr_values = [obs.snr for obs in state.observations if obs.snr > 0]
        if len(snr_values) < 3:
            return result
        
        mean_snr = np.mean(snr_values)
        std_snr = np.std(snr_values)
        max_snr = np.max(snr_values)
        
        suspicious_prns = []
        reason = None
        
        if std_snr < self.snr_uniformity_threshold and len(snr_values) >= 4:
            elevations = [obs.el for obs in state.observations if obs.snr > 0 and obs.el > 0]
            snrs = [obs.snr for obs in state.observations if obs.snr > 0 and obs.el > 0]
            
            if len(elevations) >= 4:
                el_mean = np.mean(elevations)
                snr_mean = np.mean(snrs)
                
                cov = sum((e - el_mean) * (s - snr_mean) for e, s in zip(elevations, snrs))
                var_el = sum((e - el_mean)**2 for e in elevations)
                
                if var_el > 0:
                    correlation = cov / (var_el * std_snr + 1e-10)
                    if correlation < 0.3:
                        suspicious_prns = [obs.prn for obs in state.observations]
                        reason = 'uniform_snr_no_el_correlation'
            else:
                suspicious_prns = [obs.prn for obs in state.observations]
                reason = 'uniform_snr'
        
        if max_snr > self.snr_high_threshold:
            for obs in state.observations:
                if obs.snr > self.snr_high_threshold:
                    if obs.prn not in suspicious_prns:
                        suspicious_prns.append(obs.prn)
            reason = reason or 'high_snr'
        
        if len(history.history) >= 2:
            for obs in state.observations:
                sat_history = history.get_satellite_history(obs.prn, 5)
                if len(sat_history) >= 2:
                    snr_change = abs(obs.snr - sat_history[-2].snr)
                    if snr_change > 6.0:
                        if obs.prn not in suspicious_prns:
                            suspicious_prns.append(obs.prn)
                        reason = reason or 'sudden_snr_change'
        
        if suspicious_prns:
            result.detection_methods['snr_monitoring'] = True
            result.suspicious_satellites = list(set(suspicious_prns))
            result.confidence = min(len(set(suspicious_prns)) / state.nsat, 1.0)
            result.details['snr_monitoring'] = {
                'mean_snr': mean_snr,
                'std_snr': std_snr,
                'max_snr': max_snr,
                'reason': reason
            }
        
        return result


class GeometryVerificationDetector:
    
    def __init__(self, max_az_change_per_epoch: float = 0.8, max_el_change_per_epoch: float = 0.4):
        self.max_az_change = max_az_change_per_epoch
        self.max_el_change = max_el_change_per_epoch
    
    def detect(self, state: ReceiverState, history: HistoryBuffer) -> SpoofingDetectionResult:
        result = SpoofingDetectionResult(
            timestamp=state.timestamp,
            detection_methods={'geometry_verification': False}
        )
        
        if len(history.history) < 2:
            return result
        
        suspicious_prns = []
        max_change = 0.0
        
        for obs in state.observations:
            sat_history = history.get_satellite_history(obs.prn, 5)
            if len(sat_history) < 2:
                continue
            
            prev_obs = sat_history[-2]
            
            az_change = abs(obs.az - prev_obs.az)
            if az_change > 180:
                az_change = 360 - az_change
            
            el_change = abs(obs.el - prev_obs.el)
            
            el_factor = max(1.0, 2.0 - obs.el / 30.0)
            
            if az_change > self.max_az_change * el_factor or el_change > self.max_el_change * el_factor:
                suspicious_prns.append(obs.prn)
                max_change = max(max_change, az_change, el_change)
        
        if suspicious_prns:
            result.detection_methods['geometry_verification'] = True
            result.suspicious_satellites = list(set(suspicious_prns))
            result.confidence = min(max_change / 10.0, 1.0)
            result.details['geometry_verification'] = {
                'max_change_deg': max_change,
                'threshold_az': self.max_az_change,
                'threshold_el': self.max_el_change
            }
        
        return result


class ClockDriftDetector:
    
    def __init__(self, max_clock_jump: float = 5e-5, max_drift_rate: float = 5e-7):
        self.max_clock_jump = max_clock_jump
        self.max_drift_rate = max_drift_rate
    
    def detect(self, state: ReceiverState, history: HistoryBuffer) -> SpoofingDetectionResult:
        result = SpoofingDetectionResult(
            timestamp=state.timestamp,
            detection_methods={'clock_drift': False}
        )
        
        if len(history.history) < 3:
            return result
        
        recent = history.get_recent(5)
        if len(recent) < 3:
            return result
        
        clock_biases = [s.clk_bias for s in recent if s.clk_bias != 0]
        if len(clock_biases) < 3:
            return result
        
        clock_changes = [clock_biases[i] - clock_biases[i-1] for i in range(1, len(clock_biases))]
        
        current_change = abs(state.clk_bias - recent[-2].clk_bias)
        
        suspicious = False
        reason = None
        
        if current_change > self.max_clock_jump:
            suspicious = True
            reason = 'clock_jump'
        
        if len(clock_changes) >= 2:
            drift_changes = [abs(clock_changes[i] - clock_changes[i-1]) for i in range(1, len(clock_changes))]
            if any(d > self.max_drift_rate for d in drift_changes):
                suspicious = True
                reason = reason or 'drift_rate_change'
        
        if suspicious:
            result.detection_methods['clock_drift'] = True
            result.suspicious_satellites = [obs.prn for obs in state.observations]
            result.confidence = min(current_change / self.max_clock_jump, 1.0)
            result.details['clock_drift'] = {
                'clock_bias_s': state.clk_bias,
                'clock_change_s': current_change,
                'reason': reason
            }
        
        return result


class CarrierPhaseConsistencyDetector:
    
    def __init__(self, system: str = 'g', divergence_threshold: float = 2.0):
        self.system = system
        self.divergence_threshold = divergence_threshold
        self.wavelength = get_wavelength(system)
    
    def detect(self, state: ReceiverState, history: HistoryBuffer) -> SpoofingDetectionResult:
        result = SpoofingDetectionResult(
            timestamp=state.timestamp,
            detection_methods={'carrier_phase_consistency': False}
        )
        
        obs_with_phase = [obs for obs in state.observations 
                         if obs.carrier_phase is not None and obs.pseudorange > 0]
        
        if len(obs_with_phase) < 2:
            return result
        
        suspicious_prns = []
        max_divergence_change = 0.0
        
        for obs in obs_with_phase:
            carrier_range = obs.carrier_phase * self.wavelength
            divergence = obs.pseudorange - carrier_range
            
            sat_history = history.get_satellite_history(obs.prn, 5)
            prev_divergences = []
            
            for prev_obs in sat_history[:-1]:
                if prev_obs.carrier_phase is not None and prev_obs.pseudorange > 0:
                    prev_carrier = prev_obs.carrier_phase * self.wavelength
                    prev_div = prev_obs.pseudorange - prev_carrier
                    prev_divergences.append(prev_div)
            
            if len(prev_divergences) >= 2:
                mean_div = np.mean(prev_divergences)
                std_div = np.std(prev_divergences)
                
                divergence_change = abs(divergence - mean_div)
                
                adaptive_threshold = self.divergence_threshold * max(1.0, state.gdop / 2.0)
                
                if divergence_change > adaptive_threshold and (std_div == 0 or divergence_change > 3 * std_div):
                    suspicious_prns.append(obs.prn)
                    max_divergence_change = max(max_divergence_change, divergence_change)
        
        if suspicious_prns:
            result.detection_methods['carrier_phase_consistency'] = True
            result.suspicious_satellites = list(set(suspicious_prns))
            result.confidence = min(max_divergence_change / (self.divergence_threshold * 5), 1.0)
            result.details['carrier_phase_consistency'] = {
                'max_divergence_change_m': max_divergence_change,
                'threshold_m': self.divergence_threshold
            }
        
        return result


class SpoofingDetector:
    
    def __init__(self, 
                 system: str = 'g',
                 alert_threshold: int = 3, 
                 required_score: int = 4):
        self.system = system
        self.history = HistoryBuffer(max_size=100)
        
        self.detectors = [
            PseudorangeDopplerDetector(system=system),
            DopplerConsistencyDetector(),
            SNRMonitoringDetector(),
            GeometryVerificationDetector(),
            ClockDriftDetector(),
            CarrierPhaseConsistencyDetector(system=system),
        ]
        
        self.detector_weights = [2, 2, 2, 2, 1, 2]
        
        self.alert_count = 0
        self.alert_threshold = alert_threshold
        self.required_score = required_score
    
    def process_data(self, json_data: dict) -> Tuple[List[SpoofingDetectionResult], dict]:
        state = self._parse_json(json_data)
        self.history.add(state)
        
        results = []
        for detector in self.detectors:
            result = detector.detect(state, self.history)
            results.append(result)
        
        detection_score = sum(
            weight for weight, result in zip(self.detector_weights, results)
            if any(result.detection_methods.values())
        )
        
        required = max(6, self.required_score) if state.gdop > 5.0 else self.required_score
        
        if detection_score >= required:
            self.alert_count += 1
        else:
            self.alert_count = max(0, self.alert_count - 1)
        
        alert_info = {
            'alert_count': self.alert_count,
            'alert_threshold': self.alert_threshold,
            'spoofing_detected': self.alert_count >= self.alert_threshold,
            'detection_score': detection_score,
            'required_score': required,
            'detection_count': detection_score,
            'required_detections': required,
            'detectors_total': len(self.detectors)
        }
        
        return results, alert_info
    
    def _parse_json(self, data: dict) -> ReceiverState:
        observations = []
        
        for obs_data in data.get('observations', []):
            sat_pos = obs_data.get('sat_pos')
            sat_pos_xyz = tuple(sat_pos) if sat_pos and len(sat_pos) == 3 else None
            
            obs = SatelliteObservation(
                prn=obs_data.get('prn'),
                tow=obs_data.get('tow'),
                week=obs_data.get('week'),
                snr=obs_data.get('snr'),
                pseudorange=obs_data.get('pseudorange', obs_data.get('doppler', 0)),
                az=obs_data.get('az'),
                el=obs_data.get('el'),
                doppler=obs_data.get('doppler_freq'),
                carrier_phase=obs_data.get('carrier_phase'),
                sat_pos_xyz=sat_pos_xyz,
                codei_diff=obs_data.get('codei_diff'),
                residual=obs_data.get('residual', 0.0),
                innovation=obs_data.get('innovation', 0.0)
            )
            observations.append(obs)
        
        position = data.get('position', {})
        
        return ReceiverState(
            elapsed_time=data.get('elapsed_time', 0.0),
            time=data.get('time', ''),
            lat=position.get('lat', 0.0),
            lon=position.get('lon', 0.0),
            hgt=position.get('hgt', 0.0),
            gdop=position.get('gdop', 0.0),
            clk_bias=position.get('clk_bias', 0.0),
            nsat=position.get('nsat', 0),
            observations=observations
        )
    
    def get_summary(self, results: List[SpoofingDetectionResult]) -> dict:
        all_suspicious = set()
        methods_detected = {}
        
        for result in results:
            for method, detected in result.detection_methods.items():
                if detected:
                    methods_detected[method] = True
                    all_suspicious.update(result.suspicious_satellites)
        
        return {
            'spoofing_detected': len(all_suspicious) > 0,
            'suspicious_satellites': sorted(list(all_suspicious)),
            'methods_triggered': list(methods_detected.keys()),
            'timestamp': time.time()
        }




