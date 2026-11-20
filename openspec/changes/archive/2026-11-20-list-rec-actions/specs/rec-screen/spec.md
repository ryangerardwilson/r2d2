## MODIFIED Requirements

### Requirement: The rec control screen shows the take

The control screen SHALL show the status word, the clock, whether the camera is in the corner or fills the screen, and the last error or the last saved path. It SHALL list start, pause, resume, stop and save with the camera staying open, fill the recording with you, and save and close. When the camera already fills the screen, that fill row SHALL say to put the camera back in the corner. `j` and `k` SHALL move through the list and SHALL wrap. Enter SHALL run the selected action. Number keys SHALL NOT run an action. The screen SHALL NOT use a shortcut page for these actions. It SHALL ask the widget for its status when it opens, about once a second after that, and on `r`. It SHALL NOT ask on other keys. It SHALL use black, white, and gray, with bold, dim, and reverse, and SHALL NOT use red, green, cyan, or yellow. `q` and `ctrl+c` SHALL leave the screen and SHALL leave the widget as it is. Save and close SHALL quit the widget and the screen, and SHALL save a running take before the widget closes, the same as `rec quit`. A failed command SHALL show its text. The screen SHALL NOT delete a recording and SHALL NOT edit Omarchy.

#### Scenario: The actions are a list

- **WHEN** the control screen opens
- **THEN** the view SHALL list start, pause, resume, stop and save, fill the recording with you, and save and close
- **AND** start SHALL be selected
- **AND** the view SHALL NOT contain a shortcut page

#### Scenario: j and k move through the actions

- **WHEN** the operator presses `j`
- **THEN** pause SHALL be selected
- **AND** `k` from start SHALL select save and close
- **AND** `j` from save and close SHALL select start

#### Scenario: Enter runs the selected action

- **GIVEN** pause is selected
- **WHEN** the operator presses Enter
- **THEN** the screen SHALL send pause
- **AND** `1` SHALL NOT send a command

#### Scenario: A stop shows the saved path

- **GIVEN** a take is recording and stop is selected
- **WHEN** the operator presses Enter
- **THEN** the screen SHALL show the saved mp4 path
- **AND** the widget SHALL stay open

#### Scenario: q leaves the camera running

- **GIVEN** a take is recording
- **WHEN** the operator presses `q`
- **THEN** the screen SHALL leave
- **AND** it SHALL NOT send quit
