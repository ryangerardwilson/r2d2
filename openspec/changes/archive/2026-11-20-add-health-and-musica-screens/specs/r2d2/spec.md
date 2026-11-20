## MODIFIED Requirements

### Requirement: r2d2 opens the screens

The module SHALL build one command, `r2d2`. On a terminal, `r2d2` with no arguments SHALL open a screen menu and SHALL NOT print the text help. `j` and `k` SHALL move the cursor without wrapping. Enter SHALL open the selected screen. `q` SHALL quit the menu. The menu and `r2d2 web` SHALL NOT write shortcut text until the operator presses `?`. `?` SHALL toggle that list, and a second `?` SHALL hide it. While the list is open, `j` and `k` SHALL NOT move. When stdout is not a terminal, `r2d2` with no arguments SHALL print the screen list and SHALL exit 0. `r2d2 help` SHALL print that same list and SHALL exit 0. The screens SHALL be `web`, `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, `looptab`, `slack`, `health`, and `musica`. `r2d2 version` SHALL print a line that starts with `r2d2 `. An unknown screen SHALL exit 2 and SHALL name that screen. `web`, `health`, and `musica` SHALL run inside r2d2. Every other screen SHALL run inside r2d2 through the c3po screen package, with the remaining arguments, including `--resume`. r2d2 SHALL NOT exec `c3po tui`.

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
