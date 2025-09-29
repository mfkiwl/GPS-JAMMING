import numpy as np
import os

def read_iq_data(filename):
    """
    Reads IQ data from a binary file.
    This version is specifically for uint8 formatted data.
    """
    try:
        # Read the raw bytes from the file as unsigned 8-bit integers.
        with open(filename, 'rb') as f:
            raw_data = np.fromfile(f, dtype=np.uint8)

        if raw_data.size == 0:
            print(f"Warning: The file '{filename}' is empty.")
            return None

        # Convert uint8 data [0, 255] to a normalized float format [-1.0, 1.0]
        float_data = (raw_data.astype(np.float32) - 127.5) / 127.5

        # De-interleave the data into I and Q components
        i_samples = float_data[0::2]
        q_samples = float_data[1::2]

        # Combine into complex numbers (I + jQ)
        complex_data = i_samples + 1j * q_samples
        return complex_data
    except FileNotFoundError:
        print(f"Error: The file '{filename}' was not found.")
        print("Please make sure your data file is in the same directory as the script.")
        return None

def find_change_point(amplitude_data, threshold):
    """
    Finds the first index where the amplitude exceeds a given threshold.
    """
    change_indices = np.where(amplitude_data > threshold)[0]
    if len(change_indices) > 0:
        return change_indices[0]
    return None

def estimate_distance(received_power_dbm, tx_power_dbm, frequency_mhz, path_loss_exponent):
    """
    Estimates distance using the Log-Distance Path Loss Model.
    """
    # Free space path loss constant for the first meter
    path_loss_at_1m = 20 * np.log10(frequency_mhz) + 20 * np.log10(1) - 27.55

    # Formula to find distance based on path loss
    distance = 10 ** ((tx_power_dbm - received_power_dbm - path_loss_at_1m) / (10 * path_loss_exponent))
    return distance

def main():
    # --- 1. Setup ---
    iq_filename = "AfterMinute.bin"
    print(f"--- Attempting to load IQ data from '{iq_filename}' (assuming uint8 format) ---")

    # --- 2. Read and Prepare Data ---
    iq_samples = read_iq_data(iq_filename)
    if iq_samples is None:
        return

    print(f"Successfully loaded and processed {len(iq_samples)} IQ samples.")

    # --- 3. Signal Change Detection ---
    amplitude = np.abs(iq_samples)

    # Define a threshold for detecting the signal.
    # You will need to carefully tune this value. It should be just above your noise floor.
    signal_threshold = 0.1 # This is a starting guess, adjust as needed.

    turn_on_index = find_change_point(amplitude, signal_threshold)

    if turn_on_index is not None:
        print(f"\nSignal change detected at sample index: {turn_on_index}")

        # --- 4. Distance Estimation ---
        avg_amplitude = np.mean(amplitude[turn_on_index:])
        print(f"Average signal amplitude after turn-on: {avg_amplitude:.4f}")

        # This calculation is a rough, uncalibrated estimate based on the normalized amplitude.
        # The accuracy of the final distance calculation depends heavily on this assumption.
        received_signal_power_dbm = 10 * np.log10(avg_amplitude**2)
        print(f"Estimated received signal power: {received_signal_power_dbm:.2f} dBm (Hypothetical)")

        # --- Parameters for the Path Loss Model (Adjust as needed) ---
        
        # Power of the hypothetical transmitter (in dBm).
        transmit_power_dbm = 20.0 # e.g., 20 dBm = 100 mW, a common value for lab equipment.
        
        # L1 GPS frequency in MHz.
        signal_frequency_mhz = 1575.42
        
        # Environmental factor (2.0 for free space, 2.5-4 for indoor/obstructed paths).
        path_loss_exponent = 2.5

        print("\n--- Estimating Distance with Path Loss Model ---")
        print(f"Using the following parameters:")
        print(f"  - Transmitter Power: {transmit_power_dbm} dBm")
        print(f"  - Signal Frequency: {signal_frequency_mhz} MHz")
        print(f"  - Path Loss Exponent: {path_loss_exponent}")

        # Estimate the distance in meters
        calculated_distance_meters = estimate_distance(
            received_signal_power_dbm,
            transmit_power_dbm,
            signal_frequency_mhz,
            path_loss_exponent
        )

        print(f"\n>>> Estimated Distance to Antenna: {calculated_distance_meters:.2f} meters")

    else:
        print(f"\nSignal did not cross the threshold of {signal_threshold}. No change detected.")
        if len(amplitude) > 0:
            print(f"Max amplitude detected in file was: {np.max(amplitude):.4f}")
            print("You may need to adjust the 'signal_threshold' value.")


if __name__ == "__main__":
    main()