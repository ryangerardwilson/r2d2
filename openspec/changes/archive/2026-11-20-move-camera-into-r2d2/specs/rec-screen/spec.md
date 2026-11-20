## MODIFIED Requirements

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
