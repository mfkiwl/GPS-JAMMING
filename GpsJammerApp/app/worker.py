import os
from PySide6.QtCore import QThread, Signal
import subprocess  
import sys
import json
import threading
import time
import numpy as np
from collections import deque
from http.server import HTTPServer, BaseHTTPRequestHandler
import datetime

sys.path.append(os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))), 'skrypty'))
from triangulateRSSI import triangulate_jammer_location

# [FIX] Klasa serwera z wymuszonym ponownym użyciem portu
# Zapobiega błędowi "Address already in use" przy restarcie analizy
class ReusableHTTPServer(HTTPServer):
    allow_reuse_address = True

class _DataReceiverHandler(BaseHTTPRequestHandler):
    thread_instance = None

    def do_POST(self):
        if self.path == '/data':
            try:
                content_length = int(self.headers.get('Content-Length', 0))
                body = self.rfile.read(content_length)
                data = json.loads(body.decode('utf-8'))
                
                if self.thread_instance:
                    self.thread_instance.process_incoming_data(data)
                    
                try:
                    self.send_response(200)
                    self.send_header('Content-Type', 'application/json')
                    response_body = b'{"status":"ok"}'
                    self.send_header('Content-Length', str(len(response_body)))
                    self.end_headers()
                    self.wfile.write(response_body)
                except (BrokenPipeError, ConnectionResetError):
                    pass
                    
            except json.JSONDecodeError:
                print("Błąd parsowania JSON")
                try:
                    self.send_response(400)
                    self.end_headers()
                except (BrokenPipeError, ConnectionResetError):
                    pass
            except Exception as e:
                print(f"[HTTP HANDLER] Błąd: {e}")
                try:
                    self.send_response(500)
                    self.end_headers()
                except (BrokenPipeError, ConnectionResetError):
                    pass
        else:
            try:
                self.send_response(404) 
                self.end_headers()
            except (BrokenPipeError, ConnectionResetError):
                pass

    def log_message(self, format, *args):
        pass

class GPSAnalysisThread(QThread):

    analysis_complete = Signal(list)  
    progress_update = Signal(int, str)
    new_analysis_text = Signal(str) 
    new_position_data = Signal(float, float, float)
    jamming_analysis_complete = Signal(list) 
    triangulation_complete = Signal(dict)
    jamming_detected_realtime = Signal(bool, dict)

    def __init__(self, file_paths, power_threshold=6.0, antenna_positions=None, satellite_system='GPS', hold_position=False):
        super().__init__()
        self.file_paths = file_paths
        
        self.POWER_CHUNK_SIZE = 32768 
        
        # Progi Naukowe
        # Ignorujemy argument power_threshold na rzecz stałej 6.0dB (ITU-R)
        self.THRESHOLD_POWER_RISE_DB = 6.0
        print(f"[GPS THREAD] Próg detekcji mocy (ITU-R): {self.THRESHOLD_POWER_RISE_DB} dB")
        
        self.THRESHOLD_CN0_DROP_DB = 8.0 
        self.THRESHOLD_RESIDUALS_MEDIAN_M = 40.0
        self.THRESHOLD_RESIDUAL_SINGLE_SAT_M = 800.0 
        self.MIN_BAD_SATS_FOR_ALARM = 2
        self.THRESHOLD_HGT_MAX = 10000.0 
        self.THRESHOLD_GDOP_MAX = 6.0 
        self.THRESHOLD_NSAT_MIN = 4   
        
        self.antenna_positions = antenna_positions if antenna_positions else {
            'antenna1': [0.0, 0.0],
            'antenna2': [0.5, 0.0],
            'antenna3': [0.0, 0.5]
        }
        
        self.satellite_system = satellite_system
        if satellite_system == 'GPS':
            self.gnss_system_flag = '-g'
        elif satellite_system == 'GLONASS':
            self.gnss_system_flag = '-n'
        elif satellite_system == 'Galileo':
            self.gnss_system_flag = '-l'
        else:
            self.gnss_system_flag = '-g'
        
        self.hold_position = hold_position
        
        # Zmienne stanu
        self.current_buffcnt = 0
        self.current_lat = 0.0
        self.current_lon = 0.0
        self.current_hgt = 0.0
        self.current_nsat = 0
        self.current_gdop = 0.0
        self.current_clk_bias = 0.0
        self.current_signal_time = 0.0
        
        self.current_cn0_avg = 0.0 
        self.current_residuals_median = 0.0 
        self.current_residuals_bad_count = 0 
        
        # Zmienne Mocy
        self.power_map = [] 
        self.global_baseline_power = 0.0 
        self.current_iq_power = 0.0 
        self.power_map_ready = False
        self.total_file_bytes = 0 
        self.power_detection_enabled = True
        self.jamming_start_byte_offset = None 
        
        # Historia
        self.cn0_history = deque(maxlen=100) 
        self.median_cn0 = 0.0
        
        # Logika Jammingu
        self.jamming_detected = False
        self.jamming_events = [] 
        
        self.potential_jamming_start_signal_time = None 
        self.potential_jamming_end_signal_time = None 
        self.potential_start_buffcnt = 0 
        self.active_event_start_buffcnt = 0 
        self.active_event_start_time = 0.0
        
        self.required_jamming_duration_sec = 2.5 
        self.required_clean_duration_sec = 2.0   
        
        self.last_safe_position_buffer = deque(maxlen=50) 
        self.last_position_before_jamming = {
            'lat': 0.0, 'lon': 0.0, 'hgt': 0.0, 'buffcnt': 0, 'valid': False
        } 
        
        self.http_server = None
        self.http_thread = None
        self.jamming_thread = None
        self.triangulation_thread = None
        self.total_samples = 0
        self.estimated_total_samples = 0
        self.triangulation_result = None
        self.stop_requested = False 
        self.triangulation_started = False
        
        self.triangulation_complete.connect(self.on_triangulation_complete)
        
        # Ustawienie ścieżek
        try:
            base_dir = os.path.dirname(os.path.abspath(__file__)) 
            # Zakładamy strukturę: backend/analysis -> backend/bin/gnssdec
            # Czyli wychodzimy z analysis (..) i wchodzimy do bin
            self.gnssdec_path = os.path.abspath(os.path.join(base_dir, "..","backend", "bin", "gnssdec"))
        except NameError:
            self.gnssdec_path = "gnssdec" # Fallback

        if self.file_paths:
            self.calculate_file_samples() 
            
    def calculate_file_samples(self):
        try:
            if not self.file_paths or not os.path.exists(self.file_paths[0]):
                return
            file_path = self.file_paths[0]
            file_size = os.path.getsize(file_path)
            self.total_file_bytes = file_size 
            bytes_per_sample = 2 
            self.total_samples = file_size // bytes_per_sample
            self.estimated_total_samples = self.total_samples
        except Exception as e:
            print(f"[PROGRESS] Błąd przy obliczaniu próbek: {e}")
            self.total_samples = 0

    def precalculate_power_profile(self):
        """Skanuje plik .bin (format uint8) i tworzy mapę mocy."""
        if not self.file_paths or not os.path.exists(self.file_paths[0]):
            return

        file_path = self.file_paths[0]
        print(f"[POWER SCAN] Skanowanie (uint8): {os.path.basename(file_path)}")
        self.progress_update.emit(0, "scanning_power")
        
        try:
            dt = np.uint8 
            chunk_size_bytes = self.POWER_CHUNK_SIZE * 2 
            temp_powers = []
            
            self.total_file_bytes = os.path.getsize(file_path)
            
            with open(file_path, 'rb') as f:
                processed_bytes = 0
                while not self.stop_requested:
                    raw_data = f.read(chunk_size_bytes)
                    if not raw_data: break
                    
                    samples = np.frombuffer(raw_data, dtype=dt)
                    # KOREKTA RTL-SDR (0..255 -> -127.5..127.5)
                    samples_f = samples.astype(np.float32) - 127.5
                    
                    i = samples_f[0::2]
                    q = samples_f[1::2]
                    min_len = min(len(i), len(q))
                    
                    power = i[:min_len]**2 + q[:min_len]**2
                    avg_pow = np.mean(power) + 1e-10
                    temp_powers.append(avg_pow)
                    
                    processed_bytes += len(raw_data)
                    
                    if self.total_file_bytes > 0:
                        progress = int((processed_bytes / self.total_file_bytes) * 10)
                        if progress % 2 == 0: 
                             self.progress_update.emit(progress, "scanning_power")

            self.power_map = np.array(temp_powers)
            
            if len(self.power_map) > 0:
                self.global_baseline_power = np.percentile(self.power_map, 5) 
                if self.global_baseline_power <= 0: self.global_baseline_power = 1.0
                
                threshold_ratio = 10**(self.THRESHOLD_POWER_RISE_DB / 10.0) 
                power_threshold_linear = self.global_baseline_power * threshold_ratio
                
                jamming_indices = np.where(self.power_map > power_threshold_linear)[0]
                
                # Wykrywanie przedziałów
                self.jamming_byte_ranges = []
                if len(jamming_indices) > 0:
                    jamming_mask = self.power_map > power_threshold_linear
                    diffs = np.diff(jamming_mask.astype(int))
                    starts = np.where(diffs == 1)[0] + 1
                    ends = np.where(diffs == -1)[0] + 1
                    
                    if jamming_mask[0]: starts = np.insert(starts, 0, 0)
                    if jamming_mask[-1]: ends = np.append(ends, len(self.power_map))
                    
                    for s, e in zip(starts, ends):
                        start_byte = s * chunk_size_bytes
                        end_byte = e * chunk_size_bytes
                        self.jamming_byte_ranges.append((start_byte, end_byte))
                        
                    print(f"[POWER SCAN] Wykryto {len(self.jamming_byte_ranges)} okresów wysokiej mocy (F1).")
                else:
                    print(f"[POWER SCAN] Nie wykryto skoku mocy powyżej progu {self.THRESHOLD_POWER_RISE_DB} dB.")
            
            self.power_map_ready = True
            self.progress_update.emit(10, "scanning_power_done")

        except Exception as e:
            print(f"[POWER SCAN] BŁĄD: {e}")
            self.power_map_ready = False

    def process_incoming_data(self, data):
        try:
            position = data.get('position', {})
            observations = data.get('observations', [])
            elapsed_str = data.get('elapsed_time', 0.0)
            try:
                self.current_signal_time = float(elapsed_str)
            except: pass

            if position:
                self.current_buffcnt = position.get('buffcnt', 0) 
                self.current_lat = float(position.get('lat', 0.0))
                self.current_lon = float(position.get('lon', 0.0))
                self.current_hgt = float(position.get('hgt', 0.0))
                self.current_nsat = position.get('nsat', 0)
                self.current_gdop = float(position.get('gdop', 0.0))
                self.current_clk_bias = float(position.get('clk_bias', 0.0))
                
                # Aktualizacja mocy
                if self.power_map_ready and self.total_file_bytes > 0:
                    ratio = self.current_buffcnt / self.total_file_bytes
                    ratio = max(0.0, min(1.0, ratio))
                    idx = int(ratio * len(self.power_map))
                    idx = min(idx, len(self.power_map)-1)
                    self.current_iq_power = self.power_map[idx]

                # Analiza obserwacji
                snr_values = [obs.get('snr', 0.0) for obs in observations if 'snr' in obs]
                if snr_values:
                    self.current_cn0_avg = np.mean(snr_values)
                    residuals = [obs.get('residual', 0.0) for obs in observations if 'residual' in obs]
                    if residuals:
                        self.current_residuals_median = np.median(residuals)
                        bad_sats = sum(1 for r in residuals if r > self.THRESHOLD_RESIDUAL_SINGLE_SAT_M)
                        self.current_residuals_bad_count = bad_sats
                    else:
                        self.current_residuals_median = 0.0
                        self.current_residuals_bad_count = 0
                else:
                    self.current_cn0_avg = 0.0
                    self.current_residuals_median = 0.0
                    self.current_residuals_bad_count = 0

                if not self.jamming_detected and self.current_cn0_avg > 0:
                    self.cn0_history.append(self.current_cn0_avg)
                if len(self.cn0_history) > 10: 
                    self.median_cn0 = np.median(self.cn0_history)
                else:
                    self.median_cn0 = self.current_cn0_avg 

                self.update_progress_bar()
                
                is_safe = True
                # Sprawdzenie mapy mocy
                if self.jamming_byte_ranges:
                    # Jeśli jesteśmy w którymkolwiek zakresie lub po nim
                    if self.current_buffcnt >= self.jamming_byte_ranges[0][0]:
                        is_safe = False
                
                if self.jamming_detected:
                    is_safe = False

                if is_safe and self.current_lat != 0.0 and self.current_nsat >= 4:
                     self.last_position_before_jamming = {
                        'lat': self.current_lat,
                        'lon': self.current_lon,
                        'hgt': self.current_hgt,
                        'buffcnt': self.current_buffcnt,
                        'valid': True
                    }

                self.check_jamming_conditions()

            pwr_db = 0.0
            if self.global_baseline_power > 0 and self.current_iq_power > 0:
                pwr_db = 10 * np.log10(self.current_iq_power / self.global_baseline_power)
            
            txt = f"[{self.current_signal_time:.2f}s, Lat:{self.current_lat:.6f}, Lon:{self.current_lon}, Pwr:{pwr_db:.1f}dB]"
            self.new_analysis_text.emit(txt)
            
            if not self.jamming_detected and (self.current_lat != 0.0 or self.current_lon != 0.0):
                self.new_position_data.emit(self.current_lat, self.current_lon, self.current_hgt)
                    
        except Exception as e:
            print(f"[WORKER] Błąd: {e}")

    def check_jamming_conditions(self):
        # F1: MOC
        flag_f1 = False
        if self.jamming_byte_ranges:
            for start_byte, end_byte in self.jamming_byte_ranges:
                if start_byte <= self.current_buffcnt <= end_byte:
                    flag_f1 = True
                    break
        
        # F2: JAKOŚĆ
        flag_f2 = False
        if len(self.cn0_history) > 40: 
            if self.current_cn0_avg < (self.median_cn0 - self.THRESHOLD_CN0_DROP_DB):
                flag_f2 = True

        # F3: INTEGRITY
        flag_f3 = False
        integrity_fail = (self.current_residuals_median > self.THRESHOLD_RESIDUALS_MEDIAN_M) or \
                         (self.current_residuals_bad_count >= self.MIN_BAD_SATS_FOR_ALARM)
        
        # F4: WYSOKOŚĆ
        flag_hgt = False
        if self.current_nsat > 0 and abs(self.current_hgt) > self.THRESHOLD_HGT_MAX:
            flag_hgt = True

        nav_issue = (flag_f3 or flag_hgt) and (self.current_nsat > 0)
        is_jamming_now = flag_f1 or flag_f2 or nav_issue
        
        if not self.jamming_detected:
            if is_jamming_now:
                if flag_f1:
                     self.confirm_jamming_start(reason="Moc (Mapowana)")
                else:
                    if self.potential_jamming_start_signal_time is None:
                        self.potential_jamming_start_signal_time = self.current_signal_time
                        self.potential_start_buffcnt = self.current_buffcnt
                    elif (self.current_signal_time - self.potential_jamming_start_signal_time) >= self.required_jamming_duration_sec:
                        self.confirm_jamming_start(reason="Jakość/Integrity")
            else:
                self.potential_jamming_start_signal_time = None
        else:
            if not is_jamming_now:
                if self.potential_jamming_end_signal_time is None:
                    self.potential_jamming_end_signal_time = self.current_signal_time
                else:
                    clean_duration = self.current_signal_time - self.potential_jamming_end_signal_time
                    if clean_duration >= self.required_clean_duration_sec:
                        self.confirm_jamming_end()
                        self.potential_jamming_end_signal_time = None
            else:
                self.potential_jamming_end_signal_time = None

    def confirm_jamming_start(self, reason="N/A"):
        self.jamming_detected = True
        
        start_byte = self.potential_start_buffcnt
        if reason == "Moc (Mapowana)" and self.jamming_byte_ranges:
             for s, e in self.jamming_byte_ranges:
                 if s <= self.current_buffcnt <= e:
                     start_byte = s
                     break
        else:
             start_byte = self.potential_start_buffcnt if self.potential_start_buffcnt > 0 else self.current_buffcnt

        self.active_event_start_buffcnt = start_byte
        self.active_event_start_time = self.current_signal_time
        
        if reason == "Jakość/Integrity" and self.potential_jamming_start_signal_time:
             self.active_event_start_time = self.potential_jamming_start_signal_time

        print(f"[DETEKTOR] 🚨 ATAK POTWIERDZONY! Powód: {reason}")
        print(f"[DETEKTOR]    Start: {self.active_event_start_time:.2f}s")
        
        if self.last_position_before_jamming['valid']:
            self.jamming_detected_realtime.emit(True, self.last_position_before_jamming)
        else:
            print("[DETEKTOR] ⚠️ Brak bezpiecznej pozycji przed atakiem!")

    def confirm_jamming_end(self):
        self.jamming_detected = False
        end_time = self.current_signal_time
        print(f"[DETEKTOR] ✅ Koniec ataku. (Koniec: {end_time:.2f}s)")
        
        start_sample = self.active_event_start_buffcnt
        end_sample = self.current_buffcnt
        
        event_data = {
            'start_sample': start_sample,
            'end_sample': end_sample,
            'start_time': self.active_event_start_time,
            'end_time': end_time,
            'duration': end_time - self.active_event_start_time
        }
        
        self.jamming_events.append(event_data)
        self.jamming_detected_realtime.emit(False, {})

    def get_best_safe_position(self):
        if not self.last_safe_position_buffer:
            return self.last_position_before_jamming 
        return self.last_safe_position_buffer[0]

    def update_progress_bar(self):
        if self.total_samples > 0 and self.current_buffcnt > 0:
            current_total = max(self.total_samples, self.estimated_total_samples)
            progress_percent = min(100, int((self.current_buffcnt / current_total) * 100))
            if self.jamming_detected:
                status = "triangulating" if (self.triangulation_thread and self.triangulation_thread.is_alive()) else "jamming"
                self.progress_update.emit(progress_percent, status)
            else:
                self.progress_update.emit(progress_percent, "normal")
        else:
            self.progress_update.emit(0, "normal")

    def run(self):
        self.stop_requested = False
        _DataReceiverHandler.thread_instance = self
        
        print("[WORKER] Uruchamianie wątku analizy...")

        # 1. SERVER
        try:
            server_address = ('127.0.0.1', 1234)
            # [FIX] Używamy ReusableHTTPServer
            self.http_server = ReusableHTTPServer(server_address, _DataReceiverHandler)
            self.http_thread = threading.Thread(target=self.http_server.serve_forever)
            self.http_thread.daemon = True 
            self.http_thread.start()
            print("[WORKER] Serwer HTTP uruchomiony (port 1234).") 
        except Exception as e:
            print(f"[WORKER] Błąd serwera: {e}")
            return
        
        file1 = self.file_paths[0] if self.file_paths else None
        if not file1: return

        # 2. PRE-SCAN
        self.precalculate_power_profile()
        
        if self.stop_requested:
            self.shutdown_server()
            return

        # 3. GNSSDEC
        try:
            print(f"[WORKER] Uruchamianie analizy {self.gnssdec_path}...")
            gnssdec_command = [self.gnssdec_path, self.gnss_system_flag]
            if self.hold_position:
                gnssdec_command.append('-h')
            gnssdec_command.append(file1)
            
            self.current_signal_time = 0.0
            self.cn0_history.clear()
            
            subprocess.run(gnssdec_command, check=True, capture_output=True, text=True)
            print(f"[WORKER] Analiza gnssdec zakończona.")
            
        except Exception as e:
            print(f"[WORKER] Błąd procesu gnssdec: {e}")
            
        finally:
            self.progress_update.emit(100, "completed")
            self.shutdown_server()
            
            # Jeśli plik się skończył, a jamming trwa, zamykamy zdarzenie
            if self.jamming_detected:
                print("[WORKER] Plik zakończony w trakcie aktywnego jammingu. Zamykanie zdarzenia.")
                end_time = self.current_signal_time
                start_sample = self.active_event_start_buffcnt
                end_sample = self.current_buffcnt
                
                event_data = {
                    'start_sample': start_sample,
                    'end_sample': end_sample,
                    'start_time': self.active_event_start_time,
                    'end_time': end_time,
                    'duration': end_time - self.active_event_start_time
                }
                self.jamming_events.append(event_data)
                
                # [FIX] Uruchom triangulację, jeśli jest jamming
                print("[WORKER] Uruchamiam triangulację na koniec pliku...")
                self.analyze_triangulation_after_gnssdec()
                if self.triangulation_thread:
                    self.triangulation_thread.join()
            
            result_info = []
            if self.jamming_events:
                 for i, ev in enumerate(self.jamming_events):
                    result_info.append({
                        'type': 'jamming',
                        'event_number': i + 1,
                        'start_sample': ev['start_sample'],
                        'end_sample': ev['end_sample'],
                        'start_time': ev['start_time'],
                        'end_time': ev['end_time'],
                        'duration': ev['duration'],
                        'triangulation': self.triangulation_result
                    })
            else:
                 result_info.append({'type': 'no_jamming'})
                 
            self.analysis_complete.emit(result_info)

    def analyze_triangulation_after_gnssdec(self):
        def triangulation_worker():
            try:
                if len(self.file_paths) < 2: 
                    print("[TRIANGULACJA] Za mało plików do triangulacji.")
                    return

                print(f"[TRIANGULACJA] Start obliczeń...")
                final_position = self.last_position_before_jamming
                if final_position['valid']:
                    ref_lat = final_position['lat']
                    ref_lon = final_position['lon']
                else:
                    ref_lat = self.current_lat if self.current_lat != 0.0 else 50.0
                    ref_lon = self.current_lon if self.current_lon != 0.0 else 20.0
                
                test_files = self.get_test_files_for_triangulation()
                antenna_positions_meters = [
                    np.array(self.antenna_positions['antenna1']),
                    np.array(self.antenna_positions['antenna2']),
                    np.array(self.antenna_positions['antenna3'])
                ]
                
                result = triangulate_jammer_location(
                    file_paths=test_files,
                    antenna_positions_meters=antenna_positions_meters,
                    reference_lat=ref_lat,
                    reference_lon=ref_lon,
                    tx_power=40.0,
                    path_loss_exp=3.0,
                    frequency_mhz=1575.42,
                    threshold=0.0, 
                    verbose=False
                )
                
                if result['success'] and final_position['valid']:
                    result['reference_position'] = final_position
                
                self.triangulation_result = result
                self.triangulation_complete.emit(result)
            except Exception as e:
                print(f"[TRIANGULACJA] Błąd: {e}")
        
        self.triangulation_thread = threading.Thread(target=triangulation_worker)
        self.triangulation_thread.start()

    def get_test_files_for_triangulation(self):
        test_files = []
        base_dir = os.path.dirname(self.file_paths[0]) if self.file_paths else "../data"
        num_files = len(self.file_paths)
        for i in range(min(num_files, 3)): 
            test_filename = f"test{i+1}.bin"
            test_path = os.path.join(base_dir, test_filename)
            if os.path.exists(test_path):
                test_files.append(test_path)
            else:
                if i < len(self.file_paths):
                    test_files.append(self.file_paths[i])
        if not test_files:
            return self.file_paths
        return test_files

    def on_triangulation_complete(self, result):
        self.triangulation_result = result
        if result['success']:
            geo = result['location_geographic']
            print(f"[TRIANGULATION] 🎯 Wynik: {geo['lat']:.8f}, {geo['lon']:.8f}")
            
    def get_triangulation_result(self):
        return self.triangulation_result

    def on_jamming_detected(self, events):
        pass

    def shutdown_server(self):
        if self.http_server:
            self.http_server.shutdown() 
            self.http_thread.join() 
            self.http_server = None

    def get_current_position_data(self):
        return {
            'buffcnt': self.current_buffcnt,
            'lat': self.current_lat,
            'lon': self.current_lon,
            'nsat': self.current_nsat,
            'jamming': self.jamming_detected
        }