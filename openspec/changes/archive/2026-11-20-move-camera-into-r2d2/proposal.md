## Why

The camera program lived in the home repo at `Apps/rec`. r2d2 is the source of truth for rec, and a second copy under Apps is the wrong place.

## What Changes

- The camera program is `camera/rec.py` in this repository.
- `install.sh` copies it to `~/.local/share/r2d2/rec.py` and writes `~/.local/bin/rec`.
- `REC_INTERNAL=1` runs that installed copy. r2d2 still does not draw the camera.

## Capabilities

### New Capabilities

### Modified Capabilities

- `rec-screen`: the camera program ships with r2d2.

## Impact

- Code: `camera/`, `install.sh`.
- Spec: `openspec/specs/rec-screen/spec.md`.
- Non-goals: no change to the screen, the socket commands, or playback.
