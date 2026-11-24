## Why

Musica is becoming the third center of the Tesseract Super+T menu. The r2d2 screen menu still lists `musica` beside the terminal screens, so the player has two menus, and once r2d2 carries the center, Enter on that row would leave the terminal menu only to show a launcher surface. The r2d2 menu should list terminal screens only, and `r2d2 musica` should keep working from a shell.

## What Changes

- `musica` leaves `screen.Screens` and `screen.Own`. The menu, the pipe list, and the screen lines of `r2d2 help` no longer name it.
- `r2d2 musica` still runs in this process. It is dispatched before the screen check, so it does not exit as an unknown screen and is never sent to c3po.
- With no further arguments, `r2d2 musica` runs the launcher's `~/.config/quickshell/omarchy-launcher/open.sh musica` itself, in its own session, which shows the Tesseract Musica center. It does not call `player.Run` for that: the pinned musica module opens the terminal player with no arguments, which is how a terminal player was left holding the musica session. `end`, `play`, `open`, `config`, `help`, and `download-worker` pass through to `player.Run` unchanged.
- The `r2d2 musica` help line, `README.md`, and `AGENTS.md` say so.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `r2d2`: the screens drop `musica`; `r2d2 musica` still runs inside r2d2.
- `musica-screen`: bare `r2d2 musica` opens the Tesseract Musica center through the launcher's `open.sh musica`, other arguments go to `player.Run`, and musica is not a screen in the menu.

## Non-goals

- No musica pin bump in this change. r2d2 keeps `github.com/ryangerardwilson/musica v0.0.0-20261123015810-9f16b1ddb6c2`, and `go.mod` and `go.sum` do not change.
- No `replace` directive.
- r2d2 does not draw the center or copy player code. It starts the launcher's `open.sh musica` and does not wait for it.
- The `musica-screen` capability keeps its name.

## Impact

- Code: `internal/screen/screen.go`, `internal/cli/run.go`, `internal/cli/musica.go`
- Tests: `internal/menu/menu_test.go`, `internal/screen/screen_test.go`, `internal/cli/run_test.go`, `internal/cli/musica_test.go`
- Docs: `README.md`, `AGENTS.md`, `openspec/specs/r2d2/spec.md`, `openspec/specs/musica-screen/spec.md`
- Bare `r2d2 musica` reaches the center without a musica pin bump. Once the pin carries a `player.Run` that opens the center, the no-argument branch can call it again.
