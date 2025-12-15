import numpy as np
import matplotlib.pyplot as plt
from scipy.signal import medfilt

# --- KONFIGURACJA ---
FILENAME = "/home/szymon/Downloads/jammerAfterMinute.cu8"  # Plik .bin (uint8 I/Q)
SAMPLE_RATE = 2048000       # Częstotliwość próbkowania (taka jak w nagraniu/inspectrum)
START_TIME = 80.0            # W której sekundzie zacząć analizę?
DURATION = 30             # SKRÓCONE: 50ms (wystarczy do analizy, 100x szybsze!)

def load_cu8_snippet(filename, start_time, duration, sample_rate):
    """Wczytuje fragment pliku .bin (uint8 I/Q) i konwertuje na liczby zespolone."""
    offset = int(start_time * sample_rate)
    count = int(duration * sample_rate)
    
    # Rozmiar próbki: 2 bajty (1 bajt I, 1 bajt Q)
    bytes_offset = offset * 2
    
    print(f"[LOAD] Wczytuję {count} próbek od pozycji {bytes_offset} bajtów...")
    
    try:
        with open(filename, 'rb') as f:
            f.seek(bytes_offset)
            # Wczytujemy 2x tyle bajtów co próbek (bo I i Q)
            raw_data = np.fromfile(f, dtype=np.uint8, count=count*2)
            
        if len(raw_data) == 0:
            raise ValueError("Nie wczytano danych. Sprawdź czy plik istnieje i czy START_TIME nie jest za duży.")

        print(f"[LOAD] Wczytano {len(raw_data)} bajtów → {len(raw_data)//2} próbek IQ")
        
        # Konwersja formatu uint8 (0-255) na float (-1.0 do 1.0)
        # Format: I, Q, I, Q...
        i_samples = (raw_data[0::2].astype(np.float32) - 127.5) / 127.5
        q_samples = (raw_data[1::2].astype(np.float32) - 127.5) / 127.5
        
        return i_samples + 1j * q_samples
        
    except FileNotFoundError:
        print(f"BŁĄD: Nie znaleziono pliku '{filename}'")
        exit()

def analyze_signal(signal, fs):
    print(f"[ANALYZE] Analiza {len(signal)} próbek...")
    
    # 1. Obliczanie chwilowej częstotliwości (pochodna fazy)
    # Mnożymy próbkę przez sprzężenie poprzedniej próbki
    print("[ANALYZE] Obliczam chwilową częstotliwość...")
    phase_diff = np.angle(signal[1:] * np.conj(signal[:-1]))
    
    # Konwersja na Hz
    inst_freq = phase_diff * fs / (2 * np.pi)
    
    # 2. Wygładzanie (filtr medianowy), żeby usunąć szum
    print("[ANALYZE] Wygładzanie filtrem medianowym...")
    freq_clean = medfilt(inst_freq, kernel_size=51)
    
    # 3. Wyznaczanie MIN i MAX (używamy percentyli, żeby pominąć pojedyncze "szpilki" błędów)
    f_min = np.percentile(freq_clean, 2)
    f_max = np.percentile(freq_clean, 98)
    bandwidth = f_max - f_min
    
    # 4. Wykrywanie okresu (Period) za pomocą autokorelacji
    print("[ANALYZE] Obliczam autokorelację (to może chwilę potrwać)...")
    # Odejmujemy średnią, żeby autokorelacja zadziałała poprawnie
    freq_centered = freq_clean - np.mean(freq_clean)
    
    # OPTYMALIZACJA: Używamy FFT (100x szybsze niż np.correlate!)
    from scipy.signal import correlate
    corr = correlate(freq_centered, freq_centered, mode='full', method='fft')
    corr = corr[len(corr)//2:] # Bierzemy tylko prawą połowę
    
    print("[ANALYZE] Szukam piku w autokorelacji...")
    # Szukamy pierwszego dużego piku (pomijając sam początek czyli lag=0)
    # Szukamy piku w zakresie od 100 próbek w górę
    peaks_start_idx = 100
    if len(corr) > peaks_start_idx:
        peak_idx = np.argmax(corr[peaks_start_idx:]) + peaks_start_idx
        period_samples = peak_idx
        period_seconds = period_samples / fs
        sweep_freq_hz = 1.0 / period_seconds
        print(f"[ANALYZE] Znaleziono okres: {period_seconds*1000:.2f} ms")
    else:
        period_seconds = 0
        sweep_freq_hz = 0
        print("[ANALYZE] Nie znaleziono wyraźnego okresu")

    return bandwidth, sweep_freq_hz, period_seconds, freq_clean

# --- GŁÓWNA CZĘŚĆ ---
print(f"--- Analiza pliku: {FILENAME} ---")
data = load_cu8_snippet(FILENAME, START_TIME, DURATION, SAMPLE_RATE)
bw, sweep_rate, period, freq_plot = analyze_signal(data, SAMPLE_RATE)

print("\n=== WYNIKI ANALIZY ===")
print(f"Zmierzone Pasmo (Bandwidth): {bw/1000:.2f} kHz ({bw/1e6:.4f} MHz)")
print(f"Okres powtarzania (Period):  {period*1000:.2f} ms")
print(f"Częstotliwość Sweepu:        {sweep_rate:.2f} Hz")

print("\n=== CO WPISAĆ W GNU RADIO? ===")
print("Blok: Signal Source (Sawtooth)")
print(f" -> Frequency:   {sweep_rate:.2f}")
print("-" * 30)
print("Blok: Multiply Const (przed VCO)")
print(f" -> Constant:    {bw/2:.2f}")
print("   (Przy założeniu, że VCO Sensitivity = 6.28318)")

# Rysowanie wykresu dla weryfikacji
time_axis = np.linspace(0, DURATION, len(freq_plot))
plt.figure(figsize=(10, 5))
plt.plot(time_axis, freq_plot, label='Chwilowa częstotliwość')
plt.title(f"Analiza Jammera (Wycinek {DURATION}s)")
plt.xlabel("Czas [s]")
plt.ylabel("Częstotliwość [Hz]")
plt.grid(True)
plt.show()