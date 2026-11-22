## 1. Path

- [x] 1.1 Stop naming the scheduler file path in the r2d2 looptab requirement
- [x] 1.2 Name the c3po looptab directory in `AGENTS.md`
- [x] 1.3 Clear `XDG_CONFIG_HOME` in the edit test

## 2. Check

- [x] 2.1 Run `GOWORK=off go test ./internal/looptab/`
- [x] 2.2 Validate with `openspec validate looptab-scheduler-path-in-c3po --strict --no-interactive` and archive it
