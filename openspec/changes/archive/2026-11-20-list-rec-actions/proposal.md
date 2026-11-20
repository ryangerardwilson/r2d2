## Why

The rec screen hides start, pause, resume, stop, you, and quit behind number keys. Those actions should be rows the operator can see and move through.

## What Changes

- The control screen lists start, pause, resume, stop and save, fill the recording with you, and save and close.
- `j` and `k` move through that list and wrap. Enter runs the selected row.
- `q` and `ctrl+c` leave the screen and leave the camera as it is.
- Number keys do not run an action.

## Capabilities

### New Capabilities

### Modified Capabilities

- `rec-screen`: the actions are a list on the control screen.

## Impact

- Code: `internal/rec`.
- Spec: `openspec/specs/rec-screen/spec.md`.
- Non-goals: no change to the camera program, the socket commands, or the Hyprland keymap.
