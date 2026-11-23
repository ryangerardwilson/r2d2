## ADDED Requirements

### Requirement: r2d2 lists the skills screen

`skills` SHALL be a screen. It SHALL be listed in `screen.Screens`, the menu, and `r2d2 help`, together with the screens named in "r2d2 opens the screens". `skills` SHALL run inside r2d2 and SHALL NOT go through the c3po screen package. `screen.Own` SHALL report `skills`, and `screen.Launch` SHALL reject it. An unknown screen SHALL still exit 2.

#### Scenario: The menu lists skills

- **WHEN** the menu opens, or the operator runs `r2d2 help`
- **THEN** the screen list SHALL have a `skills` row

#### Scenario: skills runs inside r2d2

- **WHEN** the operator runs `r2d2 skills help`
- **THEN** r2d2 SHALL print the skills help without the c3po screen package
- **AND** the command SHALL exit 0
