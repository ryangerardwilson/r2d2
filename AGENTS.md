# AGENTS.md

This tool is part of a harness for 0→1 product engineering with AI agents. Prefer the smallest system that keeps rules visible and recoverable.

r2d2 is the terminal program. The module is `github.com/ryangerardwilson/r2d2`. The command is `r2d2`. c3po is the agent CLI. Agents use c3po and do not run r2d2.

c3po is the foundation and the source of truth for each operation. r2d2 is the TUI above it. An r2d2 screen opens the screen and calls exported c3po functions in process. It does not shell out to `c3po` and does not keep a second copy of a c3po operation. Config for a c3po operation stays under `~/.config/c3po/<app>/` or the sqlx database `c3po`; r2d2 does not grow a config directory for it. c3po does not import r2d2.

Running `r2d2` on a terminal opens the screen menu. `j` and `k` move, Enter opens the screen, and `q` quits. The menu and `r2d2 web` do not print shortcuts until `?` toggles them. A pipe prints the screen list and exits 0. `r2d2 help` prints that list.

`r2d2 web` is the ryangerardwilson.com admin screen. It reads the admin API key with `nebula crypt` (`crypt.Get` on vault Root, item `RGW_API_KEY`). It does not print that key. `R2D2_WEB_TOKEN` and `R2D2_WEB_ORIGIN` override the lookup and the origin.

Essays and long prose open in vim with `-c Wrap`. The saved text joins wrapped lines back into paragraphs. A blank line is a paragraph break. The first paragraph is the title.

`r2d2 health` reads systemd user units, nebula s3 objects (with `s3.Stat`, never a download), `nebula keep status --json`, and `nebula run list`. It never restarts or repairs anything. `r2d2 musica` is not a screen and is not in the menu. It is dispatched before the screen check. With no arguments, `internal/cli` runs `<config>/quickshell/omarchy-launcher/open.sh musica` in its own session, as bare `musica` does, and does not call `player.Run`, because that no-argument call opens the center on its own. `end`, `play`, `open`, `config`, `help`, and `download-worker` call `player.Run` from `github.com/ryangerardwilson/tesseract/player` in process, and downloads still run as `musica download-worker`. The player source stays in the tesseract repo.

`r2d2 looptab` is implemented in `internal/looptab`. It lists the jobs `looptab now` prints, and Enter runs the selected job once through `looptab now`. `e` calls `looptab.JobsFile` from c3po, opens that file in vim, then calls `looptab.Install` and reloads the list. With no registered file, c3po creates `~/.config/c3po/looptab/jobs` from its template and registers it as `jobs`. c3po is the only writer of the Looptab scheduler file, which it keeps in `~/.config/c3po/looptab/` beside the jobs files. r2d2 does not hold that path. The `looptab` binary reads it through the `~/.config/looptab` symlink. Missing vim says `vim is not on PATH` and leaves the screen up. Its session goes through `screen.OpenSession` in c3po.

`r2d2 rec` controls the rec camera, a GTK window that sits in the corner of the screen recording. The camera program is `camera/rec.py` in this repository. `install.sh` copies it to `~/.local/share/r2d2/rec.py` and writes `~/.local/bin/rec`. r2d2 talks to the camera over `$XDG_RUNTIME_DIR/rec.sock` and starts it as `~/.local/bin/rec` with `REC_INTERNAL=1`. r2d2 never draws the camera. On a terminal, the screen lists start, pause, resume, stop, you, and quit. `j` and `k` move, and Enter runs the selected row. `q` leaves the screen and leaves the camera as it is. Without a terminal, `r2d2 rec` opens the camera and exits 0.

`r2d2 skills` is implemented in `internal/skills`. It keeps a clone of `https://github.com/ryangerardwilson/skills` at `~/.local/share/r2d2/skills`, a checkout and not config, and lists each `plugins/skills/skills/<name>/SKILL.md`. That repository's `main` is the source of truth for the skills Claude Code, Grok Build, and Codex install as the one `skills` plugin; no c3po operation backs this screen, and it runs git itself. On open it clones a missing copy, and fetches and fast-forwards a clean `main` with no unpushed commits. A dirty worktree, unpushed commits, or another branch is left as it is and shown, and only `main` takes writes. A directory that is not that repository is never deleted; the screen shows an error and does not write. `n` creates a skill in vim, Enter edits one, and `d` removes one after `y`. Each commits `Add`, `Update`, or `Remove <name>` as Ryan Gerard Wilson and pushes `main`, never with force. A failed push stays on screen and keeps the commit. `r` fetches and fast-forwards. Missing vim says `vim is not on PATH`.

Screens other than `web`, `looptab`, `health`, `rec`, and `skills` run in this process through `github.com/ryangerardwilson/c3po/screen`. Require the published c3po, nebula, and tesseract modules. Do not add a `replace` directive.

Do not add this module to the home `go.work`. Test with `GOWORK=off go test ./...`. Install with `bash install.sh` from this checkout. Work and commit inside this Git root.

The accepted behavior is `openspec/specs/r2d2/spec.md`, `openspec/specs/health/spec.md`, `openspec/specs/musica-screen/spec.md`, `openspec/specs/rec-screen/spec.md`, and `openspec/specs/skills-screen/spec.md`.
