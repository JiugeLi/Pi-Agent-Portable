"""Encode the original Pi pixel logo as a multi-resolution Windows icon (stdlib only)."""
from pathlib import Path
import struct
import zlib

ROOT = Path(__file__).resolve().parents[1]
COLORS = {'r': (228, 138, 122, 255), 'b': (79, 142, 179, 255), 'y': (234, 182, 93, 255), '.': (0, 0, 0, 0)}
PIXELS = ['rrr.', 'b.r.', 'bb.y', 'b..y']

def chunk(kind, data):
    return struct.pack('>I', len(data)) + kind + data + struct.pack('>I', zlib.crc32(kind + data))

def png(size):
    scale = max(1, size * 3 // 16)
    offset = (size - scale * 4) // 2
    rows = bytearray()
    for y in range(size):
        rows.append(0)
        for x in range(size):
            col, row = (x - offset) // scale, (y - offset) // scale
            rows.extend(COLORS[PIXELS[row][col] if 0 <= row < 4 and 0 <= col < 4 else '.'])
    return b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('>IIBBBBB', size, size, 8, 6, 0, 0, 0)) + chunk(b'IDAT', zlib.compress(rows, 9)) + chunk(b'IEND', b'')

def dib(size):
    # Keep bitmap frames for classic Shell consumers and reserve PNG compression
    # for the 256px frame. Include the transparent AND mask.
    scale = max(1, size * 3 // 16)
    offset = (size - scale * 4) // 2
    pixels, masks = bytearray(), bytearray()
    stride = ((size + 31) // 32) * 4
    for y in range(size - 1, -1, -1):
        mask = bytearray(stride)
        for x in range(size):
            col, row = (x - offset) // scale, (y - offset) // scale
            r, g, b, a = COLORS[PIXELS[row][col] if 0 <= row < 4 and 0 <= col < 4 else '.']
            pixels.extend((b, g, r, a))
            if not a:
                mask[x // 8] |= 1 << (7 - x % 8)
        masks.extend(mask)
    return struct.pack('<IIIHHIIIIII', 40, size, size * 2, 1, 32, 0, len(pixels), 0, 0, 0, 0) + pixels + masks

def generate():
    sizes = [16, 24, 32, 48, 64, 128, 256]
    images = [png(size) if size == 256 else dib(size) for size in sizes]
    offset = 6 + 16 * len(sizes)
    directory = bytearray(struct.pack('<HHH', 0, 1, len(sizes)))
    for size, data in zip(sizes, images):
        directory.extend(struct.pack('<BBBBHHII', size % 256, size % 256, 0, 0, 1, 32, len(data), offset))
        offset += len(data)
    target = ROOT / 'assets/pi.ico'
    target.write_bytes(directory + b''.join(images))
    print('Generated Pi icon:', target.name)

if __name__ == '__main__':
    generate()
