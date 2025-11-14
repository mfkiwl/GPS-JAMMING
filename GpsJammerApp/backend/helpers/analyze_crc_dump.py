#!/usr/bin/env python3
import argparse
from pathlib import Path
from typing import List, Tuple

try:
    from .crc24q import crc24_calc_from_dump
except ImportError:
    from crc24q import crc24_calc_from_dump


def parse_dump(path: Path) -> List[dict]:
    blocks = []
    current = {}
    section = None
    buffers = {
        "dec_even": [],
        "dec_odd": [],
        "fbits_raw": [],
        "fbits_dec": [],
        "polarized": [],
    }

    for raw_line in path.read_text().splitlines():
        line = raw_line.strip()
        if not line:
            continue

        if line.startswith("PRN="):
            if current.get("dec_even") and current.get("dec_odd"):
                blocks.append(current)
            current = {
                "meta": line,
                "dec_even": None,
                "dec_odd": None,
                "fbits_raw": None,
                "fbits_dec": None,
                "polarized_bits": None,
            }
            buffers = {key: [] for key in buffers}
            section = None
        elif line.startswith("DEC_EVEN"):
            section = "dec_even"
        elif line.startswith("DEC_ODD"):
            section = "dec_odd"
        elif line.startswith("FBITS_RAW"):
            section = "fbits_raw"
        elif line.startswith("FBITS_DEC"):
            section = "fbits_dec"
        elif line.startswith("POLARIZED_BITS"):
            section = "polarized"
        elif line.startswith("----"):
            for key, buf in buffers.items():
                if buf:
                    joined = " ".join(buf).strip()
                    if key == "dec_even":
                        current["dec_even"] = joined
                    elif key == "dec_odd":
                        current["dec_odd"] = joined
                    elif key == "fbits_raw":
                        current["fbits_raw"] = joined.replace(" ", "")
                    elif key == "fbits_dec":
                        current["fbits_dec"] = joined.replace(" ", "")
                    elif key == "polarized":
                        current["polarized_bits"] = joined.replace(" ", "")
            if current.get("dec_even") and current.get("dec_odd"):
                blocks.append(current)
            current = {}
            buffers = {key: [] for key in buffers}
            section = None
        elif section and line[:4].isdigit() and ":" in line:
            payload = line.split(":", 1)[1].strip()
            buffers[section].append(payload)

    for key, buf in buffers.items():
        if buf:
            joined = " ".join(buf).strip()
            if key == "dec_even":
                current["dec_even"] = joined
            elif key == "dec_odd":
                current["dec_odd"] = joined
            elif key == "fbits_raw":
                current["fbits_raw"] = joined.replace(" ", "")
            elif key == "fbits_dec":
                current["fbits_dec"] = joined.replace(" ", "")
            elif key == "polarized":
                current["polarized_bits"] = joined.replace(" ", "")
    if current.get("dec_even") and current.get("dec_odd"):
        blocks.append(current)

    return blocks


def invert_hex(hex_str: str) -> str:
    data = bytearray(bytes.fromhex(hex_str.replace(" ", "")))
    for i in range(len(data)):
        data[i] ^= 0xFF
    return " ".join(f"{b:02x}" for b in data)


def find_preamble_offset(bits: str, pattern: str = "1000101100") -> int:
    """Return index of best match (lowest Hamming distance) or -1 if absent."""
    if not bits:
        return -1
    best_idx = -1
    best_score = len(pattern) + 1
    for i in range(0, len(bits) - len(pattern) + 1):
        window = bits[i : i + len(pattern)]
        score = sum(ch1 != ch2 for ch1, ch2 in zip(window, pattern))
        if score < best_score:
            best_score = score
            best_idx = i
            if score == 0:
                break
    return best_idx


def summarise_block(block: dict) -> Tuple[int, int, bool, List[str], int]:
    even = block["dec_even"]
    odd = block["dec_odd"]
    crc_calc, crc_msg = crc24_calc_from_dump(even, odd)
    match = crc_calc == crc_msg
    hints = []
    if not match:
        inv_even = invert_hex(even)
        inv_odd = invert_hex(odd)
        if crc24_calc_from_dump(inv_even, odd)[0] == crc_msg:
            hints.append("invert-even fixes CRC")
        if crc24_calc_from_dump(even, inv_odd)[0] == crc_msg:
            hints.append("invert-odd fixes CRC")
        if crc24_calc_from_dump(inv_even, inv_odd)[0] == crc_msg:
            hints.append("invert-both fixes CRC")
    preamble_idx = find_preamble_offset(block.get("fbits_dec", ""))
    return crc_calc, crc_msg, match, hints, preamble_idx


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Analyse gal_crc_dump.txt for Galileo CRC mismatches"
    )
    parser.add_argument(
        "dump_path",
        nargs="?",
        default="../bin/gal_crc_dump.txt",
        help="Path to gal_crc_dump.txt (default: ../bin/gal_crc_dump.txt)",
    )
    args = parser.parse_args()
    path = Path(args.dump_path).expanduser().resolve()
    if not path.exists():
        raise SystemExit(f"Dump file not found: {path}")

    blocks = parse_dump(path)
    if not blocks:
        raise SystemExit("No dump entries found.")

    for idx, block in enumerate(blocks, 1):
        crc_calc, crc_msg, match, hints, pre_idx = summarise_block(block)
        meta = block["meta"]
        print(f"[{idx:03d}] {meta}")
        print(f"      CRC calc=0x{crc_calc:06X} crcmsg=0x{crc_msg:06X} match={match}")
        if pre_idx is not None and pre_idx >= 0:
            print(f"      fbits_dec preamble offset={pre_idx}")
        else:
            print("      fbits_dec preamble offset=not found")
        if hints:
            for hint in hints:
                print(f"      hint: {hint}")
        print()


if __name__ == "__main__":
    main()

