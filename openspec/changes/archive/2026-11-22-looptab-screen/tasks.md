## 1. Screen

- [x] 1.1 Move the looptab job list, `looptab now` parsing, and terminal code into `internal/looptab`
- [x] 1.2 Add `e`, which opens the c3po jobs file in vim and asks c3po to install
- [x] 1.3 Dispatch `looptab` in r2d2 and update the menu line and the help
- [x] 1.4 Require the published c3po commit without a replace directive

## 2. Check

- [x] 2.1 Test the keys, `e`, missing vim, the help, and the install through c3po
- [x] 2.2 Run `GOWORK=off go test ./...`
- [x] 2.3 Validate with `openspec validate looptab-screen --strict --no-interactive` and archive it
