## ADDED Requirements

### Requirement: The health screen shows background jobs

`r2d2 health` SHALL show one row per check. A row SHALL have a name, a state word, and one short reason. The state SHALL be `ok`, `waiting`, or `down`. `waiting` SHALL count as healthy. The title SHALL show the count of `down` rows. The screen SHALL check once when it opens and again on `r`. It SHALL NOT check again on other keys. A failed check SHALL be `down` with the error text. Each s3 and ssh check SHALL give up after 5 seconds and SHALL be `down` with a timeout reason. The screen SHALL NOT restart, stop, enable, or repair a unit, and SHALL NOT read an object body.

The checks SHALL be:

- `tunnel.service`, `looptab.service`, and `antibloat-observer.service`: `ok` when active and running, else `down`.
- `gmail.timer` (1 hour), `gdrive.timer` (1 hour), `c3po-snapshot.timer` (10 minutes), `sqlx-backup.timer` (daily), `tesseract-snapshot.timer` (15 minutes), `rgw-slack-quickshell-plugin.timer` (30 minutes), and `genesys.timer` (daily), each with its service: `ok` when the timer is active, the service's last result is success, and the last trigger is not older than twice the interval. A timer that has never triggered, with a next run scheduled and no failed result, SHALL be `waiting`. Anything else SHALL be `down`.
- `tesseract-restore.service`: `ok` when its last result is success, `down` when it failed. It need not be running.
- `s3://backups/c3po.sqlite` (10 minutes), `s3://backups/tesseract.sqlite` (15 minutes), and `s3://backups/rgw_slack_quickshell_plugin.sqlite` (30 minutes): `ok` when the object's mtime is newer than twice the interval, else `down`. The mtime SHALL come from `s3x.Stat`.
- `s3://sqlx-backups`: `ok` when the newest UTC day prefix is within two days, else `down`.
- `s3://backups/musica.sqlite` and `s3://backups/antibloat.sqlite`: `ok` when present, `down` when missing or when the stat fails. Age alone SHALL NOT make them `down`.
- `s3://backups/crusade.sqlite`: `ok` when present. A missing object SHALL be `waiting`, because crusade skips the put while its task table is empty. A failed stat SHALL be `down`.
- `nebula run list`, with a 5 second timeout: one row per hosted process it prints. An empty list SHALL be one `ok` row that says there are no hosted processes. A failure SHALL be one `down` row named `nebula run` with the error.

The genesys bucket SHALL NOT be scanned.

#### Scenario: A running unit is ok

- **GIVEN** `tunnel.service` is active and running
- **WHEN** the screen checks
- **THEN** its row SHALL be `ok`

#### Scenario: A stopped unit is down

- **GIVEN** `looptab.service` is inactive
- **WHEN** the screen checks
- **THEN** its row SHALL be `down`

#### Scenario: A timer inside its window is ok

- **GIVEN** `gmail.timer` is active, last triggered 30 minutes ago, and `gmail.service` succeeded
- **WHEN** the screen checks
- **THEN** its row SHALL be `ok`

#### Scenario: A stale timer is down

- **GIVEN** `gmail.timer` last triggered three hours ago and `gmail.service` succeeded
- **WHEN** the screen checks
- **THEN** its row SHALL be `down`

#### Scenario: A timer that has not fired is waiting

- **GIVEN** `c3po-snapshot.timer` is active, has never triggered, and has a next run scheduled
- **WHEN** the screen checks
- **THEN** its row SHALL be `waiting`

#### Scenario: A failed oneshot is down

- **GIVEN** `genesys.service` last exited with status 1
- **WHEN** the screen checks
- **THEN** the `genesys.timer` row SHALL be `down`
- **AND** the screen SHALL NOT start genesys

#### Scenario: Objects

- **GIVEN** `s3://backups/c3po.sqlite` was written five minutes ago
- **AND** `s3://backups/tesseract.sqlite` was written two hours ago
- **AND** `s3://backups/musica.sqlite` is missing
- **WHEN** the screen checks
- **THEN** the c3po row SHALL be `ok`
- **AND** the tesseract row SHALL be `down`
- **AND** the musica row SHALL be `down`

#### Scenario: A stat that hangs is down

- **GIVEN** a stat does not return within 5 seconds
- **WHEN** the screen checks
- **THEN** that row SHALL be `down` with a timeout reason
- **AND** the screen SHALL still show every other row

### Requirement: The health screen keys and output

`j` and `k` SHALL move and SHALL wrap. `g` and `G` SHALL jump to the first and the last row. `r` SHALL check again. `q` SHALL quit. `?` SHALL show the shortcut list, and a second `?` SHALL hide it. The screen SHALL NOT write shortcut text until `?`. While the list is open, `j` and `k` SHALL NOT move. The screen SHALL paint in black, white, and gray only: `down` SHALL be bold, `waiting` SHALL be dim, and `ok` SHALL be plain. When stdout is not a terminal, `r2d2 health` SHALL print the rows and SHALL exit 0 without opening the TUI. `r2d2 health help` SHALL explain the screen and SHALL exit 0.

#### Scenario: A pipe prints rows

- **WHEN** the operator runs `r2d2 health` with stdout on a pipe
- **THEN** each row SHALL be printed with its state and reason
- **AND** the command SHALL exit 0

#### Scenario: Shortcuts wait for question mark

- **WHEN** the screen opens
- **THEN** the view SHALL NOT contain shortcut text
- **AND** `?` SHALL show the list, which names `r` as reload
- **AND** `j` SHALL NOT move while the list is open
