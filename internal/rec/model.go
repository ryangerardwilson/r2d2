package rec

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	do         func(string) Result
	word       string
	clock      string
	focus      bool
	note       string
	kind       string
	working    string
	seq        int
	polling    bool
	cursor     int
	recordings []recording
	final      Result
	// scan lists the recent recordings. playback opens one in its own window.
	scan     func() []recording
	playback func(path string) error
}

type tick struct{}

type polled struct{ res Result }

type done struct {
	seq     int
	command string
	res     Result
}

func newModel(do func(string) Result, scan func() []recording, playback func(string) error) model {
	m := model{do: do, word: "closed", working: "open", scan: scan, playback: playback}
	m.reloadRecordings()
	return m
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
	if m.polling || m.quitting() {
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

type item struct {
	command string
	path    string
	label   string
}

// items are the rows on the screen. The focus row names what Enter will do.
func (m model) items() []item {
	you := "fill the recording with you"
	if m.focus {
		you = "put the camera back in the corner"
	}
	rows := []item{
		{"start", "", "start"},
		{"pause", "", "pause"},
		{"resume", "", "resume"},
		{"stop", "", "stop and save, camera stays open"},
		{"focus", "", you},
		{"quit", "", "save and close"},
	}
	for _, rec := range m.recordings {
		rows = append(rows, item{path: rec.path, label: rec.label})
	}
	return rows
}

func (m *model) reloadRecordings() {
	m.recordings = m.scan()
	if m.cursor < 0 {
		m.cursor = 0
	}
	if n := len(m.items()); n > 0 && m.cursor >= n {
		m.cursor = n - 1
	}
}

func (m model) key(key string) (tea.Model, tea.Cmd) {
	if m.quitting() {
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}
	if key == "ctrl+c" || key == "q" {
		return m, tea.Quit
	}
	rows := m.items()
	switch key {
	case "j", "down":
		m.cursor = cycled(m.cursor, 1, len(rows))
		return m, nil
	case "k", "up":
		m.cursor = cycled(m.cursor, -1, len(rows))
		return m, nil
	case "g", "home":
		m.cursor = 0
		return m, nil
	case "G", "end":
		m.cursor = len(rows) - 1
		return m, nil
	case "r":
		return m.poll()
	case "enter":
		row := rows[m.cursor]
		if row.path != "" {
			return m.play(row)
		}
		return m.run(row.command)
	}
	return m, nil
}

func (m model) play(row item) (model, tea.Cmd) {
	if err := m.playback(row.path); err != nil {
		m.note, m.kind = firstText(err.Error()), "error"
		return m, nil
	}
	m.note, m.kind = row.label, "playing"
	return m, nil
}

func (m model) run(command string) (tea.Model, tea.Cmd) {
	m.seq++
	m.working = command
	return m, send(m.seq, command, m.do)
}

// quitting is true from save and close until the widget answers. No key can
// run another command meanwhile, so working stays quit.
func (m model) quitting() bool { return m.working == "quit" }

func cycled(cursor, delta, n int) int {
	if n <= 0 {
		return 0
	}
	cursor = min(max(cursor, 0), n-1)
	return (cursor + delta%n + n) % n
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
		m.reloadRecordings()
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
	defer m.reloadRecordings()
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
