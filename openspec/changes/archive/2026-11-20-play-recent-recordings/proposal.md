## Why

The rec screen can start and save a take, and the saved files sit in Videos with no way to play one from the screen. Super+R is not part of this screen.

## What Changes

- After the six actions, the screen lists up to 20 recent screen recordings, newest first.
- `j` and `k` move through the actions and the recordings and wrap.
- Enter on a recording plays it in its own window and does not send a camera command.
- A missing player shows its error in bold.
- `q` and `ctrl+c` leave the screen and leave both the camera and playback as they are.
- The screen does not delete a recording.

## Capabilities

### New Capabilities

### Modified Capabilities

- `rec-screen`: the control screen lists recent recordings and plays the selected one.

## Impact

- Code: `internal/rec`.
- Spec: `openspec/specs/rec-screen/spec.md`.
- Non-goals: no change to the camera program, the socket commands, or the Hyprland keymap. Playback does not stop when the screen closes.
