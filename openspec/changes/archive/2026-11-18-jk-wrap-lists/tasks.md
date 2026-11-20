## 1. Lists

- [x] 1.1 The screen menu wraps `j` and `k`
- [x] 1.2 Each `r2d2 web` list wraps `j` and `k`, and help still blocks motion

## 2. Verification

- [x] 2.1 `openspec validate jk-wrap-lists --strict --no-interactive`
- [x] 2.2 `GOWORK=off go test` for the menu and the web package
