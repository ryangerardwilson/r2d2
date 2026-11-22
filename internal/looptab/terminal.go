package looptab

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type terminal struct {
	in    *os.File
	out   *os.File
	state *term.State
}

func openTerminal() (*terminal, error) {
	t := &terminal{in: os.Stdin, out: os.Stdout}
	if err := t.resume(); err != nil {
		return nil, fmt.Errorf("looptab screen needs a terminal: %w", err)
	}
	return t, nil
}

// close leaves the alternate screen and restores the terminal mode.
func (t *terminal) close() {
	_, _ = t.out.WriteString("\x1b[?25h\x1b[?1049l")
	if t.state != nil {
		_ = term.Restore(int(t.in.Fd()), t.state)
		t.state = nil
	}
}

// resume returns to the job list.
func (t *terminal) resume() error {
	state, err := term.MakeRaw(int(t.in.Fd()))
	if err != nil {
		return err
	}
	t.state = state
	_, _ = t.out.WriteString("\x1b[?1049h\x1b[?25l")
	return nil
}

// away gives the terminal to run, a job or vim, and then returns to the list.
func (t *terminal) away(run func() error) error {
	t.close()
	err := run()
	if rerr := t.resume(); err == nil {
		err = rerr
	}
	return err
}

func (t *terminal) size() (int, int) {
	width, height, err := term.GetSize(int(t.out.Fd()))
	if err != nil || width == 0 || height == 0 {
		return 80, 24
	}
	return width, height
}

// paint draws frame from the top left. Raw mode turns off output
// processing, so each line ends with a carriage return.
func (t *terminal) paint(frame string) {
	lines := strings.Split(strings.TrimSuffix(frame, "\n"), "\n")
	var b strings.Builder
	b.WriteString("\x1b[H")
	for i, line := range lines {
		if i > 0 {
			b.WriteString("\r\n")
		}
		b.WriteString(line)
		b.WriteString("\x1b[K")
	}
	b.WriteString("\x1b[J")
	_, _ = t.out.WriteString(b.String())
}

func (t *terminal) readKey() (string, error) {
	buf := make([]byte, 16)
	n, err := t.in.Read(buf)
	if err != nil {
		return "", err
	}
	return decodeKey(buf[:n]), nil
}

var escapeKeys = map[string]string{
	"[A": "up", "OA": "up",
	"[B": "down", "OB": "down",
	"[C": "right", "OC": "right",
	"[D": "left", "OD": "left",
	"[H": "home", "OH": "home", "[1~": "home", "[7~": "home",
	"[F": "end", "OF": "end", "[4~": "end", "[8~": "end",
}

func decodeKey(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if data[0] == 0x1b && len(data) > 1 {
		return escapeKeys[string(data[1:])]
	}
	switch data[0] {
	case 0x03:
		return "ctrl+c"
	case 0x1b:
		return "esc"
	case '\r', '\n':
		return "enter"
	case 0x7f, 0x08:
		return "backspace"
	default:
		return string(data[:1])
	}
}
