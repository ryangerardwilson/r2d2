package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ryangerardwilson/musica/player"
	"github.com/ryangerardwilson/r2d2/internal/health"
	"github.com/ryangerardwilson/r2d2/internal/looptab"
	"github.com/ryangerardwilson/r2d2/internal/menu"
	"github.com/ryangerardwilson/r2d2/internal/rec"
	"github.com/ryangerardwilson/r2d2/internal/screen"
	"github.com/ryangerardwilson/r2d2/internal/skills"
	"github.com/ryangerardwilson/r2d2/internal/version"
	"github.com/ryangerardwilson/r2d2/internal/web"
)

// Help is the r2d2 help. A terminal opens the menu instead of printing it.
// The screen list comes from screen.Screens, which the menu also reads.
var Help = helpUsage + "screens:\n" + screenLines() + "\n" + helpTail

const helpUsage = `r2d2 is the terminal program. c3po is the agent CLI.

  r2d2
    open the screen menu
  r2d2 web
    ryangerardwilson.com admin panel
  r2d2 cyclotree [symbol] [dir]
  r2d2 l [path] [flags]
  r2d2 o [args...]
  r2d2 tunnel
  r2d2 gdrive
  r2d2 pass
  r2d2 deport
  r2d2 xl [file]
  r2d2 looptab
    run a Looptab job now, or edit the jobs file in vim
  r2d2 slack
  r2d2 health
    whether background jobs and nebula backups are working
  r2d2 musica [args...]
    the musica player; end, play, open, config, and help pass through
  r2d2 rec [start|pause|resume|stop|focus|quit|status|toggle]
    the camera in the corner of a screen recording, and its controls
  r2d2 skills
    create, edit, and delete the shared skills in vim; each change is pushed
  r2d2 version
  r2d2 help

`

const helpTail = `A pipe prints this text. A terminal opens the menu. j and k move, Enter
opens the screen, and q quits.
r2d2 web reads the admin API key with nebula crypt. Essays open in vim
with :Wrap, and the saved text is a flowing paragraph.
r2d2 <screen> help explains one screen.
`

func screenLines() string {
	var b strings.Builder
	for _, s := range screen.Screens {
		b.WriteString("  " + s.Line() + "\n")
	}
	return b.String()
}

// Run dispatches one screen.
func Run(args []string, stdout, stderr io.Writer) int {
	rest := []string{}
	if len(args) > 0 {
		rest = args[1:]
	}
	if len(rest) == 0 {
		if code, opened := menu.Run(stdout, stderr); opened {
			return code
		}
		fmt.Fprint(stdout, Help)
		return 0
	}
	if helpWord(rest[0]) {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if versionWord(rest[0]) && len(rest) == 1 {
		fmt.Fprintf(stdout, "r2d2 %s\n", version.String())
		return 0
	}
	name := rest[0]
	switch name {
	case "web":
		return web.Run(rest[1:], stdout, stderr)
	case "looptab":
		return looptab.Run(rest[1:], stdout, stderr)
	case "health":
		return health.Run(rest[1:], stdout, stderr)
	case "musica":
		return player.Run(rest[1:], os.Stdin, stdout, stderr)
	case "rec":
		return rec.Run(rest[1:], stdout, stderr)
	case "skills":
		return skills.Run(rest[1:], stdout, stderr)
	}
	if !screen.Known(name) {
		fmt.Fprintf(stderr, "unknown screen %q\n", name)
		fmt.Fprint(stderr, Help)
		return 2
	}
	return screen.Launch(name, rest[1:], stdout, stderr)
}

func helpWord(word string) bool {
	return word == "help" || word == "-h" || word == "--help"
}

func versionWord(word string) bool {
	return word == "version" || word == "--version" || word == "-v"
}
