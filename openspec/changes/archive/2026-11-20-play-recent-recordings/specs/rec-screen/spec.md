## MODIFIED Requirements

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
