package screen

import (
	"fmt"
	"io"

	c3poscreen "github.com/ryangerardwilson/c3po/screen"
)

// Names are the screens r2d2 opens. web, health, and musica are implemented
// here. The others run in this process through the c3po screen package.
var Names = []string{"web", "cyclotree", "l", "o", "tunnel", "gdrive", "pass", "deport", "xl", "looptab", "slack", "health", "musica"}

// Known reports whether name is a screen.
func Known(name string) bool {
	for _, screen := range Names {
		if screen == name {
			return true
		}
	}
	return false
}

// Own reports whether r2d2 runs the screen itself rather than through c3po.
func Own(name string) bool {
	return name == "web" || name == "health" || name == "musica"
}

// Launch runs a screen that lives in c3po. web, health, and musica are not
// launched here. Arguments after the screen name, including --resume, are
// passed through.
func Launch(name string, args []string, stdout, stderr io.Writer) int {
	if Own(name) || !Known(name) {
		fmt.Fprintf(stderr, "unknown screen %q\n", name)
		return 2
	}
	return c3poscreen.Run(name, args, stdout, stderr)
}
