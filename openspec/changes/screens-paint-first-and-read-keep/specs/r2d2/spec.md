## MODIFIED Requirements

### Requirement: The admin screen uses the site API

`r2d2 web` SHALL call the ryangerardwilson.com admin API. The default origin SHALL be `https://ryangerardwilson.com`. `R2D2_WEB_ORIGIN` SHALL override it. The admin API key SHALL be read with nebula `crypt` from vault Root, item `RGW_API_KEY`, unless `R2D2_WEB_TOKEN` is set. The key SHALL NOT be printed, including in an error. The screen SHALL paint before it reads the key, and SHALL read it in the background with a timeout. A non-terminal SHALL print the web help and SHALL exit 0. `r2d2 web help` SHALL exit 0 and SHALL name `:Wrap` and `RGW_API_KEY`. Extra arguments SHALL exit 2.

The screen SHALL list, create, edit, and delete essays, edit the landing page, list, create, edit, and delete pages, and list, create, and revoke API keys. It SHALL move a future essay up or down in the queue and SHALL edit the weekly publish slot. It SHALL set the portrait and the cover from a file path. It SHALL refuse to delete the pages `about`, `privacy`, and `fulfillment`.

While a request runs, every key SHALL keep working: navigation moves at once, and writes join one queue that the screen sends in order. A move, a delete, or a field edit SHALL show at once as pending on the list the site last answered; the site's answer, from the write's own response when it carries the list, SHALL replace that list whole, and a failure SHALL drop the pending change and show the error. The screen SHALL NOT edit its copy of the site's list.

#### Scenario: A missing proof action still saves

- **GIVEN** the proof route answers 404 or 405
- **WHEN** an essay is saved
- **THEN** the original title and body SHALL be saved

#### Scenario: A failed proof does not save

- **GIVEN** the proof route answers with another error
- **WHEN** an essay is saved
- **THEN** the essay SHALL NOT be written

### Requirement: The looptab screen calls c3po looptab
`r2d2 looptab` SHALL be implemented in r2d2 and SHALL NOT be dispatched through the c3po screen package. It SHALL list the jobs `looptab now` prints. `j` and `k` SHALL move and SHALL wrap. Enter SHALL run the selected job once with `looptab now` and that row's index. `q` SHALL quit. `?` SHALL toggle the shortcut list, and the list and `r2d2 looptab help` SHALL NOT print shortcut text until then. `e` SHALL call the exported c3po `looptab.JobsFile`, SHALL open that file in vim, and after vim exits SHALL call `looptab.Install` and SHALL reload the job list. Enter and `e` hand the terminal to the job or to vim; the reload after them, and every other `looptab now` listing, SHALL run in the background with a 10 second timeout while the screen paints `loading` and keeps `j`, `k`, `?`, and `q` working. Enter during a reload SHALL NOT run a job and SHALL say `still loading`. The cursor SHALL stay on the same job across a reload. With no registered file, c3po SHALL create `~/.config/c3po/looptab/jobs` from its template and register it as `jobs`. When vim is not on `PATH`, the screen SHALL say `vim is not on PATH`, SHALL stay up, and SHALL NOT install. The screen SHALL NOT shell out to `c3po`, SHALL NOT write the Looptab scheduler file or name its path, and SHALL NOT create a config directory under `~/.config/r2d2`. `r2d2 looptab --resume <id>` SHALL restore the selected job through the c3po screen session. The selected job SHALL be written to the session in the background, only when it changes, and leaving SHALL wait at most 2 seconds for that write. `r2d2 looptab help` SHALL name running a job and editing the jobs file. The menu line for looptab SHALL name both.

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
