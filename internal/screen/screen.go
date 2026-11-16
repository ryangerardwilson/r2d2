package screen

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Names are the screens r2d2 opens. web is implemented here. The others are
// the screens that still live in c3po, launched with c3po tui.
var Names = []string{"web", "cyclotree", "l", "o", "tunnel", "gdrive", "pass", "deport", "xl", "looptab"}

// Known reports whether name is a screen.
func Known(name string) bool {
	for _, screen := range Names {
		if screen == name {
			return true
		}
	}
	return false
}

// Command is the c3po invocation for a screen that is not web.
func Command(name string, args []string) []string {
	return append([]string{"tui", name}, args...)
}

// Launch runs a c3po screen. web is not launched here.
func Launch(name string, args []string, stdout, stderr io.Writer) int {
	path, err := exec.LookPath("c3po")
	if err != nil {
		fmt.Fprintln(stderr, "c3po is not installed")
		return 1
	}
	cmd := exec.Command(path, Command(name, args)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err = cmd.Run(); err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	fmt.Fprintln(stderr, err.Error())
	return 1
}
