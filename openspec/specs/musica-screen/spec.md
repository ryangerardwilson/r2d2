# musica-screen Specification

## Purpose
r2d2 musica opens the musica player, which stays in the musica module, as an r2d2 screen.
## Requirements
### Requirement: r2d2 opens the musica player

`r2d2 musica` SHALL open the musica player in the r2d2 process by calling `player.Run` from `github.com/ryangerardwilson/musica/player` with the remaining arguments. r2d2 SHALL NOT reimplement the player and SHALL NOT copy its source. `r2d2 musica end` SHALL end the running musica session. `r2d2 musica help` SHALL print the player help. Other arguments, such as `play`, `open`, `config`, and `download-worker`, SHALL pass through. r2d2 SHALL require the published musica module and SHALL NOT use a replace directive.

#### Scenario: The player opens

- **WHEN** the operator runs `r2d2 musica`
- **THEN** the musica player SHALL open with the saved library

#### Scenario: Help passes through

- **WHEN** the operator runs `r2d2 musica help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL be the musica player help

