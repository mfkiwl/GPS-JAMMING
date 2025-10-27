import numpy as np

# 1. Wczytaj swój plik int8 (lub użyj go jako zmiennej)
# Załóżmy, że masz go w zmiennej: final_signal_int8
final_signal_int8 = np.fromfile('jammer_int8.bin', dtype=np.int8)

# 2. Konwertuj na większy typ (int16), aby uniknąć błędów
# (np. 127 + 128 = 255, co nie mieści się w int8)
final_signal_16 = final_signal_int8.astype(np.int16)

# 3. TO JEST NAPRAWA: Dodaj 128
# (Przesuwasz sygnał z centrum 0 na centrum 128)
final_signal_16_offset = final_signal_16 + 128

# 4. Teraz bezpiecznie konwertuj do uint8
final_signal_uint8 = final_signal_16_offset.astype(np.uint8)

# 5. Zapisz poprawny plik
final_signal_uint8.tofile('jammer_poprawny_uint8.bin')