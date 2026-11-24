## MODIFIED Requirements

### Requirement: r2d2 opens the musica player

`r2d2 musica` SHALL be dispatched in the r2d2 process. With no further arguments, r2d2 SHALL run `<config>/quickshell/omarchy-launcher/open.sh musica` in its own session without waiting for it, `<config>` being `$XDG_CONFIG_HOME` when it is absolute and `~/.config` otherwise, which shows the Tesseract Musica center, the Super+T launcher showing Musica, as bare `musica` does. It SHALL NOT call `player.Run` for that and SHALL NOT open the terminal player, because the pinned musica module opens the terminal player with no arguments. When that script cannot start, r2d2 SHALL exit 1 and name it. `r2d2 musica end`, `play`, `open`, `config`, `help`, and `download-worker` SHALL pass their arguments unchanged to `player.Run` from `github.com/ryangerardwilson/musica/player`: `end` SHALL end the running musica session, and `help` SHALL print the player help. `musica` SHALL NOT be a screen in the r2d2 menu. r2d2 SHALL NOT reimplement the player and SHALL NOT copy its source. r2d2 SHALL require the published musica module, SHALL NOT use a replace directive, and SHALL NOT need a musica pin bump to open the center.

#### Scenario: The player opens

- **WHEN** the operator runs `r2d2 musica`
- **THEN** r2d2 SHALL run `~/.config/quickshell/omarchy-launcher/open.sh musica` in its own session
- **AND** it SHALL NOT call `player.Run` or open the terminal player
- **AND** the Tesseract Musica center SHALL show

#### Scenario: Help passes through

- **WHEN** the operator runs `r2d2 musica help`
- **THEN** r2d2 SHALL call `player.Run` with `help`
- **AND** the command SHALL exit 0
- **AND** the text SHALL be the musica player help

#### Scenario: End stays on the player

- **WHEN** the operator runs `r2d2 musica end`
- **THEN** r2d2 SHALL call `player.Run` with `end`
- **AND** the launcher script SHALL NOT run
