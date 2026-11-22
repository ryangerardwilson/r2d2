## MODIFIED Requirements

### Requirement: r2d2 rec commands keep the rec replies

`r2d2 rec start`, `pause`, `resume`, `stop`, `focus`, `quit`, `status`, and `toggle` SHALL send that word to the widget over `$XDG_RUNTIME_DIR/rec.sock`, or `/tmp/rec-<uid>/rec.sock` without `XDG_RUNTIME_DIR`, and SHALL print the reply on stdout and exit 0. A reply such as `recording 0:01`, `paused 0:01`, `idle`, `busy`, a saved mp4 path, or `bye` SHALL be printed as it is. `start`, `toggle`, `stop`, and `quit` SHALL wait up to 600 seconds for the reply. The others SHALL wait 2 seconds. `start`, `toggle`, and `focus` SHALL start a closed widget with that command and SHALL print its status once the take has started or failed, or once focus is shown. `status`, `pause`, `resume`, and `stop` on a closed widget SHALL print `rec is not open` on stderr and SHALL exit 1. `quit` on a closed widget SHALL run `rec quit` with `REC_INTERNAL=1` and SHALL pass its output and exit code through, so a running recorder is still saved. `--size small|medium|large` SHALL be written to `<config>/r2d2/rec/size` and SHALL apply when the widget next opens. `<config>` SHALL be `$XDG_CONFIG_HOME` when it is an absolute path, otherwise `~/.config`. r2d2 SHALL write the size only to that file, and the widget SHALL read only that file. When that file is absent and `<config>/rec/size` exists, an `r2d2 rec` command SHALL copy the old file there once before it writes a size or talks to the widget, and SHALL leave the old file in place. `--device` SHALL pass to a widget that r2d2 starts. Another word SHALL exit 2.

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

#### Scenario: The size lives under r2d2

- **GIVEN** `XDG_CONFIG_HOME` is unset
- **WHEN** the operator runs `r2d2 rec --size medium status`
- **THEN** `~/.config/r2d2/rec/size` SHALL be `medium`
- **AND** `~/.config/rec/size` SHALL NOT be written
- **AND** the widget SHALL open at that size

#### Scenario: The old size is copied once

- **GIVEN** `~/.config/rec/size` is `large` and `~/.config/r2d2/rec/size` is absent
- **WHEN** the operator runs `r2d2 rec status`
- **THEN** `~/.config/r2d2/rec/size` SHALL be `large`
- **AND** `~/.config/rec/size` SHALL stay in place
- **AND** a later change to `~/.config/rec/size` SHALL NOT be copied again
