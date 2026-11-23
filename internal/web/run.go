package web

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryangerardwilson/nebula/crypt"
	"github.com/ryangerardwilson/r2d2/internal/version"
	"golang.org/x/term"
)

// Help is the help for r2d2 web.
const Help = `r2d2 web
the ryangerardwilson.com admin panel in the terminal

  r2d2 web
    open the panel
  r2d2 web help
    show this help

The admin API key is read with nebula crypt from vault Root, item RGW_API_KEY,
after the screen opens. It is not printed. R2D2_WEB_TOKEN overrides that lookup.
R2D2_WEB_ORIGIN overrides https://ryangerardwilson.com.

Essays and pages open in vim. Vim runs :Wrap when the file loads, so the
draft sits near 79 characters per line. A blank line keeps paragraphs apart.
What is saved is the paragraph with those wraps removed, so the site renders
a flowing paragraph. The title is the first paragraph.

j and k move. Enter edits. n creates. x deletes. On essays, u and d move a
future essay in the queue, and s edits the weekly slot. On the landing page,
p and c set the portrait and the cover from a file path. v opens the site.
q goes back, and q on the menu quits.

Keys work while the site answers. A move, a delete, or an edit shows at once
and the site's answer replaces it; a refusal puts the row back and names the
error. Writes go to the site one at a time, in the order they were asked.
q on the menu waits a short while for writes still going; q again quits.
`

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help") {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if len(args) == 1 && (args[0] == "version" || args[0] == "--version" || args[0] == "-v") {
		fmt.Fprintf(stdout, "r2d2 %s\n", version.String())
		return 0
	}
	if len(args) > 0 {
		fmt.Fprint(stderr, Help)
		return 2
	}
	if !isTerminal(stdout) {
		fmt.Fprint(stdout, Help)
		return 0
	}
	origin := strings.TrimSpace(os.Getenv("R2D2_WEB_ORIGIN"))
	if origin == "" {
		origin = "https://ryangerardwilson.com"
	}
	m := newModel(origin, os.Getenv("R2D2_WEB_TOKEN"), func() (string, error) {
		return crypt.Get("Root", "RGW_API_KEY")
	})
	program := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(stdout))
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(stderr, "r2d2: %s\n", err.Error())
		return 1
	}
	return 0
}

func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}
