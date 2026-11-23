// Package skills is r2d2 skills, the screen over the skills repository. It
// keeps a clone of github.com/ryangerardwilson/skills, lists its skills, and
// creates, edits, and deletes them in vim. Each change is committed and
// pushed to main, which every host installs from.
package skills

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	c3poscreen "github.com/ryangerardwilson/c3po/screen"
	"golang.org/x/term"
)

// Help is the help for r2d2 skills.
const Help = `r2d2 skills
create, edit, and delete the shared skills for Claude Code, Grok Build, and Codex

  r2d2 skills
    open the skill list
  r2d2 skills help
    show this help

main on https://github.com/ryangerardwilson/skills is the source of truth.
The screen keeps a clone at ~/.local/share/r2d2/skills and lists each
plugins/skills/skills/<name>/SKILL.md. On open, a missing clone is cloned,
and a clean main with no unpushed commits is fetched and fast-forwarded. A
dirty worktree, unpushed commits, or another branch is left as it is and
the screen says so. A directory that is not the skills repository is never
deleted; the screen shows an error and does not write.

n asks for a name and opens a new SKILL.md in vim. A name is lowercase
letters, digits, and hyphens, and does not start or end with a hyphen.
rn renames the selected skill. The field starts as the current name.
enter opens the selected SKILL.md in vim. d removes the selected skill
after y. Each one commits Add, Update, Remove, or Rename <name> as Ryan
Gerard Wilson and pushes main, never with force. A failed push stays on
screen and keeps the commit. rr fetches and fast-forwards main. r and
then any other key cancels.

j and k move, g and G jump to the ends, ? shows the shortcuts, and q
quits. A pipe prints this help.
`

// Run serves r2d2 skills. args are the arguments after the screen name.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, isTerminal(os.Stdin) && isTerminal(stdout))
}

func run(args []string, stdout, stderr io.Writer, tty bool) int {
	_, rest, err := c3poscreen.TakeResume(args)
	if err != nil {
		fmt.Fprint(stderr, Help)
		return 2
	}
	if len(rest) == 1 && isHelp(rest[0]) {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if len(rest) != 0 {
		fmt.Fprint(stderr, Help)
		return 2
	}
	if !tty {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if err := show(); err != nil {
		fmt.Fprintf(stderr, "skills: %s\n", err.Error())
		return 1
	}
	return 0
}

// show runs the skill list until the operator quits. The terminal is
// restored before an error is printed.
func show() error {
	dir, err := defaultDir()
	if err != nil {
		return err
	}
	term, err := openTerminal()
	if err != nil {
		return err
	}
	defer term.close()
	m := &model{
		repo:    &repo{dir: dir, remote: remoteURL, git: runGit},
		lookVim: func() (string, error) { return exec.LookPath("vim") },
		openVim: func(vim, path string) error {
			return term.away(func() error { return runVim(vim, path) })
		},
	}
	return m.loop(term)
}

func runVim(vim, path string) error {
	cmd := exec.Command(vim, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func isTerminal(w any) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func isHelp(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}
