package looptab

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	styleTitle  = "\x1b[1m"
	styleActive = "\x1b[1;97m"
	styleMuted  = "\x1b[90m"
	styleError  = "\x1b[7m"
	styleReset  = "\x1b[0m"
)

// screen is the terminal the list draws on and reads keys from.
type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

// model is the job list. rows is a cache of the last looptab now answer and
// is replaced whole; the cursor, the status, and selected are local.
type model struct {
	rows    []job
	failed  error // the last answer's error; rows is empty then
	pending int   // reloads asked of the worker and not answered yet
	cursor  int
	status  string
	alarm   bool // status is an error
	help    bool
	width   int
	height  int
	// selected is the id of the job under the cursor. It keeps the cursor
	// on that job across reloads, and the session place follows it. A
	// resumed screen starts it at the job the session names.
	selected string
	load     func(context.Context) ([]job, error)
	run      func(job) error
	edit     func() (int, error)
	session  placer
	timeout  time.Duration // bound on one looptab now; zero is reloadTimeout
	saveWait time.Duration // bound on the last place write; zero is placeWait
	reloads  *reloader
	places   *placeWriter
}

type pressed struct {
	key string
	err error
}

// loop paints, then waits for a key or a worker answer. A key is read only
// after the previous one is handled, so a terminal hand-off on Enter or e
// never races a read for the terminal.
func (m *model) loop(term screen) error {
	m.start()
	defer m.stop(term)
	keys := make(chan pressed, 1)
	reading := false
	for {
		m.notePlace()
		m.width, m.height = term.size()
		term.paint(m.view())
		if !reading {
			reading = true
			go func() {
				key, err := term.readKey()
				keys <- pressed{key, err}
			}()
		}
		select {
		case p := <-keys:
			reading = false
			if p.err != nil {
				return p.err
			}
			if m.handleKey(p.key) {
				return nil
			}
		case a := <-m.answers():
			m.loaded(a)
		}
	}
}

func (m *model) start() {
	if m.session != nil {
		m.places = startPlaceWriter(m.session, m.selected)
	}
	if m.load != nil {
		m.reloads = startReloader(m.load, orDefault(m.timeout, reloadTimeout))
		m.reload()
	}
}

// stop drops a reload in flight and drains the session place. A write still
// out paints a saving frame and gets saveWait, no more.
func (m *model) stop(term screen) {
	if m.reloads != nil {
		m.reloads.stop()
	}
	if m.places == nil {
		return
	}
	m.places.finish()
	if m.places.wait(0) {
		return
	}
	term.paint(paintStyle(styleTitle, "looptab") + "\n\nsaving place\n")
	m.places.wait(orDefault(m.saveWait, placeWait))
}

func (m *model) answers() <-chan answer {
	if m.reloads == nil {
		return nil
	}
	return m.reloads.answers
}

// reload asks the worker for the job list. The frame says loading until the
// answer comes back.
func (m *model) reload() {
	if m.reloads != nil && m.reloads.ask() {
		m.pending++
	}
}

// loaded replaces the rows with the worker's answer. The cursor stays on the
// selected job when it is still listed.
func (m *model) loaded(a answer) {
	m.pending--
	if a.err != nil {
		m.rows, m.cursor, m.failed = nil, 0, a.err
		m.status, m.alarm = a.err.Error(), true
		return
	}
	m.rows, m.failed = a.rows, nil
	m.cursor = m.clamp()
	for i, row := range m.rows {
		if row.ID == m.selected {
			m.cursor = i
		}
	}
}

// notePlace hands the selected job to the place writer when it changed. The
// write happens off the loop.
func (m *model) notePlace() {
	if len(m.rows) == 0 {
		return
	}
	id := m.rows[m.clamp()].ID
	if id == m.selected {
		return
	}
	m.selected = id
	if m.places != nil {
		m.places.note(id)
	}
}

func orDefault(d, fallback time.Duration) time.Duration {
	if d <= 0 {
		return fallback
	}
	return d
}

func (m *model) handleKey(key string) bool {
	if m.help {
		switch key {
		case "?", "esc":
			m.help = false
		case "ctrl+c", "q":
			return true
		}
		return false
	}
	if key == "?" {
		m.help = true
		m.status = ""
		return false
	}
	switch key {
	case "ctrl+c", "q", "esc":
		return true
	case "j", "down":
		m.cursor = cycled(m.cursor, 1, len(m.rows))
	case "k", "up":
		m.cursor = cycled(m.cursor, -1, len(m.rows))
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(m.rows) - 1
	case "enter":
		m.trigger()
	case "e":
		m.editJobs()
	}
	m.cursor = m.clamp()
	return false
}

// trigger runs the selected job, a terminal hand-off, and then asks the
// worker for the list. While a reload is out the rows may be stale, so the
// index is not run.
func (m *model) trigger() {
	if m.pending > 0 {
		m.status, m.alarm = "still loading", false
		return
	}
	if len(m.rows) == 0 {
		if m.failed == nil {
			m.status, m.alarm = "no jobs", false
		}
		return
	}
	if m.run == nil {
		return
	}
	row := m.rows[m.clamp()]
	err := m.run(row)
	m.reload()
	if err != nil {
		m.status, m.alarm = row.ID+" failed", true
		return
	}
	m.status, m.alarm = "ran "+row.ID, false
}

// editJobs opens the jobs file in vim. After vim exits, c3po has installed
// the registered files and the worker reloads the list. Missing vim leaves
// the list up.
func (m *model) editJobs() {
	if m.edit == nil {
		return
	}
	n, err := m.edit()
	if errors.Is(err, errNoVim) {
		m.status, m.alarm = err.Error(), true
		return
	}
	m.reload()
	if err != nil {
		m.status, m.alarm = err.Error(), true
		return
	}
	m.status, m.alarm = fmt.Sprintf("installed %d jobs files", n), false
}

func (m *model) clamp() int {
	if len(m.rows) == 0 || m.cursor < 0 {
		return 0
	}
	if m.cursor >= len(m.rows) {
		return len(m.rows) - 1
	}
	return m.cursor
}

func cycled(cursor, delta, n int) int {
	if n <= 0 {
		return 0
	}
	if cursor < 0 {
		cursor = 0
	} else if cursor >= n {
		cursor = n - 1
	}
	return (cursor + delta%n + n) % n
}

func (m *model) view() string {
	if m.help {
		return m.helpView()
	}
	var b strings.Builder
	title := paintStyle(styleTitle, "looptab")
	if m.pending > 0 {
		title += "  " + paintStyle(styleMuted, "loading")
	}
	fmt.Fprintf(&b, "%s\n\n", title)
	if len(m.rows) == 0 {
		if m.failed == nil && m.pending == 0 {
			b.WriteString("no jobs\n")
		}
	} else {
		start, end := m.window()
		for i := start; i < end; i++ {
			prefix, style := "  ", ""
			if i == m.clamp() {
				prefix, style = "> ", styleActive
			}
			line := truncate(prefix+m.rows[i].label(), m.width)
			b.WriteString(paintStyle(style, line))
			b.WriteByte('\n')
		}
	}
	if m.status != "" {
		style := styleMuted
		if m.alarm {
			style = styleError
		}
		fmt.Fprintf(&b, "\n%s\n", paintStyle(style, m.status))
	}
	return b.String()
}

func (m *model) helpView() string {
	lines := []string{
		paintStyle(styleTitle, "looptab"),
		"",
		"j/k    move",
		"enter  run now",
		"e      edit jobs file",
		"?      shortcuts",
		"q      quit",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *model) window() (int, int) {
	rows := m.height - 3
	if m.height <= 0 || rows < 1 || len(m.rows) <= rows {
		return 0, len(m.rows)
	}
	start := 0
	if m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	end := start + rows
	if end > len(m.rows) {
		end = len(m.rows)
	}
	return start, end
}

func paintStyle(style, text string) string {
	if style == "" {
		return text
	}
	return style + text + styleReset
}

func truncate(value string, width int) string {
	if width <= 4 || len(value) <= width {
		return value
	}
	return value[:width-3] + "..."
}
