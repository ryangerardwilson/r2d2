## Why

`r2d2 skills` pushes each change to `main` on `ryangerardwilson/skills`, and then says `pushed`. Grok Build, Claude Code, and Codex each run their own installed copy of the `skills` plugin, and a push does not touch those copies. On 2026-11-24 `main` held `coding`, while Claude Code still ran commit `8f768ba`, and Grok's and Codex's installed `skills/` directories held only `.gitkeep`. An agent that looked for `coding` did not find it. The screen said the change was done when no host could load it.

## What Changes

- After a push of `main` succeeds, r2d2 refreshes the installed `skills` plugin for Grok Build, Claude Code, and Codex, in that order:
  - `grok plugin marketplace update skills`, then `grok plugin update skills`
  - `claude plugin marketplace update skills`, then `claude plugin update skills@skills -y`
  - `codex plugin marketplace upgrade skills`, then `codex plugin add skills@skills --json`
- After each host's commands, r2d2 reads where that host installed the plugin and compares its `skills/` tree with `plugins/skills/skills/` at the commit just pushed. A command that exits 0 and leaves the old files is a failure.
- The status says `pushed` only after all three hosts match. A failed host is named with its error, the pushed commit stays, and the screen does not claim the skill is ready.
- An empty commit and a failed push refresh nothing.
- The skills help explains the refresh.

## Capabilities

### Modified Capabilities

- `skills-screen`: a push installs the commit for each host before the screen says `pushed`.

## Impact

- Spec: `openspec/specs/skills-screen/spec.md`.
- Code: `internal/skills/`.
- Non-goals: no rollback of a commit or a push, no hand edits to a host's cache, no change to other plugins, marketplaces, or host settings, and no restart of a running host. A running Grok Build, Claude Code, or Codex still loads the new files only after a restart. The `?` shortcut list does not change because no key changes.
