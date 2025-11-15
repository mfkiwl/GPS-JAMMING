#!/usr/bin/env python3

import csv
import math
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np


def haversine_distance(lat1, lon1, lat2, lon2):
    R = 6371000
    
    lat1_rad = math.radians(lat1)
    lat2_rad = math.radians(lat2)
    delta_lat = math.radians(lat2 - lat1)
    delta_lon = math.radians(lon2 - lon1)
    
    a = math.sin(delta_lat / 2)**2 + math.cos(lat1_rad) * math.cos(lat2_rad) * math.sin(delta_lon / 2)**2
    c = 2 * math.asin(math.sqrt(a))
    
    return R * c


def process_csv_file(filepath):
    time_to_first_fix = None
    position_errors = []
    
    try:
        with open(filepath, 'r') as f:
            reader = csv.DictReader(f)
            for row in reader:
                try:
                    elapsed_time = float(row['elapsed_time'])
                    lat = float(row['lat'])
                    lon = float(row['lon'])
                    
                    if (lat != 0.0 or lon != 0.0) and time_to_first_fix is None:
                        time_to_first_fix = elapsed_time
                    
                    if time_to_first_fix is not None:
                        position_errors.append((elapsed_time, lat, lon))
                except (ValueError, KeyError):
                    continue
    except Exception as e:
        print(f"Błąd: {filepath}: {e}")
        return None, []
    
    return time_to_first_fix, position_errors


def calculate_mean_error(position_errors, ref_lat, ref_lon):
    if not position_errors:
        return None
    
    errors = [haversine_distance(ref_lat, ref_lon, lat, lon) 
              for _, lat, lon in position_errors if (lat != 0.0 or lon != 0.0)]
    
    return np.mean(errors) if errors else None


def main():
    files_with_positions = {
        '/home/tux/bald/rec/capture01.csv': (49.775528, 19.658722),
        '/home/tux/bald/rec/capture03.csv': (49.775528, 19.658722),
        '/home/tux/bald/rec/capture08.csv': (50.017244, 19.940212),
        '/home/tux/bald/rec/capture11.csv': (50.017244, 19.940212),
        '/home/tux/bald/rec/capture14.csv': (50.017244, 19.940212),
        '/home/tux/bald/rec/capture18.csv': (50.017244, 19.940212),
        '/home/tux/bald/rec/capture23.csv': (50.017244, 19.940212),
        '/home/tux/bald/rec/capture27.csv': (50.017244, 19.940212),
        '/home/tux/bald/rec/capture30.csv': (50.017244, 19.940212),
    }
    
    results = {}
    for filepath, (ref_lat, ref_lon) in files_with_positions.items():
        time_to_fix, position_errors = process_csv_file(filepath)
        
        if time_to_fix is not None:
            mean_error = calculate_mean_error(position_errors, ref_lat, ref_lon)
            filename = Path(filepath).name
            results[filename] = {
                'time_to_first_fix': time_to_fix,
                'mean_position_error': mean_error
            }
    
    if not results:
        return
    
    filenames = list(results.keys())
    times_to_fix = [results[f]['time_to_first_fix'] for f in filenames]
    mean_errors = [results[f]['mean_position_error'] for f in filenames]
    
    avg_time = np.mean(times_to_fix)
    avg_error = np.mean([e for e in mean_errors if e is not None])
    
    fig1, ax1 = plt.subplots(figsize=(12, 6))
    bars1 = ax1.bar(range(len(filenames)), times_to_fix, color='steelblue', alpha=0.7)
    ax1.axhline(y=avg_time, color='red', linestyle='--', linewidth=2, label=f'Średnia: {avg_time:.2f}s')
    ax1.set_xlabel('Plik CSV', fontsize=12)
    ax1.set_ylabel('Czas (sekundy)', fontsize=12)
    ax1.set_title('Czas do otrzymania pierwszej niezerowej pozycji', fontsize=14, fontweight='bold')
    ax1.set_xticks(range(len(filenames)))
    ax1.set_xticklabels(filenames, rotation=45, ha='right')
    ax1.grid(axis='y', alpha=0.3)
    ax1.legend()
    
    for i, (bar, value) in enumerate(zip(bars1, times_to_fix)):
        ax1.text(bar.get_x() + bar.get_width()/2, bar.get_height() * 0.95, 
                f'{value:.2f}s', ha='center', va='top', fontsize=9, color='white', fontweight='bold')
    
    plt.tight_layout()
    plt.savefig('/home/tux/bald/helpers/time_to_first_fix.png', dpi=150, bbox_inches='tight')
    plt.close()
    
    fig2, ax2 = plt.subplots(figsize=(12, 6))
    bars2 = ax2.bar(range(len(filenames)), mean_errors, color='coral', alpha=0.7)
    ax2.axhline(y=avg_error, color='red', linestyle='--', linewidth=2, label=f'Średnia: {avg_error:.2f}m')
    ax2.set_xlabel('Plik CSV', fontsize=12)
    ax2.set_ylabel('Błąd pozycji (metry)', fontsize=12)
    ax2.set_title('Średni błąd pozycji względem lokacji referencyjnej', fontsize=14, fontweight='bold')
    ax2.set_xticks(range(len(filenames)))
    ax2.set_xticklabels(filenames, rotation=45, ha='right')
    ax2.grid(axis='y', alpha=0.3)
    ax2.legend()
    
    for i, (bar, value) in enumerate(zip(bars2, mean_errors)):
        ax2.text(bar.get_x() + bar.get_width()/2, bar.get_height() * 0.95, 
                f'{value:.2f}m', ha='center', va='top', fontsize=9, color='white', fontweight='bold')
    
    plt.tight_layout()
    plt.savefig('/home/tux/bald/helpers/mean_position_error.png', dpi=150, bbox_inches='tight')
    plt.close()


if __name__ == '__main__':
    main()
