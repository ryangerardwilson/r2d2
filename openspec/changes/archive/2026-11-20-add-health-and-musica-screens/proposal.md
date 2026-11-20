## Why

Background jobs fail quietly. genesys.service has been failing, and nothing on screen says so. Backups land on nebula s3, and the only way to learn whether one is fresh is to read units and objects by hand. The music player is a separate binary with its own popup, outside the r2d2 screen menu.

## What Changes

- `r2d2 health` is a read-only screen with one row per background job, backup object, and nebula-hosted process. Each row is `ok`, `waiting`, or `down` with one short reason.
- `r2d2 musica` opens the musica player through the public `github.com/ryangerardwilson/musica/player` package. The player stays in the musica module.
- The menu and `r2d2 help` list `health` and `musica`.

## Capabilities

### New Capabilities

- `health`: the background job and backup health screen.
- `musica-screen`: the musica player as an r2d2 screen.

### Modified Capabilities

- `r2d2`: the screen list names `health` and `musica`.

## Impact

- Spec: `openspec/specs/r2d2/spec.md`, `openspec/specs/health/spec.md`, `openspec/specs/musica-screen/spec.md`.
- Code: `internal/health/`, `internal/cli/`, `internal/screen/`, `internal/menu/`.
- Modules: the published nebula (for `s3x.Stat`) and musica (for `player`). No replace directive.
- Non-goals: the health screen does not restart, stop, enable, or repair anything, and does not read object bodies. The player look and colors do not change. c3po's screen map does not change.
