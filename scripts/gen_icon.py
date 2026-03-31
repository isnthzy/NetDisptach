#!/usr/bin/env python3
"""Generate tray icon bytes for Go code"""

import struct

def create_ico():
    size = 16

    # Create pixel data (BGRA, bottom-up for BMP)
    pixels = []
    for y in range(size):
        row = []
        for x in range(size):
            dx = x - size // 2 + 0.5
            dy = y - size // 2 + 0.5
            dist = (dx * dx + dy * dy) ** 0.5

            if dist <= 6:
                row.extend([0, 0, 0, 255])  # Black, opaque
            else:
                row.extend([0, 0, 0, 0])  # Transparent
        pixels.append(row)

    # BMP stores rows bottom-up
    pixels = pixels[::-1]
    pixel_data = bytes([b for row in pixels for b in row])

    # AND mask
    mask_rows = []
    for y in range(size):
        mask_byte = 0
        for x in range(size):
            dx = x - size // 2 + 0.5
            dy = y - size // 2 + 0.5
            dist = (dx * dx + dy * dy) ** 0.5
            if dist > 6:
                mask_byte |= (1 << (7 - x))
        mask_rows.append(mask_byte)

    mask_rows = mask_rows[::-1]
    # Pad mask to multiple of 4 bytes per row (2 bytes for 16 pixels)
    and_mask = bytes(mask_rows) + bytes(16)  # 16 rows + padding

    # BMP header
    bmp_header = struct.pack('<I', 40)  # Header size
    bmp_header += struct.pack('<i', size)  # Width
    bmp_header += struct.pack('<i', size * 2)  # Height
    bmp_header += struct.pack('<H', 1)  # Planes
    bmp_header += struct.pack('<H', 32)  # Bits per pixel
    bmp_header += struct.pack('<I', 0)  # Compression
    bmp_header += struct.pack('<I', len(pixel_data) + len(and_mask))
    bmp_header += struct.pack('<i', 0)  # X ppm
    bmp_header += struct.pack('<i', 0)  # Y ppm
    bmp_header += struct.pack('<I', 0)  # Colors used
    bmp_header += struct.pack('<I', 0)  # Important colors

    data_offset = 6 + 16 + 40
    image_size = 40 + len(pixel_data) + len(and_mask)

    # ICONDIR
    icondir = struct.pack('<HHH', 0, 1, 1)

    # ICONDIRENTRY
    icondirentry = struct.pack('<BBBBHHII',
        size, size, 0, 0, 1, 32, image_size, data_offset)

    ico_data = icondir + icondirentry + bmp_header + pixel_data + and_mask
    return ico_data

if __name__ == '__main__':
    ico_data = create_ico()

    # Output as Go byte slice
    print("// Auto-generated tray icon (16x16 black circle)")
    print("return []byte{")
    for i in range(0, len(ico_data), 16):
        chunk = ico_data[i:i+16]
        hex_vals = ', '.join(f'0x{b:02x}' for b in chunk)
        print(f"\t{hex_vals},")
    print("}")
