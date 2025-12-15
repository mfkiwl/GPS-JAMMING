#!/usr/bin/env python3

import argparse
import json
import sys
from datetime import datetime
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from spoofing_detection import SpoofingDetector, SpoofingDetectionResult


class SpoofingDetectionHandler(BaseHTTPRequestHandler):    
    detector: SpoofingDetector = None
    verbose: bool = False
    enable_detection: bool = True
    request_count: int = 0
    
    def do_POST(self):
        if self.path == '/data':
            self.request_count += 1
            
            try:
                content_length = int(self.headers.get('Content-Length', 0))
                body = self.rfile.read(content_length)
                data = json.loads(body.decode('utf-8'))
                
                if self.enable_detection and self.detector:
                    results, alert_info = self.detector.process_data(data)
                    self._display_detection_results(data, results, alert_info)
                elif self.verbose:
                    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S.%f")[:-3]
                    print(f"\n[{timestamp}] Request #{self.request_count}")
                    print(f"Satelity: {len(data.get('observations', []))}")
                
                try:
                    self.send_response(200)
                    self.send_header('Content-Type', 'application/json')
                    self.send_header('Content-Length', '15')
                    self.end_headers()
                    self.wfile.write(b'{"status":"ok"}')
                except (BrokenPipeError, ConnectionResetError):
                    pass
                    
            except json.JSONDecodeError as e:
                if self.verbose:
                    print(f"Blad parsowania JSON: {e}")
                try:
                    self.send_response(400)
                    self.end_headers()
                except (BrokenPipeError, ConnectionResetError):
                    pass
            except Exception as e:
                if self.verbose:
                    print(f"Blad: {e}")
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
    
    def _display_detection_results(self, data: dict, results: list, alert_info: dict):
        timestamp = datetime.now().strftime("%H:%M:%S.%f")[:-3]
        position = data.get('position', {})
        nsat = len(data.get('observations', []))
        
        suspicious_prns = set()
        any_detection = False
        for result in results:
            if any(result.detection_methods.values()):
                any_detection = True
                suspicious_prns.update(result.suspicious_satellites)
        
        alert_count = alert_info.get('alert_count', 0)
        alert_threshold = alert_info.get('alert_threshold', 1)
        spoofing_detected = alert_info.get('spoofing_detected', False)
        detection_count = alert_info.get('detection_count', 0)
        required_detections = alert_info.get('required_detections', 5)
        
        if spoofing_detected:
            print(f"\n")
            print(f"[{timestamp}] [ALERT] SPOOFING DETECTED! Request #{self.request_count}")
            print(f"")
            print(f"Alert Count: {alert_count}/{alert_threshold}")
            print(f"Pozycja: lat={position.get('lat', 0):.6f}, lon={position.get('lon', 0):.6f}, hgt={position.get('hgt', 0):.1f}m")
            print(f"Satelity: {nsat} (podejrzane: {sorted(suspicious_prns) if suspicious_prns else 'brak'})")
            print(f"GDOP: {position.get('gdop', 0):.2f}")
        elif self.verbose:
            triggered_methods = []
            for result in results:
                for method, detected in result.detection_methods.items():
                    if detected:
                        short_name = method.replace('_consistency', '').replace('_monitoring', '').replace('_verification', '').replace('_drift', '').upper()
                        triggered_methods.append(short_name)
            
            if any_detection:
                methods_str = ','.join(triggered_methods) if triggered_methods else '-'
                print(f"[{timestamp}] #{self.request_count} | Alert: {alert_count}/{alert_threshold} | Detekcje: {detection_count}/{required_detections} | Detektory: [{methods_str}] | GDOP: {position.get('gdop', 0):.2f}")
            else:
                print(f"[{timestamp}] #{self.request_count} | Satelity: {nsat} | GDOP: {position.get('gdop', 0):.2f} | Detektory: [-]")
        
        if spoofing_detected and any_detection:
            print(f"\nSzczegoly detekcji:")
            for result in results:
                for method, detected in result.detection_methods.items():
                    if detected:
                        confidence = result.confidence
                        prns = result.suspicious_satellites
                        method_name = method.replace('_', ' ').title()
                        print(f"\n  [{method_name}]:")
                        print(f"     Podejrzane PRN: {sorted(prns)}")
                        
                        if method in result.details:
                            details = result.details[method]
                            for key, value in details.items():
                                print(f"     {key}: {value}")
        
        if spoofing_detected:
            print(f"\n")
    
    def log_message(self, format, *args):
        pass


def parse_args():
    parser = argparse.ArgumentParser(
        description='Wykrywanie spoofingu GNSS',
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        '--verbose', '-v',
        action='store_true',
        help='Wyswietlaj szczegolowe informacje'
    )
    parser.add_argument(
        '--no-detection', '-n',
        action='store_true',
        help='Wylacz detekcje spoofingu (tylko odbieranie danych)'
    )
    parser.add_argument(
        '--port', '-p',
        type=int,
        default=1234,
        help='Port serwera HTTP (domyslnie: 1234)'
    )
    parser.add_argument(
        '--system', '-s',
        type=str,
        default='g',
        choices=['g', 'a', 'l'],
        help='System GNSS: g (GPS), a (Galileo), l (GLONASS) - domyslnie: g'
    )
    return parser.parse_args()


def main():
    args = parse_args()
    
    if not args.no_detection:
        system_names = {'g': 'GPS L1 C/A', 'a': 'Galileo E1B', 'l': 'GLONASS G1'}
        print(f"System: {system_names[args.system]}")
        SpoofingDetectionHandler.detector = SpoofingDetector(system=args.system)
    else:
        print("[WARNING] Detekcja spoofingu wylaczona (--no-detection)")
    
    enable_detection = not args.no_detection
    SpoofingDetectionHandler.verbose = args.verbose
    SpoofingDetectionHandler.enable_detection = enable_detection
    
    # Uruchom serwer HTTP
    server_address = ('127.0.0.1', args.port)
    httpd = HTTPServer(server_address, SpoofingDetectionHandler)
    

    print(f"Adres: http://127.0.0.1:{args.port}")
    print(f"Verbose: {'Tak' if args.verbose else 'Nie'}")
    print(f"\n[OK]\n")
    
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        httpd.server_close()


if __name__ == '__main__':
    main()
