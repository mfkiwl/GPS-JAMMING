CRC24_POLY = 0x1864CFB


def crc24_table(poly=CRC24_POLY):
    table = []
    for i in range(256):
        crc = i << 16
        for _ in range(8):
            crc <<= 1
            if crc & 0x1000000:
                crc ^= poly
        table.append(crc & 0xFFFFFF)
    return table


CRC24_TABLE = crc24_table()


def crc24_from_bytes(data):
    crc = 0
    for byte in data:
        idx = ((crc >> 16) ^ byte) & 0xFF
        crc = ((crc << 8) & 0xFFFFFF) ^ CRC24_TABLE[idx]
    return crc & 0xFFFFFF


def extract_crcbits(dec_even, dec_odd):
    bits = [0] * 196

    for i in range(15):
        for j in range(8):
            idx = 8 * i + j
            if idx == 114:
                break
            value = ((dec_even[i] << j) & 0x80) >> 7
            bits[idx] = -2 * value + 1
        if idx >= 113:
            break

    for i in range(11):
        for j in range(8):
            idx = 114 + 8 * i + j
            if idx == 196:
                break
            value = ((dec_odd[i] << j) & 0x80) >> 7
            bits[idx] = -2 * value + 1
        if idx >= 195:
            break
    return bits


def bits_to_bytes_right(bits, nbin=25):

    total_bits = nbin * 8
    rem = total_bits - len(bits)
    buf = [0] * total_bits
    buf[rem:rem + len(bits)] = bits[:]
    out = []
    for i in range(nbin):
        b = 0
        for j in range(8):
            b <<= 1
            if buf[i * 8 + j] < 0:
                b |= 1
        out.append(b)
    return bytes(out)


def crc24_calc_from_dump(dec_even_hex, dec_odd_hex):
    even = bytes.fromhex(dec_even_hex.replace(" ", ""))
    odd = bytes.fromhex(dec_odd_hex.replace(" ", ""))
    crcbits = extract_crcbits(even, odd)
    crc_bytes = bits_to_bytes_right(crcbits)
    crc_calc = crc24_from_bytes(crc_bytes)


    crc_msg = 0
    for idx in range(82, 82 + 24):
        byte = odd[idx // 8]
        bit = (byte >> (7 - (idx % 8))) & 0x01
        crc_msg = (crc_msg << 1) | bit
    return crc_calc, crc_msg


if __name__ == "__main__":
    dec_even = "03 40 3b 8c b9 c0 b5 59 6f d7 65 1c f1 62 40"
    dec_odd = "6c 4d 4e 62 8b ee 90 2a aa aa 06 6c cc d8 00"
    crc_calc, crc_msg = crc24_calc_from_dump(dec_even, dec_odd)
    print("DEC_EVEN:", dec_even)
    print("DEC_ODD :", dec_odd)
    print(f"CRC calc : 0x{crc_calc:06X}")
    print(f"CRC msg  : 0x{crc_msg:06X}")