## Why

c3po is the source of truth for each operation and r2d2 is the terminal program above it. The `r2d2 looptab` screen lived in c3po, and the vim session that edits the jobs file was `c3po looptab` itself, so the CLI opened a terminal UI. The screen moves here and calls c3po for the jobs file and the install.

## What Changes

- `r2d2 looptab` is implemented in `internal/looptab` and is no longer dispatched through the c3po screen package.
- `j`, `k`, Enter, `q`, and `?` behave as before. `e` asks c3po for the jobs file, opens it in vim, asks c3po to install, and reloads the list.
- Missing vim says `vim is not on PATH` and leaves the screen up.
- The menu line and the help name both running a job and editing the jobs file.
- r2d2 requires the c3po commit that exports `looptab.JobsFile`, `looptab.Install`, and the screen session helpers.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `r2d2`: looptab runs inside r2d2 and edits the jobs file through c3po.

## Impact

- Code: `internal/looptab/`, `internal/cli/run.go`, `internal/screen/screen.go`, `go.mod`, `go.sum`
- Tests: `internal/looptab/`, `internal/cli/run_test.go`, `internal/screen/screen_test.go`
- Not changed: the catalog in the sqlx database `c3po`, `~/.config/c3po/looptab/jobs`, and `~/.config/looptab/looptab`, which c3po still writes.
