# rec-screen Specification

## Purpose
r2d2 rec controls the rec camera, a GTK window in the corner of the screen recording, and shows the take on a terminal. r2d2 never draws the camera.
## Requirements
### Requirement: r2d2 rec opens the camera and its control screen

`r2d2 rec` SHALL control the rec camera widget, a GTK window with app id `com.ryan.rec` that sits in the corner of the screen recording. r2d2 SHALL NOT draw the camera. When the widget is closed, r2d2 SHALL start it as the installed `rec` program with `REC_INTERNAL=1`, in its own session, so it outlives r2d2. When stdin and stdout are terminals, `r2d2 rec` SHALL start the widget if it is closed and SHALL open the control screen. When either is not a terminal, `r2d2 rec` SHALL start the widget if it is closed, SHALL raise it if it is open, SHALL NOT start the control screen, and SHALL exit 0. It SHALL exit 1 with a message if the widget does not answer within 10 seconds. r2d2 SHALL NOT open rec through a terminal popup. `r2d2 rec help` SHALL explain the screen and SHALL exit 0.

#### Scenario: Super+R opens the camera without a terminal

- **GIVEN** the widget is closed
- **WHEN** `r2d2 rec` runs without a terminal
- **THEN** r2d2 SHALL start the widget with `REC_INTERNAL=1`
- **AND** the control screen SHALL NOT start
- **AND** the command SHALL exit 0

#### Scenario: A terminal opens the control screen

- **WHEN** the operator runs `r2d2 rec` on a terminal
- **THEN** the widget SHALL be open
- **AND** the control screen SHALL open

### Requirement: r2d2 rec commands keep the rec replies

`r2d2 rec start`, `pause`, `resume`, `stop`, `focus`, `quit`, `status`, and `toggle` SHALL send that word to the widget over `$XDG_RUNTIME_DIR/rec.sock`, or `/tmp/rec-<uid>/rec.sock` without `XDG_RUNTIME_DIR`, and SHALL print the reply on stdout and exit 0. A reply such as `recording 0:01`, `paused 0:01`, `idle`, `busy`, a saved mp4 path, or `bye` SHALL be printed as it is. `start`, `toggle`, `stop`, and `quit` SHALL wait up to 600 seconds for the reply. The others SHALL wait 2 seconds. `start`, `toggle`, and `focus` SHALL start a closed widget with that command and SHALL print its status once the take has started or failed, or once focus is shown. `status`, `pause`, `resume`, and `stop` on a closed widget SHALL print `rec is not open` on stderr and SHALL exit 1. `quit` on a closed widget SHALL run `rec quit` with `REC_INTERNAL=1` and SHALL pass its output and exit code through, so a running recorder is still saved. `--size small|medium|large` SHALL be written to `~/.config/rec/size` and SHALL apply when the widget next opens. `--device` SHALL pass to a widget that r2d2 starts. Another word SHALL exit 2.

#### Scenario: A command prints the reply

- **GIVEN** the widget is open and recording
- **WHEN** the operator runs `r2d2 rec status`
- **THEN** stdout SHALL be the widget reply, such as `recording 0:01`
- **AND** the command SHALL exit 0

#### Scenario: A closed widget fails

- **GIVEN** the widget is closed
- **WHEN** the operator runs `r2d2 rec status`
- **THEN** stderr SHALL be `rec is not open`
- **AND** the command SHALL exit 1
- **AND** the widget SHALL NOT start

#### Scenario: Quit saves a take when the widget is gone

- **GIVEN** the widget is closed and the recorder is running
- **WHEN** the operator runs `r2d2 rec quit`
- **THEN** r2d2 SHALL run `rec quit` with `REC_INTERNAL=1`
- **AND** it SHALL print the saved path that command prints

### Requirement: The rec control screen shows the take

The control screen SHALL show the status word, the clock, whether the camera is in the corner or fills the screen, and the last error or the last saved path. It SHALL ask the widget for its status when it opens, about once a second after that, and on `r`. It SHALL NOT ask on other keys. It SHALL use black, white, and gray, with bold, dim, and reverse, and SHALL NOT use red, green, cyan, or yellow. It SHALL NOT write shortcut text until the operator presses `?`, and a second `?` SHALL hide it. While the shortcut list is open, keys other than `?`, `esc`, `q`, and `ctrl+c` SHALL do nothing. `1` SHALL start, `2` SHALL pause, `3` SHALL resume, `4` SHALL stop and save while the camera stays open, `5` SHALL toggle focus, and space SHALL toggle recording. `6` SHALL quit the widget and the screen. `q` SHALL quit the widget and the screen. `6` and `q` SHALL save a running take before the widget closes, the same as `rec quit`. `ctrl+c` SHALL leave the screen and SHALL leave the widget as it is. A failed command SHALL show its text. The screen SHALL NOT delete a recording and SHALL NOT edit Omarchy.

#### Scenario: Shortcuts stay hidden

- **WHEN** the control screen opens
- **THEN** the view SHALL NOT contain shortcut text
- **AND** `?` SHALL show the shortcuts
- **AND** `1` SHALL NOT start a take while they are shown
- **AND** a second `?` SHALL hide them

#### Scenario: A stop shows the saved path

- **GIVEN** a take is recording
- **WHEN** the operator presses `4`
- **THEN** the screen SHALL show the saved mp4 path
- **AND** the widget SHALL stay open

