package menu

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryangerardwilson/r2d2/internal/screen"
	"golang.org/x/term"
)

var blurbs = map[string]string{
	"web":       "ryangerardwilson.com admin panel",
	"cyclotree": "calls of one function, as a tree you move with hjkl",
	"l":         "compact gray tree of a directory",
	"o":         "vim-inspired file navigator",
	"tunnel":    "Cloudflare Tunnel route editor",
	"gdrive":    "Drive preset, then browse",
	"pass":      "1Password vaults, then copy one item",
	"deport":    "occupied external ports, freed with dd",
	"xl":        "spreadsheet for a budget workbook",
	"looptab":   "Looptab jobs, run one now",
	"slack":     "direct messages, tags, and what to know",
	"health":    "background jobs and nebula backups",
	"musica":    "player, cassettes, and downloads",
	"rec":       "screen recorder and camera",
}

type model struct {
	cursor int
	status string
	help   bool
}

type cameBack struct{ err error }

// Run opens the screen menu when stdout is a terminal.
// The second result is false when there is no terminal, and nothing is printed.
func Run(stdout, stderr io.Writer) (int, bool) {
	if !isTerminal(stdout) {
		return 0, false
	}
	program := tea.NewProgram(model{}, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(stdout))
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(stderr, "r2d2: %s\n", err.Error())
		return 1, true
	}
	return 0, true
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?":
			m.help = !m.help
			return m, nil
		case "esc":
			if m.help {
				m.help = false
				return m, nil
			}
			return m, tea.Quit
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.help {
				return m, nil
			}
			if m.cursor < len(screen.Names)-1 {
				m.cursor++
			}
			return m, nil
		case "k", "up":
			if m.help {
				return m, nil
			}
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "enter":
			if m.help {
				return m, nil
			}
			cmd := exec.Command(executable(), screen.Names[m.cursor])
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return cameBack{err}
			})
		default:
			return m, nil
		}
	case cameBack:
		m.status = exitStatus(msg.err)
		return m, nil
	default:
		return m, nil
	}
}

func (m model) View() string {
	if m.help {
		return "r2d2\n\nj/k    move\nenter  open\n?      shortcuts\nq      quit\n"
	}
	var b strings.Builder
	b.WriteString("r2d2\n\n")
	for i, name := range screen.Names {
		mark := "  "
		if i == m.cursor {
			mark = "> "
		}
		fmt.Fprintf(&b, "%s%-12s %s\n", mark, name, blurbs[name])
	}
	if m.status != "" {
		fmt.Fprintf(&b, "\n%s", m.status)
	}
	b.WriteString("\n")
	return b.String()
}

func exitStatus(err error) string {
	if err == nil {
		return ""
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return fmt.Sprintf("screen exited %d", exit.ExitCode())
	}
	return err.Error()
}

func executable() string {
	path, err := os.Executable()
	if err != nil || path == "" {
		return os.Args[0]
	}
	return path
}

func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}
