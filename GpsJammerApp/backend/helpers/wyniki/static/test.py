import glob
import json
import math
from pathlib import Path

# === USTAW SWOJĄ POZYCJĘ REFERENCYJNĄ (WGS84) ===
REF_LAT_DEG = 50.0172546  # stopnie
REF_LON_DEG = 19.9402644  # stopnie
REF_HGT_M   = 225.0        # metry

# -----------------------------------------
# Funkcja licząca błąd pozycji 2D i 3D w metrach
# -----------------------------------------
def position_errors_m(lat_deg, lon_deg, hgt_m,
                      ref_lat_deg=REF_LAT_DEG,
                      ref_lon_deg=REF_LON_DEG,
                      ref_hgt_m=REF_HGT_M):
    # zamiana na radiany
    lat = math.radians(lat_deg)
    lon = math.radians(lon_deg)
    ref_lat = math.radians(ref_lat_deg)
    ref_lon = math.radians(ref_lon_deg)

    # promień Ziemi (przybliżenie)
    R = 6378137.0

    dlat = lat - ref_lat
    dlon = lon - ref_lon
    mean_lat = 0.5 * (lat + ref_lat)

    # lokalne N/E/U (metry)
    dN = dlat * R
    dE = dlon * R * math.cos(mean_lat)
    dU = hgt_m - ref_hgt_m

    err_2d = math.sqrt(dN * dN + dE * dE)
    err_3d = math.sqrt(err_2d * err_2d + dU * dU)
    return err_2d, err_3d

# -----------------------------------------
# Funkcja parsująca pojedynczy plik
# -----------------------------------------
def process_file(path: Path,
                 ref_lat_deg=REF_LAT_DEG,
                 ref_lon_deg=REF_LON_DEG,
                 ref_hgt_m=REF_HGT_M):
    with path.open("r", encoding="utf-8") as f:
        content = f.read()

    # Bloki są oddzielone liniami z '='
    blocks = content.split("=" * 80)

    pos_errors_2d = []   # lista błędów 2D [m]
    pos_errors_3d = []   # lista błędów 3D [m]
    snr_values = []      # lista wszystkich SNR > 0

    for block in blocks:
        # Szukamy JSON-a (pierwszy '{' do ostatniej '}')
        start = block.find("{")
        end = block.rfind("}")
        if start == -1 or end == -1 or end <= start:
            continue

        json_str = block[start:end + 1].strip()

        try:
            data = json.loads(json_str)
        except json.JSONDecodeError:
            # jeśli coś się nie sparsuje, pomijamy blok
            continue

        # --- Pozycja ---
        pos = data.get("position", {})
        lat = pos.get("lat", 0.0)
        lon = pos.get("lon", 0.0)
        hgt = pos.get("hgt", 0.0)

        # Ignorujemy rekord, jeśli pozycja 0 (brak fixa)
        if not (lat == 0.0 and lon == 0.0 and hgt == 0.0):
            err_2d, err_3d = position_errors_m(
                lat, lon, hgt,
                ref_lat_deg, ref_lon_deg, ref_hgt_m
            )
            pos_errors_2d.append(err_2d)
            pos_errors_3d.append(err_3d)

        # --- Observations / SNR ---
        observations = data.get("observations", [])
        for obs in observations:
            snr = obs.get("snr", 0.0)
            # ignorujemy brak locka (snr == 0)
            if snr and snr > 0:
                snr_values.append(snr)

    return pos_errors_2d, pos_errors_3d, snr_values

# -----------------------------------------
# Główna część – przelicz wszystkie test_*.txt
# -----------------------------------------
def main():
    files = sorted(Path(".").glob("capture*.txt"))
    if not files:
        print("Brak plików capture*.txt w bieżącym katalogu.")
        return

    # Łączne statystyki
    all_pos_errors_2d = []
    all_pos_errors_3d = []
    all_snr_values = []

    print("===== STATYSTYKI DLA POSZCZEGÓLNYCH PLIKÓW =====")

    for f in files:
        pos2d, pos3d, snrs = process_file(f)

        all_pos_errors_2d.extend(pos2d)
        all_pos_errors_3d.extend(pos3d)
        all_snr_values.extend(snrs)

        if pos2d:
            mean_2d = sum(pos2d) / len(pos2d)
        else:
            mean_2d = float("nan")

        if pos3d:
            mean_3d = sum(pos3d) / len(pos3d)
        else:
            mean_3d = float("nan")

        if snrs:
            mean_snr = sum(snrs) / len(snrs)
        else:
            mean_snr = float("nan")

        print(f"\nPlik: {f.name}")
        print(f"  Liczba próbek pozycji (z fixa): {len(pos3d)}")
        print(f"  Liczba obserwacji SNR > 0:      {len(snrs)}")
        print(f"  Średni błąd 2D:                 {mean_2d:.3f} m")
        print(f"  Średni błąd 3D:                 {mean_3d:.3f} m")
        print(f"  Średni SNR:                     {mean_snr:.2f} dB-Hz")

    # ===== PODSUMOWANIE ŁĄCZNE =====
    if all_pos_errors_2d:
        mean_2d_all = sum(all_pos_errors_2d) / len(all_pos_errors_2d)
    else:
        mean_2d_all = float("nan")

    if all_pos_errors_3d:
        mean_3d_all = sum(all_pos_errors_3d) / len(all_pos_errors_3d)
    else:
        mean_3d_all = float("nan")

    if all_snr_values:
        mean_snr_all = sum(all_snr_values) / len(all_snr_values)
    else:
        mean_snr_all = float("nan")

    print("\n===== PODSUMOWANIE ŁĄCZNE (WSZYSTKIE PLIKI) =====")
    print(f"Łączna liczba próbek pozycji (z fixa): {len(all_pos_errors_3d)}")
    print(f"Łączna liczba obserwacji SNR > 0:      {len(all_snr_values)}")
    print(f"Średni błąd 2D (wszystkie pliki):      {mean_2d_all:.3f} m")
    print(f"Średni błąd 3D (wszystkie pliki):      {mean_3d_all:.3f} m")
    print(f"Średni SNR (wszystkie pliki):          {mean_snr_all:.2f} dB-Hz")

if __name__ == "__main__":
    main()
