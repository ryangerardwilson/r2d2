# AGENTS.md

r2d2 is the terminal-screen program. The module is `github.com/ryangerardwilson/r2d2`. The command is `r2d2`. c3po stays the agent CLI.

`r2d2 web` is the ryangerardwilson.com admin screen. It reads the admin API key with `nebula crypt` (`crypt.Get` on vault Root, item `RGW_API_KEY`). It does not print that key. `R2D2_WEB_TOKEN` and `R2D2_WEB_ORIGIN` override the lookup and the origin.

Essays and long prose open in vim with `-c Wrap`. The saved text joins wrapped lines back into paragraphs. A blank line is a paragraph break. The first paragraph is the title.

Screens other than `web` launch `c3po tui <screen>`. Do not import c3po. Do not add a `replace` directive to nebula. Require the published nebula module.

Do not add this module to the home `go.work`. Test with `GOWORK=off go test ./...`. Install with `bash install.sh` from this checkout. Work and commit inside this Git root.

The accepted behavior is `openspec/specs/r2d2/spec.md`.
