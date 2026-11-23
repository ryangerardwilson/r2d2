## Why

Skills for Claude Code, Grok Build, and Codex are written by hand in three places, and nothing keeps them alike. The public repository `ryangerardwilson/skills` is now the one source: a marketplace for each host with one plugin, `skills`, that holds every skill once. Editing it still means a clone, a text editor, and git by hand, and there is no screen for it in r2d2.

## What Changes

- New screen `r2d2 skills`, implemented in r2d2, that keeps a clone of `https://github.com/ryangerardwilson/skills` at `~/.local/share/r2d2/skills` and lists the skills under `plugins/skills/skills/`.
- On open, a missing clone is cloned. A clean clone on `main` with no unpushed commits is fetched and fast-forwarded. A dirty clone, unpushed commits, or another branch is left as it is and the screen says so. A directory that is not this repository is never deleted, and the screen shows an error with no writable list.
- `n` creates a skill from a name, Enter edits one in vim, `d` deletes one after a confirmation, and `r` fetches and fast-forwards. Each create, edit, and delete commits as Ryan Gerard Wilson and pushes `main`. A refused push stays on screen and keeps the local commit.
- The menu and `r2d2 help` list `skills`, and `skills` runs inside r2d2.

## Capabilities

### New Capabilities

- `skills-screen`: the skills screen, its clone, and its commits.

### Modified Capabilities

- `r2d2`: the screen list names `skills`, and `skills` runs inside r2d2.

## Impact

- Spec: `openspec/specs/r2d2/spec.md`, `openspec/specs/skills-screen/spec.md`.
- Code: `internal/skills/`, `internal/cli/`, `internal/screen/`.
- Docs: `AGENTS.md`, `README.md`, and the home `AGENTS.md` screen list.
- Non-goals: the screen does not shell out to `c3po`, does not force-push, reset, stash, or discard a change, does not install plugins into a host, and does not edit the marketplace manifests. c3po's screen map does not change.
