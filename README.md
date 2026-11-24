# r2d2

r2d2 is the terminal program. Running `r2d2` opens the screen menu. c3po is the agent CLI.

On a terminal, `r2d2` with no arguments opens that menu. `j` and `k` move, Enter opens the screen, and `q` quits. The menu and `r2d2 web` do not print shortcuts until `?` toggles them. A pipe prints the screen list. `r2d2 help` prints the same list.

`r2d2 web` is the admin panel for [ryangerardwilson.com](https://ryangerardwilson.com). The other screens run in this process. There is no `c3po tui` command. A session restores with `r2d2 <screen> --resume <id>`.

`r2d2 looptab` lists the Looptab jobs. Enter runs the selected job once through `looptab now`. `e` opens the jobs file in vim, and when vim exits c3po installs it onto the scheduler and the list reloads. The screen calls the exported c3po looptab functions; `c3po looptab` is the command line and does not open vim.

`r2d2 slack` shows the Slack desk. `h` and `l` switch workspace. The table is direct messages, tags, and what to know. Enter opens the row. `r` marks it read on the desk. Opening a row does not mark it read. `s` syncs the desk from Slack. The screen shows when the desk last synced. The user token is not shown.

`r2d2 health` shows whether the background jobs are working: the long-running user services, each timer and the last run of its oneshot, the backups on nebula s3, how far each `nebula keep` copy lags, and each process `nebula run list` names. A row is `ok`, `waiting`, or `down` with a short reason. It checks on open and on `r`, and only reads. A pipe prints the rows.

`r2d2 musica` is not in the screen menu. With no arguments it runs the launcher's `~/.config/quickshell/omarchy-launcher/open.sh musica` in its own session, as bare `musica` does, which shows the Tesseract Musica center, the Super+T launcher showing Musica. It does not open the terminal player. `r2d2 musica end`, `play`, `open`, `config`, `help`, and `download-worker` call the public `player` package of [musica](https://github.com/ryangerardwilson/musica) in this process. Downloads still run as `musica download-worker`, so the musica binary stays installed.

`r2d2 rec` controls the rec camera, the circle in the corner of a screen recording. The camera stays its own window so it is in the recording. On a terminal the screen shows the status, the clock, whether the camera is in the corner or fills the screen, and the last error or saved file. `r2d2 rec start`, `pause`, `resume`, `stop`, `focus`, `quit`, `status`, and `toggle` print the camera's reply. Without a terminal, `r2d2 rec` opens the camera and exits 0.

`r2d2 skills` edits the shared skills for Claude Code, Grok Build, and Codex. Their source of truth is `main` on [ryangerardwilson/skills](https://github.com/ryangerardwilson/skills), which each host installs as one plugin named `skills`. The screen keeps a clone at `~/.local/share/r2d2/skills` and lists the skills. `n` creates one in vim, Enter edits one, and `d` removes one. Each change is committed and pushed to `main`. A dirty clone, unpushed commits, or another branch is left as it is and the screen says so.

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

## Operator console

Agents use `c3po`. Operators use `r2d2`.

The same tools (call tree with complexity, tunnels, secrets, scheduled jobs, site admin) appear as terminal screens so a human can review, override, or continue the work without reading agent logs. Live complexity and the call chain stay visible before any change ships.

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

Part of an open-source harness for shipping the smallest inspectable system that agents and operators can run and improve. See also [nebula](https://github.com/ryangerardwilson/nebula), [c3po](https://github.com/ryangerardwilson/c3po), [r2d2](https://github.com/ryangerardwilson/r2d2), [crusade](https://github.com/ryangerardwilson/crusade), [tesseract](https://github.com/ryangerardwilson/tesseract).
