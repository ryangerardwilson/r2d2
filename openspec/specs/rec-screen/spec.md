# rec-screen Specification

## Purpose
r2d2 rec controls the rec camera, a GTK window in the corner of the screen recording, and shows the take on a terminal. r2d2 never draws the camera.
## Requirements
### Requirement: r2d2 rec opens the camera and its control screen

`r2d2 rec` SHALL control the rec camera widget, a GTK window with app id `com.ryan.rec` that sits in the corner of the screen recording. r2d2 SHALL NOT draw the camera. The widget program SHALL be `camera/rec.py` in this repository. `install.sh` SHALL copy that file to `~/.local/share/r2d2/rec.py`, and the installed `rec` launcher SHALL run that copy when `REC_INTERNAL=1`. When the widget is closed, r2d2 SHALL start it as the installed `rec` program with `REC_INTERNAL=1`, in its own session, so it outlives r2d2. When stdin and stdout are terminals, `r2d2 rec` SHALL start the widget if it is closed and SHALL open the control screen. When either is not a terminal, `r2d2 rec` SHALL start the widget if it is closed, SHALL raise it if it is open, SHALL NOT start the control screen, and SHALL exit 0. It SHALL exit 1 with a message if the widget does not answer within 10 seconds. r2d2 SHALL NOT open rec through a terminal popup. `r2d2 rec help` SHALL explain the screen and SHALL exit 0.

#### Scenario: A command without a terminal opens the camera

- **GIVEN** the widget is closed
- **WHEN** `r2d2 rec` runs without a terminal
- **THEN** r2d2 SHALL start the widget with `REC_INTERNAL=1`
- **AND** the control screen SHALL NOT start
- **AND** the command SHALL exit 0

#### Scenario: A terminal opens the control screen

- **WHEN** the operator runs `r2d2 rec` on a terminal
- **THEN** the widget SHALL be open
- **AND** the control screen SHALL open

#### Scenario: Install copies the camera program

- **WHEN** `install.sh` runs from this repository
- **THEN** `~/.local/share/r2d2/rec.py` SHALL be a copy of `camera/rec.py`
- **AND** `~/.local/bin/rec` with `REC_INTERNAL=1` SHALL run that copy

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

The control screen SHALL show the status word, the clock, whether the camera is in the corner or fills the screen, and the last error, the last saved path, or the recording that is playing. It SHALL list start, pause, resume, stop and save with the camera staying open, fill the recording with you, and save and close, then up to 20 recent screen recordings, newest first. A recording is an mp4 whose name starts with `screenrecording-` and does not contain `-processed`. The directory SHALL be `OMARCHY_SCREENRECORD_DIR`, otherwise `XDG_VIDEOS_DIR`, otherwise `XDG_VIDEOS_DIR` from `~/.config/user-dirs.dirs`, otherwise `~/Videos`. The list SHALL refresh when the screen opens, about once a second, on `r`, and after a save. An empty directory SHALL add no recording rows. When the camera already fills the screen, that fill row SHALL say to put the camera back in the corner. `j` and `k` SHALL move through the whole list and SHALL wrap. Enter SHALL run the selected action. Enter on a recording SHALL play that file and SHALL NOT send a camera command. A missing player SHALL show its error in bold. Number keys SHALL NOT run an action. The screen SHALL NOT use a shortcut page for these actions. It SHALL ask the widget for its status when it opens, about once a second after that, and on `r`. It SHALL NOT ask on other keys. It SHALL use black, white, and gray, with bold, dim, and reverse, and SHALL NOT use red, green, cyan, or yellow. `q` and `ctrl+c` SHALL leave the screen, SHALL leave the widget as it is, and SHALL NOT stop playback. Save and close SHALL quit the widget and the screen, and SHALL save a running take before the widget closes, the same as `rec quit`. A failed command SHALL show its text. The screen SHALL NOT delete a recording and SHALL NOT edit Omarchy.

#### Scenario: The actions are a list

- **WHEN** the control screen opens and Videos has no screen recordings
- **THEN** the view SHALL list start, pause, resume, stop and save, fill the recording with you, and save and close
- **AND** start SHALL be selected
- **AND** the view SHALL NOT list recordings
- **AND** the view SHALL NOT contain a shortcut page

#### Scenario: j and k move through the actions

- **GIVEN** Videos has no screen recordings
- **WHEN** the operator presses `j`
- **THEN** pause SHALL be selected
- **AND** `k` from start SHALL select save and close
- **AND** `j` from save and close SHALL select start

#### Scenario: j and k wrap across recordings

- **GIVEN** Videos has two screen recordings
- **WHEN** the operator presses `k` from start
- **THEN** the older recording SHALL be selected
- **AND** `j` from that row SHALL select start

#### Scenario: Enter runs the selected action

- **GIVEN** pause is selected
- **WHEN** the operator presses Enter
- **THEN** the screen SHALL send pause
- **AND** `1` SHALL NOT send a command

#### Scenario: Enter plays a recording

- **GIVEN** two recordings are listed and the newer one is selected
- **WHEN** the operator presses Enter
- **THEN** the screen SHALL play the newer file
- **AND** it SHALL NOT send a camera command

#### Scenario: A missing player shows the error

- **GIVEN** a recording is selected and the player is missing
- **WHEN** the operator presses Enter
- **THEN** the screen SHALL show the error in bold
- **AND** it SHALL NOT send a camera command

#### Scenario: A stop shows the saved path

- **GIVEN** a take is recording and stop is selected
- **WHEN** the operator presses Enter
- **THEN** the screen SHALL show the saved mp4 path
- **AND** the widget SHALL stay open

#### Scenario: q leaves the camera running

- **GIVEN** a take is recording and a recording is playing
- **WHEN** the operator presses `q`
- **THEN** the screen SHALL leave
- **AND** it SHALL NOT send quit
- **AND** it SHALL NOT stop playback

