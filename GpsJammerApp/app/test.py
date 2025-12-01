import numpy as np
import sys
import os

CHUNK_SIZE_BYTES = 131072

def count_samples_in_file(file_path: str) -> int:
    total_samples = 0
    try:
        with open(file_path, 'rb') as f:
            while True:
                raw_bytes = f.read(CHUNK_SIZE_BYTES)
                if not raw_bytes:
                    break
                raw_chunk_uint8 = np.frombuffer(raw_bytes, dtype=np.uint8)
                num_new_samples_in_chunk = raw_chunk_uint8.size // 2
                total_samples += num_new_samples_in_chunk
        return total_samples
    except Exception as e:
        print(f"Błąd podczas liczenia próbek: {e}")
        return -1

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"Użycie: python {os.path.basename(__file__)} <nazwa_pliku.bin>")
        sys.exit(1)
    SDR_FILE_PATH = sys.argv[1]
    if not os.path.exists(SDR_FILE_PATH):
        print(f"BŁĄD: Nie znaleziono pliku: {SDR_FILE_PATH}")
        sys.exit(1)
    total_samples = count_samples_in_file(SDR_FILE_PATH)
    if total_samples >= 0:
        print(f"Liczba próbek w pliku: {total_samples}")
    else:
        print("Wystąpił błąd podczas liczenia próbek.")
