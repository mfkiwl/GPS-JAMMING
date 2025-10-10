import argparse
import csv

def linear_trajectory(start_lat, start_lon, start_alt,
                      end_lat, end_lon, end_alt,
                      duration_s, step_s, out_file):
    """generates linear trajectory and saves into .csv file."""
    if duration_s <= 0:
        raise ValueError("time must be > 0")
    if step_s <= 0:
        raise ValueError("step must be > 0")

    n_steps = int(round(duration_s / step_s))
    rows = []

    for i in range(n_steps + 1):
        t = round(i * step_s, 6)
        if i == n_steps:
            t = float(duration_s)

        frac = t / duration_s
        lat = start_lat + (end_lat - start_lat) * frac
        lon = start_lon + (end_lon - start_lon) * frac
        alt = start_alt + (end_alt - start_alt) * frac

        rows.append([t, round(lat, 9), round(lon, 9), round(alt, 3)])

    with open(out_file, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerows(rows)

    print(f"[generate_trajectory] Zapisano {len(rows)} wierszy do {out_file}")
    print(f"Start: ({start_lat}, {start_lon}, {start_alt}) → "
          f"Koniec: ({end_lat}, {end_lon}, {end_alt}), czas={duration_s}s, krok={step_s}s")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="generates linear trajectory and saves into .csv file.")
    parser.add_argument("--start-lat", type=float, required=True)
    parser.add_argument("--start-lon", type=float, required=True)
    parser.add_argument("--start-alt", type=float, required=True)
    parser.add_argument("--end-lat", type=float, required=True)
    parser.add_argument("--end-lon", type=float, required=True)
    parser.add_argument("--end-alt", type=float, required=True)
    parser.add_argument("--duration", type=float, required=True)
    parser.add_argument("--step", type=float, default=0.1)
    parser.add_argument("--out", type=str, default="traj.csv")
    args = parser.parse_args()

    linear_trajectory(
        start_lat=args.start_lat,
        start_lon=args.start_lon,
        start_alt=args.start_alt,
        end_lat=args.end_lat,
        end_lon=args.end_lon,
        end_alt=args.end_alt,
        duration_s=args.duration,
        step_s=args.step,
        out_file=args.out
    )
