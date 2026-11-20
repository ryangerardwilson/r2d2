## Why

rec, the screen recorder with a camera circle in the corner, is a separate Python command. Its only controls are Super+1 through Super+6 and a status line on demand. There is no screen that shows whether a take is running, how long it has run, or where the last one was saved, and rec is not in the r2d2 menu.

## What Changes

- `r2d2 rec` on a terminal opens a control screen for the rec camera widget and starts the widget when it is closed. Without a terminal it starts or raises the widget and exits 0, so Super+R still opens the camera.
- `r2d2 rec start|pause|resume|stop|focus|quit|status|toggle` sends that command to the widget over the rec socket and prints the reply.
- The camera stays a GTK window in the corner of the recording. r2d2 starts it as the rec Python program with `REC_INTERNAL=1`. r2d2 does not draw the camera.
- The menu and `r2d2 help` list `rec`.

## Capabilities

### New Capabilities

- `rec-screen`: the rec control screen and commands.

### Modified Capabilities

- `r2d2`: the screen list names `rec`, and `rec` runs inside r2d2.

## Impact

- Spec: `openspec/specs/r2d2/spec.md`, `openspec/specs/rec-screen/spec.md`.
- Code: `internal/rec/`, `internal/cli/`, `internal/screen/`, `internal/menu/`.
- Non-goals: r2d2 does not draw the camera, run the recorder, delete a recording, or edit Omarchy. c3po's screen map does not change. The rec socket protocol and replies do not change.
