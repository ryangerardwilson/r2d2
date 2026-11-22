package rec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// Help is the help for r2d2 rec.
const Help = `r2d2 rec
screen recorder and camera

  r2d2 rec
    open the camera and this screen
  r2d2 rec start | pause | resume | stop | focus | quit | toggle | status
    send one command to the camera and print its reply
  r2d2 rec open
    open the camera, or bring it back to its corner
  r2d2 rec --size small|medium|large
    camera size, remembered for the next time it opens
  r2d2 rec help
    show this help

The camera is a window in the corner of the screen, so it is in the
recording. This screen controls it and never draws it. Recordings include
your voice and are saved in Videos. focus fills the recording with the
camera, and again puts it back in the corner, in the same file.

stop saves the take and keeps the camera open. quit saves the take and
closes the camera. status, pause, resume, and stop print rec is not open
and exit 1 when the camera is closed. Without a terminal, r2d2 rec opens
the camera and exits 0. On a terminal, this screen is the control system.

The screen lists start, pause, resume, stop and save, fill the recording
with you, and save and close, then the recent recordings, newest first.
j and k move through that list. Enter runs the selected action. Enter on
a recording plays it. q and ctrl+c leave this screen and leave the camera
as it is.
`

var commands = map[string]bool{
	"open": true, "toggle": true, "start": true, "pause": true, "resume": true,
	"stop": true, "focus": true, "quit": true, "status": true,
}

var sizes = map[string]bool{"small": true, "medium": true, "large": true}

type deps struct {
	widget Widget
	tty    bool
	sleep  func(time.Duration)
	screen func(Client, io.Writer) (Result, error)
}

// Run opens r2d2 rec.
func Run(args []string, stdout, stderr io.Writer) int {
	tty := isTerminal(os.Stdin) && isTerminal(stdout)
	return run(args, stdout, stderr, deps{widget: live{}, tty: tty, screen: runScreen})
}

type request struct {
	command string
	size    string
	flags   []string
	help    bool
}

func run(args []string, stdout, stderr io.Writer, d deps) int {
	req, err := parse(args)
	if err != nil {
		fmt.Fprintf(stderr, "r2d2 rec: %s\n", err.Error())
		fmt.Fprint(stderr, Help)
		return 2
	}
	if req.help {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if req.size != "" {
		if err := saveSize(req.size); err != nil {
			fmt.Fprintf(stderr, "r2d2 rec: %s\n", err.Error())
			return 1
		}
	}
	client := Client{Widget: d.widget, Sleep: d.sleep, Flags: req.flags}
	if req.command == "" && d.tty {
		final, err := d.screen(client, stdout)
		if err != nil {
			fmt.Fprintf(stderr, "r2d2: %s\n", err.Error())
			return 1
		}
		show(final, stdout, stderr)
		return 0
	}
	if req.command == "" {
		req.command = "open"
	}
	res := client.Do(req.command)
	show(res, stdout, stderr)
	return res.Code
}

func show(res Result, stdout, stderr io.Writer) {
	if res.Out != "" {
		fmt.Fprintln(stdout, res.Out)
	}
	if res.Err != "" {
		fmt.Fprintln(stderr, res.Err)
	}
}

func parse(args []string) (request, error) {
	var req request
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, value, inline := strings.Cut(arg, "=")
		switch {
		case arg == "help" || arg == "-h" || arg == "--help":
			req.help = true
		case name == "--size" || name == "--device":
			if !inline {
				if i+1 >= len(args) {
					return req, fmt.Errorf("%s needs a value", name)
				}
				i++
				value = args[i]
			}
			if name == "--size" && !sizes[value] {
				return req, fmt.Errorf("--size must be small, medium, or large")
			}
			if name == "--size" {
				req.size = value
			}
			req.flags = append(req.flags, name, value)
		case commands[arg] && req.command == "":
			req.command = arg
		default:
			return req, fmt.Errorf("unknown argument %q", arg)
		}
	}
	return req, nil
}

// saveSize writes the camera size where rec.py reads it.
func saveSize(name string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".config", "rec", "size")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(name+"\n"), 0o644)
}

func runScreen(client Client, stdout io.Writer) (Result, error) {
	program := tea.NewProgram(newModel(client.Do, scanVideos, playWithMpv), tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(stdout))
	final, err := program.Run()
	if err != nil {
		return Result{}, err
	}
	m, _ := final.(model)
	return m.final, nil
}

func isTerminal(v any) bool {
	file, ok := v.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}
