import numpy as np

SAMPLING_RATE = 3182239  # 2 mozliwosc 3182239
DELAY_SECONDS = 60       # po ilu sekundach jammer
DURATION_SECONDS = 15    # jak długo ma działać


# Oblicz indeksy w tablicy (w bajtach, bo I i Q są przeplatane)
start_index = int(SAMPLING_RATE * DELAY_SECONDS * 2)
duration_samples = int(SAMPLING_RATE * DURATION_SECONDS * 2)
end_index = start_index + duration_samples

gps_data = np.fromfile('test.bin', dtype=np.int8).astype(np.float32)
jammer_data = np.fromfile('test_jammer.bin', dtype=np.int8).astype(np.float32)

gps_slaby = gps_data * 0.03

# 3. Zastosuj logikę mocy jammera
skalar_mocy_jammera = 1.0 # Możesz też to zmieniać

# --- Logika opóźnienia i czasu trwania jammera ---
# Stwórz nową, "cichą" tablicę o tej samej długości co sygnał GPS
jammer_with_delay = np.zeros_like(gps_slaby)

# Sprawdź, ile danych jammera mamy, a ile potrzebujemy
available_jammer_len = len(jammer_data)
needed_jammer_len = duration_samples

# Oblicz, ile próbek jammera faktycznie skopiujemy
# (jeśli plik jammera jest krótszy niż 30s, weźmiemy cały plik)
jammer_copy_len = min(available_jammer_len, needed_jammer_len)

# Oblicz, ile mamy miejsca w pliku docelowym
space_available = len(gps_slaby) - start_index

# Ostateczna długość do skopiowania (nie możemy wyjść poza plik docelowy)
final_copy_len = min(jammer_copy_len, space_available)

# Upewnij się, że jest co kopiować
if final_copy_len > 0:
    print(f"Dodaję jammer od {DELAY_SECONDS}s do {DELAY_SECONDS + (final_copy_len / (SAMPLING_RATE * 2)):.2f}s")
    
    # Wklej dane jammera do tablicy w odpowiednim "oknie" czasowym
    jammer_with_delay[start_index : start_index + final_copy_len] = \
        jammer_data[:final_copy_len] * skalar_mocy_jammera
else:
    print("Ostrzeżenie: Plik GPS jest za krótki, aby dodać jammer po 60 sekundach.")
# ------------------------------------

# 4. Połącz sygnały
# (gps_slaby jest dodawany do zer przez pierwsze 60s i po 90s)
sygnal_wynikowy_float = gps_slaby + jammer_with_delay

# 5. Przytnij (clip) i konwertuj z powrotem do int8
sygnal_wynikowy_float = np.clip(sygnal_wynikowy_float, -128.0, 127.0)
final_signal_int8 = sygnal_wynikowy_float.astype(np.int8)

# 6. Konwertuj na uint8 dla gnss-sdrlib
final_signal_uint8 = (final_signal_int8.astype(np.int16) + 128).astype(np.uint8)

# 7. Zapisz finalny plik
final_signal_uint8.tofile('final_output_z_zagluszeniem.bin')
