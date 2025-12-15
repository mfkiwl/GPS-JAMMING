#!/usr/bin/env python3
"""
Test skrypt do sprawdzania detekcji jammingu bez GUI - batch processing
"""
import sys
import os
from PySide6.QtCore import QCoreApplication

# Dodaj ścieżkę do modułu app
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

from worker import GPSAnalysisThread

# Globalne zmienne dla batch processing
current_file_index = 0
test_files = []
results_summary = []

def find_test_files(base_folder):
    """Znajduje wszystkie pliki capture_ruch(1-10).bin w /Downloads"""
    files = []
    base_path = "/home/szymon/Downloads/GPS_JAMMING/GPS-JAMMING/GpsJammerApp"
    
    if not os.path.exists(base_path):
        return files
    
    # Szukaj plików capture_ruch1.bin do capture_ruch10.bin
    for i in range(1, 11):
        filename = f"test25_{i}.bin"
        file_path = os.path.join(base_path, filename)
        
        if os.path.exists(file_path):
            files.append({
                'path': file_path,
                'folder': 'Downloads',
                'name': filename
            })
    
    return files

# Globalna referencja do aktualnego workera
current_worker = None

def process_next_file(app):
    """Przetwarza następny plik z listy"""
    global current_file_index, test_files, results_summary, current_worker
    
    if current_file_index >= len(test_files):
        # Wszystkie pliki przetworzone - pokaż podsumowanie
        print(f"\n{'='*60}")
        print("📊 PODSUMOWANIE WYNIKÓW")
        print(f"{'='*60}\n")
        
        for result in results_summary:
            status = "✅ TAK" if result['jamming_detected'] else "❌ NIE"
            print(f"{status} - {result['file']}")
        
        print(f"\n{'='*60}\n")
        app.quit()
        return
    
    # Pobierz aktualny plik
    current_file = test_files[current_file_index]
    print(f"\n[{current_file_index + 1}/{len(test_files)}] Sprawdzam: {current_file['name']}")
    
    # Utwórz wątek analizy - trzymamy referencję globalnie!
    current_worker = GPSAnalysisThread(
        file_paths=[current_file['path']],
        power_threshold=6.0,
        satellite_system='GPS',
        hold_position=False
    )
    
    # Zmienna do śledzenia czy wykryto jamming
    jamming_detected = [False]  # Lista żeby móc modyfikować w callback
    
    # Callback na zakończenie
    def on_complete(results):
        global current_file_index, current_worker
        
        # Sprawdź czy był jamming
        has_jamming = False
        if results:
            for result in results:
                if result.get('type') == 'jamming':
                    has_jamming = True
                    break
        
        # Zapisz wynik
        results_summary.append({
            'file': current_file['name'],
            'jamming_detected': has_jamming or jamming_detected[0]
        })
        
        # Poczekaj aż wątek się zakończy
        if current_worker:
            current_worker.wait()
            current_worker = None
        
        # Przejdź do następnego pliku
        current_file_index += 1
        process_next_file(app)
    
    # Callback na detekcję realtime
    def on_jamming_realtime(is_jamming, position):
        if is_jamming:
            jamming_detected[0] = True
    
    # Podłącz sygnały
    current_worker.analysis_complete.connect(on_complete)
    current_worker.jamming_detected_realtime.connect(on_jamming_realtime)
    
    # Uruchom analizę
    current_worker.start()

def main():
    global test_files
    
    print(f"🔍 Testowanie detekcji jammingu")
    print(f"{'='*60}\n")
    
    # Sprawdź czy podano plik jako argument
    if len(sys.argv) > 1:
        # Tryb pojedynczego pliku
        test_file = sys.argv[1]
        
        if not os.path.exists(test_file):
            print(f"❌ Plik nie istnieje: {test_file}")
            return 1
        
        test_files = [{
            'path': test_file,
            'folder': os.path.dirname(test_file),
            'name': os.path.basename(test_file)
        }]
        
        print(f"Tryb: POJEDYNCZY PLIK")
        print(f"Plik: {test_file}\n")
    else:
        # Tryb batch - wszystkie pliki capture_ruch*.bin
        test_files = find_test_files("/home/szymon/Downloads")
        
        if not test_files:
            print("❌ Nie znaleziono plików do analizy!")
            print("Użycie: python test_worker.py [plik.bin]")
            print("  Bez argumentu: przetwarza pliki capture_ruch1.bin do capture_ruch10.bin")
            print("  Z argumentem: przetwarza tylko podany plik")
            return 1
        
        print(f"Tryb: BATCH PROCESSING")
        print(f"Znaleziono {len(test_files)} plików do sprawdzenia:\n")
        for f in test_files:
            print(f"  - {f['name']}")
        print()
    
    # Inicjalizacja Qt
    app = QCoreApplication(sys.argv)
    
    # Uruchom przetwarzanie pierwszego pliku
    process_next_file(app)
    
    # Uruchom event loop
    return app.exec()

if __name__ == "__main__":
    sys.exit(main())
