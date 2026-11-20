## 1. Rec

- [x] 1.1 Socket client for the rec widget and a detached widget start with `REC_INTERNAL=1`
- [x] 1.2 `r2d2 rec <command>` prints the widget reply, and closed-widget failures exit 1
- [x] 1.3 TUI with status, clock, corner or full screen, last error or saved path, 1 to 6, space, r, ?, and q

## 2. Menu and help

- [x] 2.1 The menu and `r2d2 help` list `rec`, and `screen.Launch` rejects it

## 3. Verification

- [x] 3.1 `openspec validate add-rec-screen --strict --no-interactive`
- [x] 3.2 `GOWORK=off go test ./...`
