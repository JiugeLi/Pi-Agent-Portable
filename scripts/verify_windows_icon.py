"""Verify Pi's exact icon images and Windows' decoding of the built PE resources."""
import argparse
import ctypes
import json
from pathlib import Path
import struct

parser = argparse.ArgumentParser()
parser.add_argument('--release-dir', type=Path, required=True)
args = parser.parse_args()
root = Path(__file__).resolve().parents[1]
release = args.release_dir.resolve()
exe = release / 'pi-windows-x64.exe'
binary, icon = exe.read_bytes(), (root / 'assets/pi.ico').read_bytes()
reserved, kind, count = struct.unpack_from('<HHH', icon)
assert (reserved, kind, count) == (0, 1, 7)
sizes = []
for index in range(count):
    width, height, _, _, planes, bits, length, offset = struct.unpack_from('<BBBBHHII', icon, 6 + 16 * index)
    assert planes == 1 and bits == 32 and icon[offset:offset + length] in binary
    sizes.append(width or 256)

shell32 = ctypes.WinDLL('shell32', use_last_error=True)
user32 = ctypes.WinDLL('user32', use_last_error=True)
shell32.ExtractIconExW.argtypes = [ctypes.c_wchar_p, ctypes.c_int,
    ctypes.POINTER(ctypes.c_void_p), ctypes.POINTER(ctypes.c_void_p), ctypes.c_uint]
shell32.ExtractIconExW.restype = ctypes.c_uint
user32.DestroyIcon.argtypes = [ctypes.c_void_p]
large, small = ctypes.c_void_p(), ctypes.c_void_p()
groups = shell32.ExtractIconExW(str(exe), -1, None, None, 0)
assert groups == 1, 'Pi must be the default executable icon'
try:
    decoded = shell32.ExtractIconExW(str(exe), 0, ctypes.byref(large), ctypes.byref(small), 1)
    assert decoded not in (0, 0xffffffff) and large.value and small.value, 'Windows could not decode the Pi icons'
finally:
    for handle in (large, small):
        if handle.value:
            user32.DestroyIcon(handle)
report = {'platform': 'windows', 'defaultPiIconGroup': True, 'exactPiIconImagesEmbedded': sizes,
    'windowsSmallIconDecoded': True, 'windowsLargeIconDecoded': True}
(release / 'acceptance-icon-windows.json').write_text(json.dumps(report, indent=2) + '\n', encoding='utf-8')
print(json.dumps(report, indent=2))
