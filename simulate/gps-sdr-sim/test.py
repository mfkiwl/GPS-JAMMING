import sys

def filter_rinex(input_file, output_file, target_prns):
    print(f"Filtrowanie {input_file} -> {output_file}")
    print(f"Wybrane satelity (PRN): {target_prns}")
    
    with open(input_file, 'r') as fin, open(output_file, 'w') as fout:
        # 1. Kopiuj nagłówek
        for line in fin:
            fout.write(line)
            if "END OF HEADER" in line:
                break
        
        # 2. Filtruj bloki danych
        keep_block = False
        block_line_count = 0
        
        for line in fin:
            # Sprawdzamy, czy to początek nowego bloku (linia z numerem PRN)
            # W RINEX 2.x PRN jest w znakach 0-2 (np. " 1" lub "15")
            # Ale uwaga: linie danych wewnątrz bloku są wcięte (zaczynają się od spacji)
            
            # Prosta heurystyka: Linia startowa ma format "PRN YY MM DD HH MM SS..."
            # PRN jest liczbą całkowitą na początku linii (ale nie wcięta za głęboko)
            
            is_new_block = False
            try:
                # Sprawdzamy czy to linia nagłówkowa bloku
                # Format RINEX: I2 (PRN)
                prn_str = line[0:2].strip()
                if prn_str.isdigit():
                    # To może być początek bloku. Sprawdźmy czy reszta to data.
                    # Omijamy linie, które są kontynuacją (one mają wcięcie > 5 spacji na początku)
                    if not line.startswith("     "): 
                        prn = int(prn_str)
                        if prn in target_prns:
                            keep_block = True
                        else:
                            keep_block = False
                        is_new_block = True
            except ValueError:
                pass

            if is_new_block:
                pass # Decyzja o keep_block podjęta wyżej
            
            if keep_block:
                fout.write(line)

    print("Gotowe.")

if __name__ == "__main__":
    # KONFIGURACJA
    ORIGINAL_FILE = "brdc2830.25n" # Twój oryginalny plik
    
    # Zestaw 1: Dwa satelity (PRN 5, 20)
    filter_rinex(ORIGINAL_FILE, "2_fake_safe.25n", [5, 20])
    
    # Zestaw 2: Trzy satelity (PRN 5, 20, 30)
    filter_rinex(ORIGINAL_FILE, "3_fake_safe.25n", [5, 20, 30])
    
    # Zestaw 3: Cztery satelity (PRN 5, 13, 20, 30)
    filter_rinex(ORIGINAL_FILE, "4_fake_safe.25n", [5, 13, 20, 30])