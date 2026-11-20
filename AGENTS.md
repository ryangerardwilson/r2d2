# AGENTS.md

r2d2 is the terminal program. The module is `github.com/ryangerardwilson/r2d2`. The command is `r2d2`. c3po is the agent CLI. Agents use c3po and do not run r2d2.

Running `r2d2` on a terminal opens the screen menu. `j` and `k` move, Enter opens the screen, and `q` quits. The menu and `r2d2 web` do not print shortcuts until `?` toggles them. A pipe prints the screen list and exits 0. `r2d2 help` prints that list.

`r2d2 web` is the ryangerardwilson.com admin screen. It reads the admin API key with `nebula crypt` (`crypt.Get` on vault Root, item `RGW_API_KEY`). It does not print that key. `R2D2_WEB_TOKEN` and `R2D2_WEB_ORIGIN` override the lookup and the origin.

Essays and long prose open in vim with `-c Wrap`. The saved text joins wrapped lines back into paragraphs. A blank line is a paragraph break. The first paragraph is the title.

`r2d2 health` reads systemd user units, nebula s3 objects (with `s3x.Stat`, never a download), and `nebula run list`. It never restarts or repairs anything. `r2d2 musica` calls `player.Run` from `github.com/ryangerardwilson/musica/player`. The player source stays in the musica repo.

`r2d2 rec` controls the rec camera, a GTK window that sits in the corner of the screen recording. The camera program is `camera/rec.py` in this repository. `install.sh` copies it to `~/.local/share/r2d2/rec.py` and writes `~/.local/bin/rec`. r2d2 talks to the camera over `$XDG_RUNTIME_DIR/rec.sock` and starts it as `~/.local/bin/rec` with `REC_INTERNAL=1`. r2d2 never draws the camera. On a terminal, the screen lists start, pause, resume, stop, you, and quit. `j` and `k` move, and Enter runs the selected row. `q` leaves the screen and leaves the camera as it is. Without a terminal, `r2d2 rec` opens the camera and exits 0.

Screens other than `web`, `health`, `musica`, and `rec` run in this process through `github.com/ryangerardwilson/c3po/screen`. Require the published c3po, nebula, and musica modules. Do not add a `replace` directive.

Do not add this module to the home `go.work`. Test with `GOWORK=off go test ./...`. Install with `bash install.sh` from this checkout. Work and commit inside this Git root.

The accepted behavior is `openspec/specs/r2d2/spec.md`, `openspec/specs/health/spec.md`, `openspec/specs/musica-screen/spec.md`, and `openspec/specs/rec-screen/spec.md`.
