#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import numpy as np
import matplotlib.pyplot as plt

# --- Konfiguracja ---
# Zmień na ścieżkę do swojego pliku z próbkami IQ
# Plik powinien zawierać surowe dane w formacie uint8: [I0, Q0, I1, Q1, ...]
FILE_PATH = 'final_output_z_zagluszeniem.bin' 

# Ustaw częstotliwość próbkowania (Sample Rate) swojego SDR-a
# np. 2.4e6 dla RTL-SDR (2.4 MS/s)
SAMPLE_RATE = 2.048e6

# Rozmiar FFT do obliczenia widma (musi być potęgą 2)
FFT_SIZE = 1024
# --------------------

def plot_iq_data(file_path, sample_rate, fft_size):
    """
    Wczytuje i plotuje dane IQ z pliku binarnego (uint8).
    """
    try:
        # Wczytaj surowe bajty (uint8) z pliku
        raw_data = np.fromfile(file_path, dtype=np.uint8)
    except FileNotFoundError:
        print(f"Błąd: Nie znaleziono pliku '{file_path}'")
        print("Upewnij się, że zmienna FILE_PATH jest poprawnie ustawiona.")
        return
    except Exception as e:
        print(f"Wystąpił błąd podczas wczytywania pliku: {e}")
        return

    if raw_data.size == 0:
        print("Błąd: Plik jest pusty.")
        return
    
    # Próbki IQ są przeplatane (interleaved), więc liczba próbek musi być parzysta
    if raw_data.size % 2 != 0:
        print("Ostrzeżenie: Plik ma nieparzystą liczbę bajtów. Ostatni bajt zostanie zignorowany.")
        raw_data = raw_data[:-1]

    # --- Krok 1: Konwersja uint8 (0 do 255) na float (-1.0 do 1.0) ---
    # Standardowa konwersja dla SDR-ów (np. RTL-SDR):
    # Wartość '0' jest mapowana na -1.0, '127.5' na 0.0, '255' na 1.0
    float_data = (raw_data.astype(np.float32) - 127.5) / 127.5

    # --- Krok 2: Konwersja na liczby zespolone (complex) ---
    # Utwórz próbki zespolone jawnie z par [I, Q] — bez użycia view(), co może powodować błędy
    # float_data[0::2] (próbki I) + 1j * float_data[1::2] (próbki Q)
    iq_samples = (float_data[0::2] + 1j * float_data[1::2]).astype(np.complex64)

    print(f"Wczytano {iq_samples.size} próbek IQ.")

    # --- Krok 3: Wykres Mocy w Czasie ---
    
    # Moc chwilowa P = I^2 + Q^2 (czyli kwadrat amplitudy |z|^2)
    # Używamy małej stałej (epsilon), aby uniknąć log(0)
    epsilon = 1e-12 
    power_vs_time_db = 10 * np.log10(np.abs(iq_samples)**2 + epsilon)
    
    # Ograniczamy liczbę punktów na wykresie, aby był czytelny
    num_samples_to_plot = min(iq_samples.size, 5000)
    time_axis = np.arange(num_samples_to_plot) / sample_rate

    plt.figure(figsize=(12, 10))

    plt.subplot(2, 1, 1)
    plt.plot(time_axis * 1000, power_vs_time_db[:num_samples_to_plot], alpha=0.7)
    plt.title('Moc sygnału w czasie (pierwsze 5000 próbek)')
    plt.xlabel('Czas (ms)')
    plt.ylabel('Moc (dB)')
    plt.grid(True)

    # --- Krok 4: Wykres Gęstości Widmowej Mocy (PSD / FFT) ---
    
    # Używamy funkcji plt.psd(), która automatycznie:
    # 1. Dzieli sygnał na segmenty (jeśli jest długi)
    # 2. Stosuje okno (domyślnie Hann)
    # 3. Oblicza FFT dla każdego segmentu
    # 4. Uśrednia wyniki (metoda Welcha)
    # 5. Przesuwa widmo (fftshift), aby 0 Hz było na środku
    # 6. Przelicza na skalę dB
    
    plt.subplot(2, 1, 2)
    plt.psd(iq_samples, 
            NFFT=fft_size,       # Rozmiar FFT
            Fs=sample_rate,      # Częstotliwość próbkowania
            sides='centered')    # Wyśrodkuj wykres na 0 Hz
    
    plt.title('Widmo sygnału (FFT / PSD)')
    plt.xlabel('Częstotliwość (Hz)')
    plt.ylabel('Gęstość widmowa mocy (dB/Hz)')
    plt.grid(True)

    # Dostosuj układ i pokaż wykresy
    plt.tight_layout()
    plt.show()

if __name__ == '__main__':
    # Upewnij się, że masz plik 'signal.cu8' w tym samym katalogu
    # lub zmień zmienną FILE_PATH na właściwą ścieżkę.
    # Aby przetestować, możesz wygenerować sztuczny plik:
    # Np. w terminalu:
    # head -c 2000000 /dev/urandom > test.cu8
    # i ustawić FILE_PATH = 'test.cu8'
    
    plot_iq_data(FILE_PATH, SAMPLE_RATE, FFT_SIZE)