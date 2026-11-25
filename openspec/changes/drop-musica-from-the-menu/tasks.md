## 1. Spec

- [x] 1.1 Validate with `c3po ast exec -- openspec validate drop-musica-from-the-menu --strict --no-interactive`
- [x] 1.2 Bring `openspec/specs/r2d2/spec.md` and `openspec/specs/musica-screen/spec.md` in line with the MODIFIED blocks
- [x] 1.3 Check that `screens-paint-first-and-read-keep` does not bring `musica` back as a screen, and validate it with `--strict`

## 2. Screens

- [x] 2.1 Drop `musica` from `screen.Screens` and `screen.Own`, and fix the comments that list the screens r2d2 runs itself
- [x] 2.2 Keep the `musica` case before `screen.Known` in `cli.Run`, and say in the help that it opens the Tesseract Musica center and which arguments pass through
- [x] 2.3 Say in `README.md` and `AGENTS.md` that musica is not in the screen menu

## 3. Check

- [x] 3.1 Test that the menu and the help's screen lines do not name musica, that `Known("musica")` is false and musica is not sent to c3po, and that `r2d2 musica help` is the player help
- [x] 3.2 Run `GOWORK=off go test ./...`
- [x] 3.3 Run `GOWORK=off go vet ./...`
- [x] 3.4 Validate `drop-musica-from-the-menu` and `screens-paint-first-and-read-keep` with `--strict --no-interactive` again

## 4. Center

- [x] 4.1 Bare `r2d2 musica` runs `open.sh musica` in its own session and does not call `player.Run`; arguments still go to `player.Run`
- [x] 4.2 Test the script path, the `$XDG_CONFIG_HOME` rule, the failure message, and that `help` and `end` do not open the center
- [x] 4.3 Bring `openspec/specs/musica-screen/spec.md` and `openspec/specs/r2d2/spec.md` in line with the MODIFIED blocks, and validate with `--strict`
- [x] 4.4 `GOWORK=off go test ./...` and `go vet ./...`. The require is the published tesseract module.
- [x] 4.5 Install from `/tmp` with `bash /home/ryan/OpenSource/r2d2/install.sh`
