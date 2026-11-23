package health

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// Help is the help for r2d2 health.
const Help = `r2d2 health
whether background jobs and nebula backups are working

  r2d2 health
    open the screen
  r2d2 health help
    show this help

Each row is one check: a name, ok, waiting, or down, and a short reason.
The title counts the down rows. waiting counts as healthy: a timer that
has not fired yet, or a backup that is skipped while its source is empty.

Units are read with systemctl --user show. A service must be running. A
timer must be active, its last run must have succeeded, and that run must
be newer than twice its interval. Backups on nebula s3 are checked with a
stat of the object, never a download. nebula keep status names each
database it copies: ok, waiting once its copy has lagged 15 minutes, and
down while the upload fails. A nebula without keep is one row that says
so. nebula run list names each hosted process. Each s3, nebula, and ssh
check gives up after 5 seconds and is down.

The screen only reads. It never restarts, stops, or repairs anything.
It checks when it opens and again on r. On a pipe it prints the rows
and exits 0.

j and k move, g and G jump to the ends, r checks again, ? shows the
shortcuts, and q quits.
`

// Run opens r2d2 health.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, live{}, time.Now, isTerminal(stdout))
}

func run(args []string, stdout, stderr io.Writer, c Checker, clock func() time.Time, tty bool) int {
	if len(args) == 1 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help") {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if len(args) > 0 {
		fmt.Fprint(stderr, Help)
		return 2
	}
	check := func() []Row { return Check(context.Background(), c, clock()) }
	if !tty {
		fmt.Fprint(stdout, Plain(check()))
		return 0
	}
	program := tea.NewProgram(newModel(check), tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(stdout))
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
