## MODIFIED Requirements

### Requirement: r2d2 opens the screens

The module SHALL build one command, `r2d2`. On a terminal, `r2d2` with no arguments SHALL open a screen menu and SHALL NOT print the text help. `j` and `k` SHALL move the cursor and SHALL wrap. On the last screen, `j` SHALL move to the first screen. On the first screen, `k` SHALL move to the last screen. Enter SHALL open the selected screen. `q` SHALL quit the menu. The menu and `r2d2 web` SHALL NOT write shortcut text until the operator presses `?`. `?` SHALL toggle that list, and a second `?` SHALL hide it. While the list is open, `j` and `k` SHALL NOT move. When stdout is not a terminal, `r2d2` with no arguments SHALL print the screen list and SHALL exit 0. `r2d2 help` SHALL print that same list and SHALL exit 0. The screens SHALL be `web`, `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, `looptab`, `slack`, `health`, and `rec`. `r2d2 version` SHALL print a line that starts with `r2d2 `. An unknown screen SHALL exit 2 and SHALL name that screen. `web`, `looptab`, `health`, and `rec` SHALL run inside r2d2. Every other screen SHALL run inside r2d2 through the c3po screen package, with the remaining arguments, including `--resume`. `musica` SHALL NOT be a screen, and the menu SHALL NOT list it. `r2d2 musica` SHALL still run inside r2d2 and SHALL NOT exit as an unknown screen: with no further arguments it SHALL show the Tesseract Musica center through the launcher's `open.sh musica`, and with arguments it SHALL call the musica `player` package. r2d2 SHALL NOT exec `c3po tui`.

On `r2d2 web`, `j` and `k` SHALL wrap each list: the section menu, the essays, the landing fields, the pages, the API keys, and the schedule fields. On the last row, `j` SHALL move to the first row. On the first row, `k` SHALL move to the last row. An empty list SHALL leave the cursor in place. While the shortcut list is open, `j` and `k` SHALL NOT move.

#### Scenario: A terminal opens the menu

- **WHEN** the operator runs `r2d2` on a terminal
- **THEN** a screen menu SHALL open
- **AND** the menu SHALL list every screen
- **AND** `j` and `k` SHALL move
- **AND** Enter SHALL open the selected screen
- **AND** `q` SHALL leave the menu

#### Scenario: The menu hides shortcuts until question mark

- **WHEN** the menu opens
- **THEN** the view SHALL NOT contain shortcut text
- **AND** `?` SHALL show the shortcut list
- **AND** a second `?` SHALL hide that list

#### Scenario: The web screen hides shortcuts until question mark

- **WHEN** `r2d2 web` is showing a screen
- **THEN** the view SHALL NOT contain shortcut text
- **AND** `?` SHALL show the shortcuts for that screen
- **AND** a second `?` SHALL hide that list

#### Scenario: Help names the admin screen

- **WHEN** the operator runs `r2d2 help`, or runs `r2d2` when stdout is not a terminal
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name `r2d2 web`, `r2d2 slack`, `:Wrap`, `nebula crypt`, and a flowing paragraph

#### Scenario: An unknown screen is rejected

- **WHEN** the operator runs `r2d2 nlp`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL say the screen is unknown

#### Scenario: An older screen keeps its arguments

- **WHEN** the operator runs `r2d2 o help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name `r2d2 o`
- **AND** `r2d2 o --resume <id>` SHALL pass `--resume` and that id to the screen

#### Scenario: The menu wraps

- **WHEN** the menu cursor is on `web`
- **AND** the operator presses `k`
- **THEN** the cursor SHALL be on `rec`
- **AND** `j` SHALL move it back to `web`

#### Scenario: An admin list wraps

- **GIVEN** the essay list has two essays and the cursor is on the last
- **WHEN** the operator presses `j`
- **THEN** the cursor SHALL be on the first essay
- **AND** `k` SHALL move it back to the last essay

#### Scenario: The menu does not list musica

- **WHEN** the menu opens, or the operator runs `r2d2 help`
- **THEN** the screen list SHALL NOT have a `musica` row
- **AND** the help SHALL still name `r2d2 musica`

#### Scenario: musica runs without being a screen

- **GIVEN** `musica` is not a screen
- **WHEN** the operator runs `r2d2 musica help`
- **THEN** r2d2 SHALL call `player.Run` in process
- **AND** the command SHALL exit 0
- **AND** the text SHALL NOT say the screen is unknown
