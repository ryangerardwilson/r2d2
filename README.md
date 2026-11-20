# r2d2

r2d2 is the terminal program. Running `r2d2` opens the screen menu. c3po is the agent CLI.

On a terminal, `r2d2` with no arguments opens that menu. `j` and `k` move, Enter opens the screen, and `q` quits. The menu and `r2d2 web` do not print shortcuts until `?` toggles them. A pipe prints the screen list. `r2d2 help` prints the same list.

`r2d2 web` is the admin panel for [ryangerardwilson.com](https://ryangerardwilson.com). The other screens run in this process. There is no `c3po tui` command. A session restores with `r2d2 <screen> --resume <id>`.

`r2d2 slack` shows the Slack desk. `h` and `l` switch workspace. The table is direct messages, tags, and what to know. Enter opens the row. `r` marks it read on the desk. Opening a row does not mark it read. `s` syncs the desk from Slack. The screen shows when the desk last synced. The user token is not shown.

`r2d2 health` shows whether the background jobs are working: the long-running user services, each timer and the last run of its oneshot, the backups on nebula s3, and each process `nebula run list` names. A row is `ok`, `waiting`, or `down` with a short reason. It checks on open and on `r`, and only reads. A pipe prints the rows.

`r2d2 musica` opens the [musica](https://github.com/ryangerardwilson/musica) player through its public `player` package. `r2d2 musica end`, `help`, `play`, `open`, and `config` pass through. Downloads still run as `musica download-worker`, so the musica binary stays installed.

`r2d2 rec` controls the rec camera, the circle in the corner of a screen recording. The camera stays its own window so it is in the recording. On a terminal the screen shows the status, the clock, whether the camera is in the corner or fills the screen, and the last error or saved file. `r2d2 rec start`, `pause`, `resume`, `stop`, `focus`, `quit`, `status`, and `toggle` print the camera's reply. Without a terminal, `r2d2 rec` opens the camera and exits 0.

```sh
r2d2 web
r2d2 cyclotree [symbol] [dir]
r2d2 l [path] [flags]
r2d2 o [args...]
r2d2 tunnel
r2d2 gdrive
r2d2 pass
r2d2 deport
r2d2 xl [file]
r2d2 looptab
r2d2 slack
r2d2 health
r2d2 musica [args...]
r2d2 rec [start|pause|resume|stop|focus|quit|status|toggle]
r2d2 version
r2d2 help
```

`r2d2 web` reads the admin API key with [nebula](https://github.com/ryangerardwilson/nebula) `crypt` from vault Root, item `RGW_API_KEY`. The key is not printed. `R2D2_WEB_TOKEN` overrides that lookup. `R2D2_WEB_ORIGIN` overrides `https://ryangerardwilson.com`.

Essays and pages open in vim. Vim runs `:Wrap` when the file loads, so the draft sits near 79 characters per line. A blank line keeps paragraphs apart. What is saved is the paragraph with those wraps removed, so the site renders a flowing paragraph. The title is the first paragraph.

## Install

Public source is https://github.com/ryangerardwilson/r2d2.

From a checkout:

```sh
./install.sh
```

The binary is installed to `$R2D2_INSTALL_DIR` or `~/.local/bin`. A clean checkout reports `r2d2 0.1.0+<commit>`. Uncommitted changes add `-dirty`.

```sh
GOBIN="$HOME/.local/bin" go install github.com/ryangerardwilson/r2d2/cmd/r2d2@v0.1.0
```

## Develop

```sh
GOWORK=off go test ./...
```

Do not add this module to a parent `go.work`.
