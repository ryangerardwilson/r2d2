## 1. Install after a push

- [x] 1.1 After a successful push, run the Grok Build, Claude Code, and Codex refresh commands with stdin closed and their output captured
- [x] 1.2 Read each host's installed plugin directory and compare its `skills/` tree with `plugins/skills/skills/` at HEAD
- [x] 1.3 Say `pushed` only when all three match; otherwise name each failed host and its error and keep the commit
- [x] 1.4 Refresh nothing after an empty commit or a failed push
- [x] 1.5 Explain the refresh in `r2d2 skills help`

## 2. Verification

- [x] 2.1 `openspec validate install-skills-after-push --strict --no-interactive`
- [x] 2.2 `GOWORK=off go test ./...` with injected host commands that call no real `grok`, `claude`, or `codex`
