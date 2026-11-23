package web

import (
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Update never waits on the site. A key changes local state and paints; a write or a remote read goes on the
// worker (work.go) as an intent the frame already shows.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.onKey(msg)
	case doneMsg:
		return m.finish(msg)
	case statusMsg:
		m.status = string(msg)
		return m, nil
	case quitMsg:
		return m, tea.Quit
	case vimDone:
		return m.afterVim(msg)
	default:
		return m, nil
	}
}

func (m model) onKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modePrompt:
		return m.onPrompt(msg)
	case modeConfirm:
		return m.onConfirm(msg)
	default:
		return m.onList(msg)
	}
}

func (m model) onList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.help {
		switch msg.String() {
		case "?", "esc":
			m.help = false
		case "q":
			m.help = false
			return m.back()
		}
		return m, nil
	}
	switch msg.String() {
	case "?":
		m.help = true
		return m, nil
	case "q":
		return m.back()
	case "j", "down":
		m.move(1, m.listCount())
		return m, nil
	case "k", "up":
		m.move(-1, m.listCount())
		return m, nil
	case "enter":
		return m.open()
	case "n":
		return m.create()
	case "x":
		return m.askDelete()
	case "u":
		return m.shift("up")
	case "d":
		return m.shift("down")
	case "s":
		if m.screen == scrEssays {
			m.screen = scrSchedule
			m.cursor = 0
			m.status = ""
		}
		return m, nil
	case "p":
		return m.askPhoto("portrait")
	case "c":
		return m.askPhoto("cover")
	case "v":
		return m, openSite(m.origin)
	default:
		return m, nil
	}
}

// back leaves a screen for the menu, and leaves r2d2 web from the menu.
func (m model) back() (tea.Model, tea.Cmd) {
	if m.screen == scrMenu {
		return m.leave()
	}
	m.screen = scrMenu
	m.cursor = 0
	m.status = ""
	return m, nil
}

func (m model) open() (tea.Model, tea.Cmd) {
	switch m.screen {
	case scrMenu:
		return m.openMenu()
	case scrEssays:
		posts := m.essays()
		if len(posts) == 0 {
			return m, nil
		}
		post := posts[m.cursor]
		m.kind = "essay"
		m.editSlug = post.Slug
		return m, m.openVim(sheet(post.Title, post.Body))
	case scrLanding:
		return m.openField()
	case scrPages:
		pages := m.pageList()
		if len(pages) == 0 {
			return m, nil
		}
		page := pages[m.cursor]
		m.kind = "page"
		m.editSlug = page.Slug
		return m, m.openVim(sheet(page.Title, page.Body))
	case scrSchedule:
		return m.askSchedule()
	default:
		return m, nil
	}
}

func (m model) openMenu() (tea.Model, tea.Cmd) {
	choice := m.cursor
	m.cursor = 0
	m.status = ""
	var load intent
	switch choice {
	case 0:
		m.screen, load = scrEssays, loadEssays{}
	case 1:
		m.screen, load = scrLanding, loadLanding{}
	case 2:
		m.screen, load = scrPages, loadPages{}
	default:
		m.screen, load = scrKeys, loadKeys{}
	}
	cmd := m.enqueue(load)
	return m, cmd
}

func (m model) openField() (tea.Model, tea.Cmd) {
	rows := m.rows()
	if len(rows) == 0 {
		return m, nil
	}
	row := rows[m.cursor]
	if row.key == "portrait" || row.key == "cover" {
		m.status = "p sets the portrait, c sets the cover, from a file path"
		return m, nil
	}
	m.editKey = row.key
	if row.long {
		m.kind = "field"
		return m, m.openVim(row.value + "\n")
	}
	m.mode = modePrompt
	m.kind = "field"
	m.prompt = row.label
	m.draft = row.value
	return m, nil
}

func (m model) create() (tea.Model, tea.Cmd) {
	switch m.screen {
	case scrEssays:
		m.kind = "essay"
		m.editSlug = ""
		return m, m.openVim(sheet("Title", ""))
	case scrPages:
		m.mode = modePrompt
		m.kind = "slug"
		m.prompt = "page slug"
		m.draft = ""
		return m, nil
	case scrKeys:
		m.mode = modePrompt
		m.kind = "key"
		m.prompt = "key name"
		m.draft = ""
		return m, nil
	default:
		return m, nil
	}
}

// askDelete picks the row under the cursor now, so the answer to an earlier job cannot change what y deletes.
func (m model) askDelete() (tea.Model, tea.Cmd) {
	switch m.screen {
	case scrEssays:
		posts := m.essays()
		if len(posts) == 0 {
			return m, nil
		}
		m.mode, m.kind, m.editSlug = modeConfirm, "essay", posts[m.cursor].Slug
		m.prompt = "delete " + posts[m.cursor].Title + "? y/n"
		return m, nil
	case scrPages:
		pages := m.pageList()
		if len(pages) == 0 {
			return m, nil
		}
		if stays(pages[m.cursor].Slug) {
			m.status = "that page stays"
			return m, nil
		}
		m.mode, m.kind, m.editSlug = modeConfirm, "page", pages[m.cursor].Slug
		m.prompt = "delete " + pages[m.cursor].Slug + "? y/n"
		return m, nil
	case scrKeys:
		keys := m.keyList()
		if len(keys) == 0 {
			return m, nil
		}
		m.mode, m.kind, m.editID = modeConfirm, "key", keys[m.cursor].ID
		m.prompt = "revoke " + keys[m.cursor].Name + "? y/n"
		return m, nil
	default:
		return m, nil
	}
}

func (m model) onConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = modeList
	m.status = ""
	if msg.String() != "y" {
		return m, nil
	}
	var drop intent
	switch m.kind {
	case "essay":
		drop = dropEssay{slug: m.editSlug}
	case "page":
		drop = dropPage{slug: m.editSlug}
	case "key":
		drop = dropKey{id: m.editID}
	default:
		return m, nil
	}
	cmd := m.enqueue(drop)
	m.clamp(m.listCount())
	return m, cmd
}

// shift moves a future essay at once in the view, and the cursor with it, and asks the site to do the same.
func (m model) shift(way string) (tea.Model, tea.Cmd) {
	if m.screen != scrEssays {
		return m, nil
	}
	posts := m.essays()
	if len(posts) == 0 {
		return m, nil
	}
	now := time.Now()
	if !futurePost(posts[m.cursor], now) {
		m.status = "a published essay stays where it is"
		return m, nil
	}
	to := neighbour(posts, m.cursor, way, now)
	if to < 0 {
		m.status = "that essay cannot move that way"
		return m, nil
	}
	m.status = ""
	cmd := m.enqueue(moveEssay{slug: posts[m.cursor].Slug, way: way})
	m.cursor = to
	return m, cmd
}

func (m model) askPhoto(name string) (tea.Model, tea.Cmd) {
	if m.screen != scrLanding {
		return m, nil
	}
	m.mode = modePrompt
	m.kind = name
	m.prompt = name + " file"
	m.draft = ""
	return m, nil
}

func (m model) askSchedule() (tea.Model, tea.Cmd) {
	sched := m.schedView()
	labels := []string{"weekday", "time", "zone"}
	values := []string{weekdayName(sched.Weekday), sched.Time, sched.Zone}
	m.mode = modePrompt
	m.kind = labels[m.cursor]
	m.prompt = labels[m.cursor]
	m.draft = values[m.cursor]
	return m, nil
}

func (m model) onPrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.draft = ""
		return m, nil
	case "enter":
		return m.submitPrompt()
	case "backspace":
		runes := []rune(m.draft)
		if len(runes) > 0 {
			m.draft = string(runes[:len(runes)-1])
		}
		return m, nil
	default:
		if len([]rune(msg.String())) == 1 {
			m.draft += msg.String()
		}
		return m, nil
	}
}

func (m model) submitPrompt() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.draft)
	m.mode = modeList
	m.status = ""
	var cmd tea.Cmd
	switch m.kind {
	case "field":
		cmd = m.enqueue(editField{key: m.editKey, value: value})
	case "slug":
		if value == "" {
			m.status = "the slug is empty"
			return m, nil
		}
		m.kind = "page"
		m.editSlug = value
		return m, m.openVim(sheet("Title", ""))
	case "key":
		if value == "" {
			m.status = "the name is empty"
			return m, nil
		}
		cmd = m.enqueue(newKey{name: value})
	case "portrait", "cover":
		cmd = m.enqueue(setPhoto{name: m.kind, path: value})
	case "weekday", "time", "zone":
		return m.saveSchedule(m.kind, value)
	}
	return m, cmd
}

func (m model) saveSchedule(field, value string) (tea.Model, tea.Cmd) {
	in := editSchedule{field: field, value: value}
	if field == "weekday" {
		day, ok := parseWeekday(value)
		if !ok {
			m.status = "weekday is Sunday through Saturday, or 0 through 6"
			return m, nil
		}
		in.day = day
	}
	cmd := m.enqueue(in)
	return m, cmd
}

// afterVim reads the vim buffer and puts the save on the worker.
func (m model) afterVim(msg vimDone) (tea.Model, tea.Cmd) {
	text, ok := m.finishVim(msg)
	if !ok {
		return m, nil
	}
	var cmd tea.Cmd
	switch m.kind {
	case "essay":
		title, body := readSheet(text)
		if title == "" || body == "" {
			m.status = "an essay needs a title and a body"
			return m, nil
		}
		m.status = ""
		cmd = m.enqueue(saveEssay{slug: m.editSlug, title: title, body: body})
	case "page":
		title, body := readSheet(text)
		if title == "" {
			m.status = "a page needs a title"
			return m, nil
		}
		m.status = ""
		cmd = m.enqueue(savePage{slug: m.editSlug, title: title, body: body})
	case "field":
		m.status = ""
		cmd = m.enqueue(editField{key: m.editKey, value: oneParagraph(text)})
	}
	return m, cmd
}

func openSite(origin string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("xdg-open", origin)
		if err := cmd.Start(); err != nil {
			return statusMsg("could not open the site")
		}
		go func() { _ = cmd.Wait() }()
		return statusMsg("opened " + origin)
	}
}
