package web

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	scrMenu screen = iota
	scrEssays
	scrLanding
	scrPages
	scrKeys
	scrSchedule
)

type inputMode int

const (
	modeList inputMode = iota
	modePrompt
	modeConfirm
)

// site is the site's last answer for each list. A field is replaced wholesale by the next answer that
// carries it and is never edited in place; the screen draws it with the pending intents applied.
type site struct {
	posts   []Post // the queue first, soonest first, then the essays already out
	sched   Schedule
	landing *Landing // nil until the site answers
	pages   []Page
	keys    []Key
}

type model struct {
	client   *Client // nil until the API key is read
	origin   string
	readKey  func() (string, error) // reads the admin API key, on the worker
	site     site
	work     []job // the worker's queue; work[0] is running when running is true
	running  bool
	lastID   int
	quitting bool
	screen   screen
	mode     inputMode
	cursor   int
	status   string
	help     bool
	width    int
	height   int
	prompt   string
	draft    string
	kind     string
	editSlug string
	editKey  string
	editID   int64
}

type statusMsg string
type vimDone struct {
	err    error
	path   string
	before string
}

// newModel paints at once. With no token it starts the worker on the API key read, so the first frame never
// waits on op.
func newModel(origin, token string, read func() (string, error)) model {
	m := model{origin: origin, readKey: read, width: 80, height: 24}
	if token = strings.TrimSpace(token); token != "" {
		m.client = &Client{Origin: origin, Token: token}
		return m
	}
	m.push(readKey{read: read, wait: keyWait})
	m.running = true
	return m
}

func (m model) Init() tea.Cmd {
	if !m.running || len(m.work) == 0 {
		return nil
	}
	return m.command(m.work[0])
}

func (m model) openVim(text string) tea.Cmd {
	file, err := os.CreateTemp("", "r2d2-web-*.txt")
	if err != nil {
		return func() tea.Msg { return statusMsg(err.Error()) }
	}
	path := file.Name()
	if _, err := file.WriteString(text); err != nil {
		file.Close()
		os.Remove(path)
		return func() tea.Msg { return statusMsg(err.Error()) }
	}
	file.Close()
	cmd := exec.Command("vim", "-c", "Wrap", path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return vimDone{err: err, path: path, before: text}
	})
}

func (m *model) finishVim(msg vimDone) (string, bool) {
	raw, readErr := os.ReadFile(msg.path)
	os.Remove(msg.path)
	if msg.err != nil {
		m.status = "vim did not save"
		return "", false
	}
	if readErr != nil {
		m.status = readErr.Error()
		return "", false
	}
	text := string(raw)
	if text == msg.before {
		m.status = "left unchanged"
		return "", false
	}
	return text, true
}

func (m *model) clamp(n int) {
	if n <= 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
}

func (m *model) move(delta, n int) {
	if n == 0 {
		return
	}
	m.cursor = cycled(m.cursor, delta, n)
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

func stays(slug string) bool {
	switch slug {
	case "about", "privacy", "fulfillment":
		return true
	default:
		return false
	}
}

func menuCount() int { return 4 }

// rows are the landing fields the screen draws: the site's last landing page with the pending edits.
func (m model) rows() []row {
	landing := m.landingView()
	if landing == nil {
		return nil
	}
	return landingRows(*landing)
}

func (m model) listCount() int {
	switch m.screen {
	case scrMenu:
		return menuCount()
	case scrEssays:
		return len(m.essays())
	case scrLanding:
		return len(m.rows())
	case scrPages:
		return len(m.pageList())
	case scrKeys:
		return len(m.keyList())
	case scrSchedule:
		return 3
	default:
		return 0
	}
}

func cut(text string, width int) string {
	if width < 4 {
		return ""
	}
	text = strings.ReplaceAll(text, "\n", " ")
	if len([]rune(text)) <= width {
		return text
	}
	runes := []rune(text)
	return string(runes[:width-1]) + "…"
}

func rule(sched Schedule) string {
	if sched.Zone == "" {
		return ""
	}
	return fmt.Sprintf("publishes %s at %s, %s", weekdayName(sched.Weekday), sched.Time, sched.Zone)
}
