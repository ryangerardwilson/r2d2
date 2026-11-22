// Package looptab is r2d2 looptab, the screen over c3po looptab. It lists the
// jobs looptab now prints, runs one, and edits the jobs file in vim. c3po
// names the jobs file and installs it; this screen does not write the
// scheduler file.
package looptab

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	c3polooptab "github.com/ryangerardwilson/c3po/looptab"
	c3poscreen "github.com/ryangerardwilson/c3po/screen"
	"github.com/ryangerardwilson/r2d2/internal/version"
)

var (
	errNotOnPath = errors.New("looptab is not on PATH")
	errNoVim     = errors.New("vim is not on PATH")
)

// Help is the help for r2d2 looptab.
const Help = `r2d2 looptab
run a Looptab job now, or edit the jobs file in vim

  r2d2 looptab
    list the jobs from looptab now
  r2d2 looptab help
    show this help

enter runs the selected job now, through looptab now and that row's index.
e edits the jobs file in vim. When vim exits, c3po installs the registered
jobs files onto looptab and the list reloads. With no registered file, c3po
creates ~/.config/c3po/looptab/jobs from a template and registers it.
? lists the shortcuts.
`

// Run serves r2d2 looptab. args are the arguments after the screen name.
func Run(args []string, stdout, stderr io.Writer) int {
	id, rest, err := c3poscreen.TakeResume(args)
	if err != nil {
		fmt.Fprint(stderr, Help)
		return 2
	}
	if len(rest) == 1 && isHelp(rest[0]) {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if len(rest) == 1 && isVersion(rest[0]) {
		fmt.Fprintln(stdout, "r2d2 "+version.String())
		return 0
	}
	if len(rest) != 0 {
		fmt.Fprint(stderr, Help)
		return 2
	}
	return openScreen(id, stderr)
}

func openScreen(id string, stderr io.Writer) int {
	if err := show(id); err != nil {
		fmt.Fprintf(stderr, "looptab: %s\n", err.Error())
		return 1
	}
	return 0
}

// show runs the job list until the operator quits. The terminal is restored
// before an error is printed.
func show(id string) error {
	term, err := openTerminal()
	if err != nil {
		return err
	}
	defer term.close()
	sess, err := c3poscreen.OpenSession("looptab", id)
	if err != nil {
		return err
	}
	defer sess.Close()
	bin, lookErr := exec.LookPath("looptab")
	m := &model{
		load: func() ([]job, error) {
			if lookErr != nil {
				return nil, errNotOnPath
			}
			return loadJobs(bin)
		},
		session: sess,
		run: func(j job) error {
			return term.away(func() error { return runNow(bin, j.Index) })
		},
		edit: editor{
			lookVim:  func() (string, error) { return exec.LookPath("vim") },
			jobsFile: c3polooptab.JobsFile,
			install:  c3polooptab.Install,
			open: func(vim, path string) error {
				return term.away(func() error { return runVim(vim, path) })
			},
		}.edit,
	}
	return m.loop(term)
}

// editor opens the jobs file c3po names in vim and then asks c3po to install
// every registered jobs file.
type editor struct {
	lookVim  func() (string, error)
	jobsFile func() (string, error)
	install  func() (int, error)
	open     func(vim, path string) error
}

func (e editor) edit() (int, error) {
	vim, err := e.lookVim()
	if err != nil {
		return 0, errNoVim
	}
	path, err := e.jobsFile()
	if err != nil {
		return 0, err
	}
	if err := e.open(vim, path); err != nil {
		return 0, err
	}
	return e.install()
}

func runVim(vim, path string) error {
	cmd := exec.Command(vim, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func isHelp(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

func isVersion(arg string) bool {
	return arg == "version" || arg == "-v" || arg == "--version"
}
