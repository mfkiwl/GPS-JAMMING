import os
from PySide6.QtCore import QThread, Signal
import subprocess  
import sys
import json
import threading
from http.server import HTTPServer, BaseHTTPRequestHandler
import datetime 

class _DataReceiverHandler(BaseHTTPRequestHandler):
    thread_instance = None

    def do_POST(self):
        if self.path == '/data':
            try:
                content_length = int(self.headers.get('Content-Length', 0))
                body = self.rfile.read(content_length)
                data = json.loads(body.decode('utf-8'))

                timestamp = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S.%f")[:-3]
                print(f"\n[{timestamp}] Otrzymano dane JSON:")
                print(json.dumps(data, indent=2, ensure_ascii=False))

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
        """Wycisza logi serwera HTTP."""
        pass

class GPSAnalysisThread(QThread):

    analysis_complete = Signal(list)  
    progress_update = Signal(int)     
    new_analysis_text = Signal(str) 
    new_position_data = Signal(float, float, float)

    def __init__(self, file_paths):
        super().__init__()
        self.file_paths = file_paths
        
        self.http_server = None
        self.http_thread = None 
        
        try:
            app_dir = os.path.dirname(os.path.abspath(__file__)) 
        except NameError:
            app_dir = os.getcwd() 
        self.project_root_dir = os.path.dirname(app_dir)
        
        self.gnssdec_path = os.path.join(
            self.project_root_dir, "backendhttp", "bin", "gnssdec"
        )
        
    def process_incoming_data(self, data):
        try:
            elapsed = data.get('elapsed_time', 'N/A')
            time = data.get('time', 'N/A')
            decoded = data.get('decoded', [])
            obs_list = []
            for obs in data.get('observations', []):
                obs_list.append(f"  PRN: {obs.get('prn')}, SNR: {obs.get('snr')}")
            obs_text = "\n".join(obs_list)

            text_output = (
                f"Czas: {time} (Upłynęło: {elapsed}s)\n"
                f"Dekodowane satelity: {decoded}\n"
                f"Obserwacje:\n{obs_text}\n"
                f"--------------------------------"
            )
            self.new_analysis_text.emit(text_output)
            
            position = data.get('position', {})
            lat = float(position.get('lat', 0.0))
            lon = float(position.get('lon', 0.0))
            hgt = float(position.get('hgt', 0.0))
            
            if lat != 0.0 or lon != 0.0:
                 self.new_position_data.emit(lat, lon, hgt)

        except Exception as e:
            print(f"[WORKER] Błąd podczas przetwarzania danych JSON: {e}")

    def run(self):
        _DataReceiverHandler.thread_instance = self

        try:
            server_address = ('127.0.0.1', 1234)
            self.http_server = HTTPServer(server_address, _DataReceiverHandler)
            self.http_thread = threading.Thread(target=self.http_server.serve_forever)
            self.http_thread.daemon = True 
            self.http_thread.start()
            print("[WORKER] Serwer HTTP uruchomiony na porcie 1234 (jak stary skrypt).") 
            
        except Exception as e:
            print(f"[WORKER] BŁĄD: Nie można uruchomić serwera HTTP na porcie 1234: {e}")
            self.analysis_complete.emit([])
            return
        
        file1 = self.file_paths[0] if self.file_paths else None
        
        if not file1 or not os.path.exists(file1):
            print(f"BŁĄD: Plik {file1} nie istnieje. Przerwanie.")
            self.shutdown_server()
            self.analysis_complete.emit([])
            return
            
        if not os.path.exists(self.gnssdec_path):
            print(f"BŁĄD: Nie znaleziono programu {self.gnssdec_path}. Przerwanie.")
            self.shutdown_server()
            self.analysis_complete.emit([])
            return
        
        try:
            print(f"[WORKER] Uruchamianie analizy {self.gnssdec_path}...")
            print(f"[WORKER] --- WĄTEK CZEKA NA ZAKOŃCZENIE ./gnssdec ---")
            gnssdec_command = [self.gnssdec_path, file1]
            result = subprocess.run(gnssdec_command, check=True, capture_output=True, text=True)
            print(f"[WORKER] Analiza {self.gnssdec_path} zakończona.")
            
        except subprocess.CalledProcessError as e:
            print(f"BŁĄD: Proces {self.gnssdec_path} zakończył się błędem!")
        except Exception as e:
            print(f"Nieoczekiwany błąd podczas uruchamiania gnssdec: {e}")
            
        finally:
            self.shutdown_server()
            print("[WORKKID] Wątek zakończył pracę. Odblokowanie UI.")
            self.analysis_complete.emit([]) # Wysyłamy pustą listę na koniec

    def shutdown_server(self):
        """Bezpiecznie zamyka serwer HTTP."""
        if self.http_server:
            print("[WORKER] Zamykanie serwera HTTP...")
            self.http_server.shutdown() 
            self.http_thread.join() 
            self.http_server = None
            self.http_thread = None
            print("[WORKER] Serwer HTTP zamknięty.")