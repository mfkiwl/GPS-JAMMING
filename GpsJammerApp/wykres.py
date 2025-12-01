import numpy as np
import matplotlib.pyplot as plt
import os
import sys
import argparse

# ==========================================================================
# SZYBKA KONFIGURACJA
DEFAULT_FILE_PATH = "" 
# ==========================================================================

def read_chunk(f, chunk_size, dtype_enum):
    """Czyta i konwertuje dane w zależności od formatu."""
    
    # Określenie parametrów na podstawie typu
    if dtype_enum == 'int16':
        # Standard I16 (BladeRF, USRP, gnss-sdr)
        # 2 bajty na składową, 4 bajty na sampel
        bytes_per_sample = 4 
        np_dtype = np.int16
    elif dtype_enum == 'int8':
        # Signed 8-bit (HackRF)
        # 1 bajt na składową, 2 bajty na sampel
        bytes_per_sample = 2
        np_dtype = np.int8
    elif dtype_enum == 'uint8':
        # Unsigned 8-bit (RTL-SDR)
        # 1 bajt na składową, 2 bajty na sampel
        # Wymaga przesunięcia o -127.5
        bytes_per_sample = 2
        np_dtype = np.uint8
    else:
        raise ValueError("Nieznany format danych")

    chunk_bytes = chunk_size * bytes_per_sample
    raw_data = f.read(chunk_bytes)
    
    if not raw_data:
        return None, None
        
    samples = np.frombuffer(raw_data, dtype=np_dtype).astype(np.float32)
    
    # Obsługa RTL-SDR (uint8 -> przesunięcie zera)
    if dtype_enum == 'uint8':
        samples -= 127.5
        
    # Rozdzielenie I / Q
    i_data = samples[0::2]
    q_data = samples[1::2]
    
    # Wyrównanie długości
    min_len = min(len(i_data), len(q_data))
    return i_data[:min_len], q_data[:min_len]

def plot_iq_power(file_path, sampling_rate=2048000, chunk_size=20480, dtype_name='int16'):
    
    if not os.path.exists(file_path):
        print(f"Błąd: Plik '{file_path}' nie istnieje.")
        return

    file_size = os.path.getsize(file_path)
    print(f"Analiza pliku: {file_path}")
    print(f"Format danych: {dtype_name}")
    print(f"Rozmiar: {file_size / (1024*1024):.2f} MB")
    
    power_values = []
    time_values = []
    processed_samples = 0
    
    try:
        with open(file_path, 'rb') as f:
            while True:
                i_data, q_data = read_chunk(f, chunk_size, dtype_name)
                
                if i_data is None:
                    break
                
                if len(i_data) == 0:
                    break

                # Obliczanie mocy: P = I^2 + Q^2
                instant_power = i_data**2 + q_data**2 + 1e-10
                avg_power = np.mean(instant_power)
                
                power_values.append(avg_power)
                
                current_time = (processed_samples + (len(i_data) / 2)) / sampling_rate
                time_values.append(current_time)
                processed_samples += len(i_data)
                
                if len(power_values) % 200 == 0:
                    print(f"\rPrzetworzono: {current_time:.2f}s...", end="")

        print("\nGenerowanie wykresu...")
        
        if not power_values:
            print("Błąd: Brak danych do wyświetlenia.")
            return

        y_raw = np.array(power_values)
        x_time = np.array(time_values)
        
        # Wyznaczanie tła (Noise Floor) - 5. percentyl
        baseline = np.percentile(y_raw, 5) 
        if baseline <= 0: baseline = 1.0
            
        # Konwersja na dB
        y_db = 10 * np.log10(y_raw / baseline)
        
        plt.figure(figsize=(14, 7))
        plt.plot(x_time, y_db, label='Moc (dB)', color='#007acc', linewidth=1)
        
        # Linie progowe
        plt.axhline(y=0, color='black', linestyle='-', alpha=0.3, label='Noise Floor')
        plt.axhline(y=6, color='red', linestyle='--', alpha=0.8, label='Próg Jammingu (+6dB)')
        
        # Znajdź max
        max_idx = np.argmax(y_db)
        max_val = y_db[max_idx]
        plt.plot(x_time[max_idx], max_val, 'ro')
        plt.annotate(f'Max: +{max_val:.1f} dB', xy=(x_time[max_idx], max_val), xytext=(x_time[max_idx], max_val+1))

        plt.title(f'Analiza Mocy IQ ({dtype_name})\nPlik: {os.path.basename(file_path)}')
        plt.xlabel('Czas [s]')
        plt.ylabel('Wzrost mocy względem tła [dB]')
        plt.grid(True, alpha=0.3)
        plt.legend()
        plt.tight_layout()
        plt.show()

    except Exception as e:
        print(f"\nBłąd: {e}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('file', nargs='?', help='Plik .bin')
    parser.add_argument('--rate', type=float, default=2048000, help='Próbkowanie Hz')
    
    # Dodano wybór formatu danych
    parser.add_argument('--dtype', type=str, default='uint8', choices=['int16', 'int8', 'uint8'], 
                        help='Format danych: int16 (domyślny), int8 (HackRF), uint8 (RTL-SDR)')
    
    args = parser.parse_args()
    target_file = args.file or DEFAULT_FILE_PATH
    
    if not target_file:
        # Auto-szukanie
        files = [f for f in os.listdir('.') if f.endswith('.bin')]
        if files:
            print("Pliki w folderze:")
            for i, f in enumerate(files): print(f"{i+1}. {f}")
            try:
                sel = input("Wybierz plik (nr): ")
                target_file = files[int(sel)-1]
            except: pass
            
    if target_file:
        chunk_s = int(args.rate * 0.01) 
        plot_iq_power(target_file, sampling_rate=args.rate, chunk_size=chunk_s, dtype_name=args.dtype)
    else:
        print("Nie podano pliku.")