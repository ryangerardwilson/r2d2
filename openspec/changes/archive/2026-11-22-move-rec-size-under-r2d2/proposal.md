## Why

rec was a separate program before it became the r2d2 rec screen, and it kept `~/.config/rec/size`. r2d2 keeps each feature's config under `~/.config/r2d2/<feature>/`.

## What Changes

- `r2d2 rec --size` writes `<config>/r2d2/rec/size`, where `<config>` is an absolute `XDG_CONFIG_HOME` or `~/.config`.
- The camera widget reads only that file.
- When that file is absent, `r2d2 rec` copies `<config>/rec/size` once and leaves it in place.
- The socket, the recordings directory, `user-dirs.dirs`, and `~/.local/share/r2d2/rec.py` stay where they are.

## Capabilities

### Modified Capabilities

- `rec-screen`: the size file moves under `r2d2/rec`.

## Impact

- `internal/rec` size path and one-time copy.
- `camera/rec.py` size path.
