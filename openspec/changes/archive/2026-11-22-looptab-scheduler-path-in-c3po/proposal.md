## Why

c3po moved the Looptab scheduler file to `~/.config/c3po/looptab/looptab`. The r2d2 spec and AGENTS.md still name `~/.config/looptab/looptab` as the file c3po writes. r2d2 calls `looptab.Install` and should not keep its own copy of that path.

## What Changes

- The looptab requirement says c3po installs into the Looptab scheduler file it owns and that the screen neither writes that file nor names its path.
- AGENTS.md names the new c3po directory.
- The edit test clears `XDG_CONFIG_HOME`, so a later c3po that honors it still writes into the test home and not the live config.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `r2d2`: the looptab screen does not name the scheduler file path.

## Impact

- Docs: `AGENTS.md`, `openspec/specs/r2d2/spec.md`
- Tests: `internal/looptab/run_test.go`
- Not changed: the c3po module version, the screen code, and its behavior.
