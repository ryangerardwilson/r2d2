## 1. Health

- [x] 1.1 Catalog of units, timers, objects, and nebula run rows with their real intervals
- [x] 1.2 Read-only checks with a 5 second timeout for each s3 and ssh call
- [x] 1.3 TUI with j/k wrap, g/G, r reload, ? shortcuts, q quit, and rows on a pipe

## 2. Musica

- [x] 2.1 `r2d2 musica` runs `player.Run` with the remaining arguments

## 3. Menu and help

- [x] 3.1 The menu and `r2d2 help` list `health` and `musica`

## 4. Verification

- [x] 4.1 `openspec validate add-health-and-musica-screens --strict --no-interactive`
- [x] 4.2 `GOWORK=off go test ./...`
