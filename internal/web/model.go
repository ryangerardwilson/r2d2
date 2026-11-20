package web

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

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

type model struct {
	client   *Client
	origin   string
	screen   screen
	mode     inputMode
	cursor   int
	status   string
	busy     bool
	help     bool
	width    int
	height   int
	posts    []Post
	pages    []Page
	home     Home
	rows     []row
	keys     []Key
	sched    Schedule
	prompt   string
	draft    string
	kind     string
	editSlug string
	editKey  string
}

type failedMsg struct{ err string }
type essaysMsg struct {
	posts []Post
	sched Schedule
}
type homeMsg struct {
	home   Home
	status string
}
type pagesMsg struct{ pages []Page }
type keysMsg struct{ keys []Key }
type schedMsg struct{ sched Schedule }
type savedMsg struct{ status string }
type vimDone struct {
	err    error
	path   string
	before string
}

func (m model) Init() tea.Cmd { return nil }

func (m model) loadEssays() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		posts, err := client.ListPosts()
		if err != nil {
			return failedMsg{err.Error()}
		}
		sched, err := client.GetSchedule()
		if err != nil && !missingAction(err) {
			return failedMsg{err.Error()}
		}
		return essaysMsg{posts: orderPosts(posts, time.Now()), sched: sched}
	}
}

func (m model) loadHome() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		home, err := client.GetLanding()
		if err != nil {
			return failedMsg{err.Error()}
		}
		return homeMsg{home: home}
	}
}

func (m model) loadPages() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		pages, err := client.ListPages()
		if err != nil {
			return failedMsg{err.Error()}
		}
		return pagesMsg{pages}
	}
}

func (m model) loadKeys() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		keys, err := client.ListKeys()
		if err != nil {
			return failedMsg{err.Error()}
		}
		return keysMsg{keys}
	}
}

func (m model) saveHome() tea.Cmd {
	client := m.client
	home := m.home
	return func() tea.Msg {
		saved, err := client.PutLanding(home)
		if err != nil {
			return failedMsg{err.Error()}
		}
		return homeMsg{home: saved, status: "saved"}
	}
}

func (m model) saveEssay(slug, title, body string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		title, body, err := proofOrKeep(client, title, body)
		if err != nil {
			return failedMsg{err.Error()}
		}
		if slug == "" {
			_, err = client.CreatePost(title, body)
		} else {
			_, err = client.PutPost(slug, title, body)
		}
		if err != nil {
			return failedMsg{err.Error()}
		}
		return savedMsg{"saved"}
	}
}

func proofOrKeep(client *Client, title, body string) (string, string, error) {
	nextTitle, nextBody, err := client.Proof(title, body)
	if missingAction(err) {
		return title, body, nil
	}
	if err != nil {
		return "", "", err
	}
	return nextTitle, nextBody, nil
}

func missingAction(err error) bool {
	var api *APIError
	return errors.As(err, &api) && (api.Status == http.StatusNotFound || api.Status == http.StatusMethodNotAllowed)
}

func (m model) openVim(text string) tea.Cmd {
	file, err := os.CreateTemp("", "r2d2-web-*.txt")
	if err != nil {
		return func() tea.Msg { return failedMsg{err.Error()} }
	}
	path := file.Name()
	if _, err := file.WriteString(text); err != nil {
		file.Close()
		os.Remove(path)
		return func() tea.Msg { return failedMsg{err.Error()} }
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

func execCommand(name, arg string) *exec.Cmd {
	return exec.Command(name, arg)
}

func (m *model) noteSaved(msg savedMsg) tea.Cmd {
	m.busy = false
	m.status = msg.status
	m.mode = modeList
	switch m.screen {
	case scrEssays:
		return m.loadEssays()
	case scrLanding:
		return m.loadHome()
	case scrPages:
		return m.loadPages()
	case scrKeys:
		return m.loadKeys()
	default:
		return nil
	}
}

func (m *model) fail(msg failedMsg) {
	m.busy = false
	m.status = msg.err
	m.mode = modeList
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

func (m model) listCount() int {
	switch m.screen {
	case scrMenu:
		return menuCount()
	case scrEssays:
		return len(m.posts)
	case scrLanding:
		return len(m.rows)
	case scrPages:
		return len(m.pages)
	case scrKeys:
		return len(m.keys)
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
