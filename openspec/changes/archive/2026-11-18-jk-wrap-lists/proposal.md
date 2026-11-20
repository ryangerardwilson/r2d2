## Why

`j` and `k` stop at the ends of the screen menu and the web lists. Moving past the last row should return to the first, and moving above the first should return to the last.

## What Changes

- On the screen menu, `j` and `k` wrap. `k` on `web` selects `slack`. `j` on `slack` selects `web`.
- On `r2d2 web`, `j` and `k` wrap the section menu, the essays, the landing fields, the pages, the API keys, and the schedule fields.
- An empty list stays put. While `?` is open, `j` and `k` still do nothing.

## Capabilities

### New Capabilities

### Modified Capabilities

- `r2d2`: `j` and `k` wrap the screen menu and each web list.

## Impact

- Spec: `openspec/specs/r2d2/spec.md`.
- Code: `internal/menu` and `internal/web`.
- The screens implemented in c3po and nebula are specified in those repositories.
- The spreadsheet stays clamped to the sheet. That screen is not in this module.
