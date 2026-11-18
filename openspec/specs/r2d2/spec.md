# r2d2

## Purpose

r2d2 is the terminal program. Running it opens a screen menu. c3po stays the agent CLI.

## Requirements

### Requirement: r2d2 opens the screens

The module SHALL build one command, `r2d2`. On a terminal, `r2d2` with no arguments SHALL open a screen menu and SHALL NOT print the text help. `j` and `k` SHALL move the cursor without wrapping. Enter SHALL open the selected screen. `q` SHALL quit the menu. The menu and `r2d2 web` SHALL NOT write shortcut text until the operator presses `?`. `?` SHALL toggle that list, and a second `?` SHALL hide it. While the list is open, `j` and `k` SHALL NOT move. When stdout is not a terminal, `r2d2` with no arguments SHALL print the screen list and SHALL exit 0. `r2d2 help` SHALL print that same list and SHALL exit 0. The screens SHALL be `web`, `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, `looptab`, and `slack`. `r2d2 version` SHALL print a line that starts with `r2d2 `. An unknown screen SHALL exit 2 and SHALL name that screen. `web` SHALL run inside r2d2. Every other screen SHALL run inside r2d2 through the c3po screen package, with the remaining arguments, including `--resume`. r2d2 SHALL NOT exec `c3po tui`.

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

### Requirement: The admin screen uses the site API

`r2d2 web` SHALL call the ryangerardwilson.com admin API. The default origin SHALL be `https://ryangerardwilson.com`. `R2D2_WEB_ORIGIN` SHALL override it. The admin API key SHALL be read with nebula `crypt` from vault Root, item `RGW_API_KEY`, unless `R2D2_WEB_TOKEN` is set. The key SHALL NOT be printed, including in an error. A non-terminal SHALL print the web help and SHALL exit 0. `r2d2 web help` SHALL exit 0 and SHALL name `:Wrap` and `RGW_API_KEY`. Extra arguments SHALL exit 2.

The screen SHALL list, create, edit, and delete essays, edit the landing page, list, create, edit, and delete pages, and list, create, and revoke API keys. It SHALL move a future essay up or down in the queue and SHALL edit the weekly publish slot. It SHALL set the portrait and the cover from a file path. It SHALL refuse to delete the pages `about`, `privacy`, and `fulfillment`.

#### Scenario: A missing proof action still saves

- **GIVEN** the proof route answers 404 or 405
- **WHEN** an essay is saved
- **THEN** the original title and body SHALL be saved

#### Scenario: A failed proof does not save

- **GIVEN** the proof route answers with another error
- **WHEN** an essay is saved
- **THEN** the essay SHALL NOT be written

### Requirement: Vim wrap is the editing view

An essay, a page, and a long landing field SHALL open in `vim -c Wrap`. The saved essay or page SHALL join wrapped lines inside a paragraph with single spaces. A blank line SHALL stay a paragraph break. The first paragraph SHALL be the title. A long landing field SHALL be stored as one paragraph.

#### Scenario: Wrapped lines become one paragraph

- **GIVEN** a vim buffer whose title lines are `A quiet` and `ending`, then a blank line, then body lines
- **WHEN** the buffer is read back
- **THEN** the title SHALL be `A quiet ending`
- **AND** the body SHALL keep a blank line between its paragraphs
