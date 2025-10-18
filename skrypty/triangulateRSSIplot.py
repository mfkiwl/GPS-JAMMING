import numpy as np
import math
import matplotlib.pyplot as plt
from matplotlib.patches import Circle
from matplotlib.colors import LogNorm

# --- KONFIGURACJA ---

# Ścieżki do plików I/Q
FILE_ANT0 = '17_10/capture0.bin'
FILE_ANT1 = '17_10/capture1.bin'

# Położenie anten w metrach
ANT0_POS = np.array([0.0, 0.0])
ANT1_POS = np.array([0.5, 0.0])

# --- PARAMETRY KALIBRACYJNE ---
CALIBRATED_TX_POWER = 40.0
CALIBRATED_PATH_LOSS_EXPONENT = 3.0

# --- Inne parametry sygnału ---
SIGNAL_FREQUENCY_MHZ = 1575.42
SIGNAL_THRESHOLD = 0.1

# --- PARAMETRY WIZUALIZACJI I OBLICZEŃ ---
NUM_DISTINCT_LOCATIONS = 8
MIN_SEPARATION_DISTANCE = 5.0 # m
GRID_DENSITY = 300
HEATMAP_CONTRAST_FACTOR = 3.0
STAR_COLOUR_INTENSITY = 1  # Intensywność kontrastu między kolejnymi gwiazdami 

# --- FUNKCJE POMOCNICZE ---

def read_iq_data(filename):
    try:
        raw_data = np.fromfile(filename, dtype=np.uint8)
        float_data = (raw_data.astype(np.float32) - 127.5) / 127.5
        complex_data = float_data[0::2] + 1j * float_data[1::2]
        return complex_data
    except FileNotFoundError: return None

def find_change_point(amplitude_data, threshold):
    change_indices = np.where(amplitude_data > threshold)[0]
    return change_indices[0] if len(change_indices) > 0 else None

def calculate_distance_from_file(iq_filename):
    print(f"--- Analizowanie pliku '{iq_filename}' ---")
    iq_samples = read_iq_data(iq_filename)
    if iq_samples is None or len(iq_samples) == 0: return None
    amplitude = np.abs(iq_samples)
    turn_on_index = find_change_point(amplitude, SIGNAL_THRESHOLD)
    if turn_on_index is not None:
        avg_amplitude = np.mean(amplitude[turn_on_index:])
        if avg_amplitude == 0: return None
        received_power_db = 10 * np.log10(avg_amplitude**2)
        print(f"Sygnał wykryty. Średnia amplituda: {avg_amplitude:.4f}")
        print(f"Hipotetyczna moc odebrana: {received_power_db:.2f} dB")
        path_loss_at_1m = 20 * np.log10(SIGNAL_FREQUENCY_MHZ) - 27.55
        distance = 10 ** ((CALIBRATED_TX_POWER - received_power_db - path_loss_at_1m) / (10 * CALIBRATED_PATH_LOSS_EXPONENT))
        print(f">>> Oszacowana odległość: {distance:.2f} m\n")
        return distance
    else: return None

def perform_grid_search(p0, r0, p1, r1, grid_density):
    print(f"Przeszukiwanie siatki {grid_density}x{grid_density}...")
    max_radius = max(r0, r1); center = (p0 + p1) / 2; search_range = max_radius * 1.5
    x = np.linspace(center[0] - search_range, center[0] + search_range, grid_density)
    y = np.linspace(center[1] - search_range, center[1] + search_range, grid_density)
    grid_x, grid_y = np.meshgrid(x, y)
    dist_to_p0 = np.sqrt((grid_x - p0[0])**2 + (grid_y - p0[1])**2)
    dist_to_p1 = np.sqrt((grid_x - p1[0])**2 + (grid_y - p1[1])**2)
    error_grid = np.abs(dist_to_p0 - r0) + np.abs(dist_to_p1 - r1)
    flat_error = error_grid.flatten(); flat_x = grid_x.flatten(); flat_y = grid_y.flatten()
    sorted_indices = np.argsort(flat_error)
    all_sorted_points = np.vstack((flat_x[sorted_indices], flat_y[sorted_indices])).T
    all_sorted_errors = flat_error[sorted_indices]
    return all_sorted_points, all_sorted_errors, grid_x, grid_y, error_grid

def find_distinct_local_minima(sorted_points, sorted_errors, num_locations, min_distance):
    champions = []; champion_errors = []
    for point, error in zip(sorted_points, sorted_errors):
        if len(champions) >= num_locations: break
        is_distinct = all(np.linalg.norm(point - champ) >= min_distance for champ in champions)
        if is_distinct:
            champions.append(point); champion_errors.append(error)
    return champions, champion_errors

# --- ZAKTUALIZOWANA FUNKCJA RYSOWANIA ---

def plot_results(p0, r0, p1, r1, grid_x, grid_y, error_grid, distinct_locations, distinct_errors):
    # Funkcja do rysowania gwiazd 
    fig = plt.figure(figsize=(11, 9))
    ax_main_pos = [0.1, 0.2, 0.7, 0.7]
    ax_cbar_pos = [0.1, 0.08, 0.7, 0.04]
    ax = fig.add_axes(ax_main_pos)
    cax = fig.add_axes(ax_cbar_pos)
    
    # --- Ustawienia skali i mapy kolorów ---
    min_error = error_grid.min()
    vmin = min_error + 1e-9
    vmax = vmin * HEATMAP_CONTRAST_FACTOR
    
    # Instancja normalizacji i mapy kolorów, do heatmapy i gwiazd
    norm = LogNorm(vmin=vmin, vmax=vmax, clip=True)
    cmap = plt.get_cmap('viridis_r')
    
    # --- Rysowanie Heatmapy ---
    heatmap = ax.pcolormesh(grid_x, grid_y, error_grid + 1e-9, norm=norm, cmap=cmap, shading='gouraud')
    cbar = fig.colorbar(heatmap, cax=cax, orientation='horizontal')
    cbar.set_label('Łączny błąd odległości (m) - skala logarymiczna')

    # --- Rysowanie elementów na głównym wykresie ---
    ax.plot(p0[0], p0[1], 'b^', markersize=12, label='Antena 0', markeredgecolor='white')
    ax.plot(p1[0], p1[1], 'g^', markersize=12, label='Antena 1', markeredgecolor='white')
    ax.add_patch(Circle(p0, r0, color='blue', fill=False, linestyle='--', linewidth=2, label=f'Promień 0 ({r0:.2f} m)'))
    ax.add_patch(Circle(p1, r1, color='green', fill=False, linestyle='--', linewidth=2, label=f'Promień 1 ({r1:.2f} m)'))

    # --- Logika kolorów gwiazd ---
    for i, (point, error) in enumerate(zip(distinct_locations, distinct_errors)):
        # Znormalizuj błąd tej gwiazdy używając tej samej skali co heatmapa
        normalized_error = norm(error + i * STAR_COLOUR_INTENSITY)
        # Pobierz kolor z mapy
        star_color = cmap(normalized_error)
        
        label = 'Najlepsza estymacja' if i == 0 else f'Lokalne minimum #{i+1}'
        ax.plot(point[0], point[1], '*', color=star_color, markersize=25, label=label, markeredgecolor='black')

    handles, labels = ax.get_legend_handles_labels()
    ax.legend(handles, labels, loc='upper left', bbox_to_anchor=(1.02, 1.0), title="Legenda")

    ax.set_aspect('equal', adjustable='box'); ax.grid(True, linestyle=':', alpha=0.5)
    ax.set_xlabel('Współrzędna X (m)'); ax.set_ylabel('Współrzędna Y (m)')
    ax.set_title('Mapa prawdopodobieństwa lokalizacji nadajnika');


if __name__ == "__main__":
    print("Rozpoczynanie lokalizacji na podstawie mocy sygnału (RSSI).\n")
    dist0 = calculate_distance_from_file(FILE_ANT0)
    dist1 = calculate_distance_from_file(FILE_ANT1)
    if dist0 is None or dist1 is None: exit()

    all_points, all_errors, grid_x, grid_y, error_grid = perform_grid_search(
        ANT0_POS, dist0, ANT1_POS, dist1, GRID_DENSITY
    )
    
    distinct_locations, distinct_errors = find_distinct_local_minima(
        all_points, all_errors, NUM_DISTINCT_LOCATIONS, MIN_SEPARATION_DISTANCE
    )
    
    print("\n--- Obliczanie lokalizacji ---")
    print(f">>> Znaleziono {len(distinct_locations)} najbardziej prawdopodobnych, odseparowanych lokalizacji:")
    for i, (loc, err) in enumerate(zip(distinct_locations, distinct_errors)):
        print(f"    Lokalizacja #{i+1}: x = {loc[0]:.2f} m, y = {loc[1]:.2f} m (Błąd: {err:.3f})")
            
    print("\nGenerowanie wykresu...")
    plot_results(ANT0_POS, dist0, ANT1_POS, dist1, grid_x, grid_y, error_grid, distinct_locations, distinct_errors)
    plt.show()