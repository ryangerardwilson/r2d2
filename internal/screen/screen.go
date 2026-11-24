package screen

import (
	"fmt"
	"io"

	c3poscreen "github.com/ryangerardwilson/c3po/screen"
)

// Screen is one screen r2d2 opens and the line the menu and the help show for it.
type Screen struct {
	Name  string
	Blurb string
}

// Screens are the screens r2d2 opens, in menu order. web, looptab, health,
// rec, and skills are implemented here. The others run in this process through the
// c3po screen package. musica is not a screen: r2d2 musica calls the musica
// player package from internal/cli and is not in the menu.
var Screens = []Screen{
	{"web", "ryangerardwilson.com admin panel"},
	{"cyclotree", "calls of one function, as a tree you move with hjkl"},
	{"l", "compact gray tree of a directory"},
	{"o", "vim-inspired file navigator"},
	{"tunnel", "Cloudflare Tunnel route editor"},
	{"gdrive", "Drive preset, then browse"},
	{"pass", "1Password vaults, then copy one item"},
	{"deport", "occupied external ports, freed with dd"},
	{"xl", "spreadsheet for a budget workbook"},
	{"looptab", "Looptab jobs, run one now or edit the jobs file"},
	{"slack", "direct messages, tags, and what to know"},
	{"health", "background jobs and nebula backups"},
	{"rec", "screen recorder and camera"},
	{"skills", "shared skills for Claude Code, Grok Build, and Codex"},
}

// Line is the screen as the menu and the help list it, after a two-column mark.
func (s Screen) Line() string {
	return fmt.Sprintf("%-12s %s", s.Name, s.Blurb)
}

// Known reports whether name is a screen.
func Known(name string) bool {
	for _, screen := range Screens {
		if screen.Name == name {
			return true
		}
	}
	return false
}

// Own reports whether r2d2 runs the screen itself rather than through c3po.
func Own(name string) bool {
	return name == "web" || name == "looptab" || name == "health" || name == "rec" || name == "skills"
}

// Launch runs a screen that lives in c3po. web, looptab, health, rec, and
// skills are not launched here, and musica is not a screen. Arguments after the screen
// name, including --resume, are passed through.
func Launch(name string, args []string, stdout, stderr io.Writer) int {
	if Own(name) || !Known(name) {
		fmt.Fprintf(stderr, "unknown screen %q\n", name)
		return 2
	}
	return c3poscreen.Run(name, args, stdout, stderr)
}
