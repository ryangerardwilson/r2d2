## MODIFIED Requirements

### Requirement: The looptab screen calls c3po looptab
`r2d2 looptab` SHALL be implemented in r2d2 and SHALL NOT be dispatched through the c3po screen package. It SHALL list the jobs `looptab now` prints. `j` and `k` SHALL move and SHALL wrap. Enter SHALL run the selected job once with `looptab now` and that row's index. `q` SHALL quit. `?` SHALL toggle the shortcut list, and the list and `r2d2 looptab help` SHALL NOT print shortcut text until then. `e` SHALL call the exported c3po `looptab.JobsFile`, SHALL open that file in vim, and after vim exits SHALL call `looptab.Install` and SHALL reload the job list. With no registered file, c3po SHALL create `~/.config/c3po/looptab/jobs` from its template and register it as `jobs`. When vim is not on `PATH`, the screen SHALL say `vim is not on PATH`, SHALL stay up, and SHALL NOT install. The screen SHALL NOT shell out to `c3po`, SHALL NOT write the Looptab scheduler file or name its path, and SHALL NOT create a config directory under `~/.config/r2d2`. `r2d2 looptab --resume <id>` SHALL restore the selected job through the c3po screen session. `r2d2 looptab help` SHALL name running a job and editing the jobs file. The menu line for looptab SHALL name both.

#### Scenario: e installs through c3po
- **WHEN** nothing is registered and the operator presses `e` and vim saves a job line
- **THEN** `~/.config/c3po/looptab/jobs` SHALL be registered as `jobs`
- **AND** c3po SHALL install it into the Looptab scheduler file it owns
- **AND** the job list SHALL reload

#### Scenario: Missing vim leaves the screen up
- **WHEN** vim is not on `PATH` and the operator presses `e`
- **THEN** the screen SHALL say `vim is not on PATH`
- **AND** the screen SHALL stay up and nothing SHALL be installed

#### Scenario: Help names the run and the edit
- **WHEN** the operator runs `r2d2 looptab help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say enter runs the selected job now and `e` edits the jobs file in vim
