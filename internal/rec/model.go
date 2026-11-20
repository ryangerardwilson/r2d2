package rec

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	do       func(string) Result
	word     string
	clock    string
	focus    bool
	note     string
	kind     string
	working  string
	seq      int
	polling  bool
	help     bool
	quitting bool
	final    Result
}

type tick struct{}

type polled struct{ res Result }

type done struct {
	seq     int
	command string
	res     Result
}

var keyCommands = map[string]string{
	"1": "start", "2": "pause", "3": "resume", "4": "stop", "5": "focus", " ": "toggle",
	"6": "quit", "q": "quit",
}

func newModel(do func(string) Result) model {
	return model{do: do, word: "closed", working: "open"}
}

// Init opens the camera if it is closed and starts the once-a-second status.
func (m model) Init() tea.Cmd {
	return tea.Batch(send(m.seq, "open", m.do), every())
}

func every() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tick{} })
}

func send(seq int, command string, do func(string) Result) tea.Cmd {
	return func() tea.Msg { return done{seq, command, do(command)} }
}

func (m model) poll() (model, tea.Cmd) {
	if m.polling || m.quitting {
		return m, nil
	}
	m.polling = true
	do := m.do
	return m, func() tea.Msg { return polled{do("status")} }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.key(msg.String())
	case tick:
		m, cmd := m.poll()
		return m, tea.Batch(cmd, every())
	case polled:
		m.polling = false
		if msg.res.Code == 0 {
			m.status(msg.res.Out)
		} else {
			m.word, m.clock = "closed", ""
		}
		return m, nil
	case done:
		return m.finish(msg)
	default:
		return m, nil
	}
}

func (m model) key(key string) (tea.Model, tea.Cmd) {
	if key == "ctrl+c" {
		return m, tea.Quit
	}
	if m.quitting {
		return m, nil
	}
	switch key {
	case "?":
		m.help = !m.help
		return m, nil
	case "esc":
		m.help = false
		return m, nil
	}
	if m.help && key != "q" {
		return m, nil
	}
	if key == "r" {
		return m.poll()
	}
	command, ok := keyCommands[key]
	if !ok {
		return m, nil
	}
	if command == "quit" {
		m.quitting, m.help = true, false
	}
	m.seq++
	m.working = command
	return m, send(m.seq, command, m.do)
}

func (m model) finish(msg done) (tea.Model, tea.Cmd) {
	if msg.command == "quit" {
		m.final = msg.res
		return m, tea.Quit
	}
	if msg.seq == m.seq {
		m.working = ""
	}
	res := msg.res
	switch {
	case res.Code != 0:
		m.note, m.kind = firstText(res.Err, res.Out), "error"
		if res.Err == notOpen {
			m.word, m.clock = "closed", ""
		}
	case strings.HasSuffix(res.Out, ".mp4"):
		m.note, m.kind = res.Out, "saved"
	case !m.status(res.Out) && res.Out != "":
		m.note, m.kind = res.Out, "reply"
	}
	if msg.command == "open" {
		return m.poll()
	}
	return m, nil
}

// status applies a widget status line. It is false for a line that is not a status.
func (m *model) status(line string) bool {
	s, ok := parseStatus(line)
	if !ok {
		return false
	}
	m.word, m.clock = s.word, s.clock
	if s.err != "" {
		m.note, m.kind = s.err, "error"
		return true
	}
	m.focus = s.focus
	return true
}

type state struct {
	word  string
	clock string
	focus bool
	err   string
}

// parseStatus reads recording 1:05, paused 0:01 focus, starting, idle focus, or idle Could not start.
func parseStatus(line string) (state, bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return state{}, false
	}
	s := state{word: fields[0]}
	rest := fields[1:]
	switch s.word {
	case "recording", "paused":
		if len(rest) > 0 && strings.Contains(rest[0], ":") {
			s.clock, rest = rest[0], rest[1:]
		}
	case "starting", "stopping", "saving":
	case "idle":
		if len(rest) > 0 && strings.Join(rest, " ") != "focus" {
			s.err = strings.Join(rest, " ")
			return s, true
		}
	default:
		return state{}, false
	}
	s.focus = len(rest) == 1 && rest[0] == "focus"
	return s, true
}

func firstText(texts ...string) string {
	for _, text := range texts {
		if text = strings.TrimSpace(text); text != "" {
			return text
		}
	}
	return "failed"
}
