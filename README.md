# r2d2

r2d2 is the terminal program. Running `r2d2` opens the screen menu. c3po is the agent CLI.

On a terminal, `r2d2` with no arguments opens that menu. `j` and `k` move, Enter opens the screen, and `q` quits. A pipe prints the screen list. `r2d2 help` prints the same list.

`r2d2 web` is the admin panel for [ryangerardwilson.com](https://ryangerardwilson.com). The other screens still run through `c3po tui`, so a session restore that launches `c3po tui <screen>` keeps working.

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
