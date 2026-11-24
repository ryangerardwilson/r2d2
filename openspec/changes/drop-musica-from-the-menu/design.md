## Context

`screen.Screens` feeds the menu, the pipe list, and the screen lines of `r2d2 help`. `screen.Own` names the screens r2d2 runs itself, so `screen.Launch` does not hand them to c3po. `cli.Run` dispatches `web`, `looptab`, `health`, `musica`, and `rec` in a switch before `screen.Known`, and sends every other known name to `screen.Launch`. In the musica repository, `player.Run` with no arguments is becoming the way into the Tesseract Musica center, the Super+T launcher showing Musica; `end`, `play`, `open`, `config`, `help`, and `download-worker` keep their behavior. r2d2 pins the published musica `v0.0.0-20261123015810-9f16b1ddb6c2`, whose `player.Run` with no arguments still opens the terminal player.

## Goals / Non-Goals

**Goals:**

- The menu and the screen list stop naming `musica`.
- `r2d2 musica` keeps working in process with every argument it takes today.
- Bare `r2d2 musica` shows the Tesseract Musica center now, without a musica pin bump.

**Non-Goals:**

- A musica pin bump, a `replace` directive, or any `go.mod` or `go.sum` change.
- Drawing the center or copying player code in r2d2.
- Renaming the `musica-screen` capability.

## Decisions

- `r2d2 musica` with arguments keeps dispatching to `player.Run` in process. The `musica` case stays in the switch before `screen.Known`, so a name that is not a screen still reaches the player and is not reported as unknown. Exec'ing the musica binary was the alternative; it adds a process and a `PATH` lookup for nothing.
- musica leaves the screen menu: it is removed from `screen.Screens` and `screen.Own`. The menu, the pipe list, and the help's screen lines all read `screen.Screens`, so one removal drops the row everywhere. `screen.Launch("musica")` now fails as an unknown screen, so musica is still never sent to c3po. Keeping the row and pointing it at the center was rejected: Enter would leave the terminal menu to show a launcher surface and return at once.
- The usage block of `r2d2 help` keeps its `r2d2 musica [args...]` line. It now says the command opens the Tesseract Musica center and that `end`, `play`, `open`, `config`, `help`, and `download-worker` pass through.
- Bare `r2d2 musica` runs `<config>/quickshell/omarchy-launcher/open.sh musica` itself, in its own session, and returns without waiting, as bare `musica` does; `<config>` is `$XDG_CONFIG_HOME` when it is absolute and `~/.config` otherwise. The pinned `player.Run` with no arguments opens the terminal player, and a terminal player holding the musica session is what the Tesseract center then has to take over, so waiting for a musica publish and a pin bump was rejected. The few lines that find and start the script repeat musica's `openCenter`. Once the pin carries it, the branch can call `player.Run` again.

## Risks / Trade-offs

- [The launcher path is this desktop's convention.] → When the script is missing, bare `r2d2 musica` exits 1 and names it; `r2d2 musica play` and `open` still open the terminal player.
- [`r2d2 musica end` runs the pinned `End`, which neither waits for the session to let go nor notes its track.] → The launcher runs the musica binary's `end`; the pin bump brings the rest.
- [An operator who opened musica from the r2d2 menu loses that row.] → Super+T shows the center, and `r2d2 musica` still works from a shell.
