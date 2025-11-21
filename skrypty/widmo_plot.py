import numpy as np
import matplotlib.pyplot as plt
from scipy import signal
import os

# --- KONFIGURACJA ---
FILENAME = '/home/szymon/Downloads/capture_ruch10.bin'  # Zmień na nazwę pliku
SAMPLE_RATE = 2.048e6           # 2.048 MSps
CHUNK_SIZE = int(SAMPLE_RATE)   # Analizujemy 1 sekundę na raz (jako jeden "pasek" wykresu)
FFT_SIZE = 1024                 # Rozdzielczość częstotliwościowa (szerokość wykresu)

def analyze_full_file(filename):
    file_size = os.path.getsize(filename)
    total_samples = file_size // 2 # 2 bajty na próbkę (I+Q)
    duration_sec = total_samples / SAMPLE_RATE
    
    print(f"Analiza pliku: {filename}")
    print(f"Rozmiar: {file_size/1024/1024:.2f} MB")
    print(f"Czas trwania: {duration_sec:.2f} sekund")
    print("Przetwarzanie... to może chwilę potrwać.")

    # Przygotowanie tablic na wyniki
    spectrogram_data = []
    histogram_samples = [] # Weźmiemy próbki losowo do histogramu
    
    with open(filename, 'rb') as f:
        while True:
            # Czytamy kawałek (CHUNK) - np. 1 sekundę
            raw_chunk = np.fromfile(f, dtype=np.uint8, count=CHUNK_SIZE * 2)
            
            if len(raw_chunk) < FFT_SIZE * 2:
                break # Koniec pliku

            # Zbieranie próbek do histogramu (bierzemy co 100-tną próbkę, żeby nie zapchać pamięci, ale mieć reprezentację całości)
            histogram_samples.extend(raw_chunk[::100])

            # Konwersja na zespolone
            raw_chunk = raw_chunk.astype(np.float32)
            i = (raw_chunk[0::2] - 127.5) / 127.5
            q = (raw_chunk[1::2] - 127.5) / 127.5
            complex_chunk = i + 1j * q
            
            # Usunięcie DC offset (dla danego kawałka)
            complex_chunk = complex_chunk - np.mean(complex_chunk)

            # Obliczenie PSD (widma) dla tego kawałka czasu
            # Używamy Welcha, żeby wygładzić szum w obrębie tej 1 sekundy
            f_axis, Pxx = signal.welch(complex_chunk, SAMPLE_RATE, nperseg=FFT_SIZE, return_onesided=False)
            
            # Shift i Logarytm
            Pxx = np.fft.fftshift(Pxx)
            Pxx_db = 10 * np.log10(Pxx + 1e-15)
            
            spectrogram_data.append(Pxx_db)

    # Konwersja listy na macierz 2D (Czas x Częstotliwość)
    spectrogram_array = np.array(spectrogram_data)
    
    # --- RYSOWANIE ---
    fig = plt.figure(figsize=(12, 10))
    gs = fig.add_gridspec(3, 1, height_ratios=[3, 1, 1])

    # 1. WODOSPAD (Spectrogram) - Cały plik
    ax1 = fig.add_subplot(gs[0])
    # Oś X: Częstotliwość (MHz), Oś Y: Czas (s)
    extent = [-SAMPLE_RATE/2/1e6, SAMPLE_RATE/2/1e6, duration_sec, 0]
    im = ax1.imshow(spectrogram_array, aspect='auto', extent=extent, cmap='inferno', interpolation='nearest')
    ax1.set_title(f'Pełny Spektrogram (Wodospad) - {duration_sec:.1f}s')
    ax1.set_ylabel('Czas [s]')
    ax1.set_xlabel('Częstotliwość [MHz]')
    plt.colorbar(im, ax=ax1, label='Moc [dB]')

    # 2. ŚREDNIE WIDMO (Z całego pliku)
    ax2 = fig.add_subplot(gs[1])
    mean_spectrum = np.mean(spectrogram_array, axis=0) # Średnia po czasie
    freq_axis = np.linspace(-SAMPLE_RATE/2/1e6, SAMPLE_RATE/2/1e6, FFT_SIZE)
    ax2.plot(freq_axis, mean_spectrum, color='blue')
    ax2.set_title('Średnie Widmo z całego nagrania')
    ax2.set_ylabel('Moc [dB]')
    ax2.grid(True)
    ax2.set_xlim(freq_axis[0], freq_axis[-1])

    # 3. HISTOGRAM (Z próbek z całego pliku)
    ax3 = fig.add_subplot(gs[2])
    ax3.hist(histogram_samples, bins=256, range=(0, 256), color='green', alpha=0.7, density=True)
    ax3.set_title('Histogram (Reprezentatywny dla całego pliku)')
    ax3.set_xlabel('Wartość surowa (0-255)')
    ax3.axvline(0, color='red', linestyle='--')
    ax3.axvline(255, color='red', linestyle='--')
    ax3.grid(True, alpha=0.3)

    plt.tight_layout()
    plt.show()

# URUCHOMIENIE
analyze_full_file(FILENAME)