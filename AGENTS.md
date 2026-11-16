# AGENTS.md

r2d2 is the terminal program. The module is `github.com/ryangerardwilson/r2d2`. The command is `r2d2`. c3po is the agent CLI. Agents use c3po and do not run r2d2.

Running `r2d2` on a terminal opens the screen menu. `j` and `k` move, Enter opens the screen, and `q` quits. The menu and `r2d2 web` do not print shortcuts until `?` toggles them. A pipe prints the screen list and exits 0. `r2d2 help` prints that list.

`r2d2 web` is the ryangerardwilson.com admin screen. It reads the admin API key with `nebula crypt` (`crypt.Get` on vault Root, item `RGW_API_KEY`). It does not print that key. `R2D2_WEB_TOKEN` and `R2D2_WEB_ORIGIN` override the lookup and the origin.

Essays and long prose open in vim with `-c Wrap`. The saved text joins wrapped lines back into paragraphs. A blank line is a paragraph break. The first paragraph is the title.

Screens other than `web` run in this process through `github.com/ryangerardwilson/c3po/screen`. Require the published c3po and nebula modules. Do not add a `replace` directive.

Do not add this module to the home `go.work`. Test with `GOWORK=off go test ./...`. Install with `bash install.sh` from this checkout. Work and commit inside this Git root.

The accepted behavior is `openspec/specs/r2d2/spec.md`.
