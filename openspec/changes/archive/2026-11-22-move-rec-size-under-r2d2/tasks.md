## 1. Spec

- [x] 1.1 Validate this change strict.

## 2. Size file

- [x] 2.1 Write `--size` to `<config>/r2d2/rec/size`.
- [x] 2.2 Copy `<config>/rec/size` once when the new file is absent.
- [x] 2.3 Read the new path in `camera/rec.py`.
- [x] 2.4 Run `GOWORK=off go test ./...` and `python3 camera/test_rec.py`.
