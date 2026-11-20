## 1. Camera

- [x] 1.1 Keep the camera program at `camera/rec.py`
- [x] 1.2 `install.sh` copies it to `~/.local/share/r2d2/rec.py` and writes the launcher

## 2. Check

- [x] 2.1 `openspec validate move-camera-into-r2d2 --strict --no-interactive`
- [x] 2.2 `python3 -m unittest discover -s camera -p test_rec.py`
