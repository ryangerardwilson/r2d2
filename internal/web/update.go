package web

import (
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.busy {
			return m, nil
		}
		return m.onKey(msg)
	case essaysMsg:
		m.busy = false
		m.posts = msg.posts
		if msg.sched.Zone != "" {
			m.sched = msg.sched
		}
		m.clamp(len(m.posts))
		return m, nil
	case homeMsg:
		m.busy = false
		m.home = msg.home
		m.rows = landingRows(msg.home)
		m.clamp(len(m.rows))
		if msg.status != "" {
			m.status = msg.status
		}
		return m, nil
	case pagesMsg:
		m.busy = false
		m.pages = msg.pages
		m.clamp(len(m.pages))
		return m, nil
	case keysMsg:
		m.busy = false
		m.keys = msg.keys
		m.clamp(len(m.keys))
		return m, nil
	case schedMsg:
		m.busy = false
		m.sched = msg.sched
		m.status = "schedule saved"
		return m, nil
	case savedMsg:
		return m, m.noteSaved(msg)
	case failedMsg:
		m.fail(msg)
		return m, nil
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
			if m.screen == scrMenu {
				return m, tea.Quit
			}
			m.screen = scrMenu
			m.cursor = 0
			m.status = ""
		}
		return m, nil
	}
	switch msg.String() {
	case "?":
		m.help = true
		return m, nil
	case "q":
		if m.screen == scrMenu {
			return m, tea.Quit
		}
		m.screen = scrMenu
		m.cursor = 0
		m.status = ""
		return m, nil
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

func (m model) open() (tea.Model, tea.Cmd) {
	switch m.screen {
	case scrMenu:
		return m.openMenu()
	case scrEssays:
		if len(m.posts) == 0 {
			return m, nil
		}
		post := m.posts[m.cursor]
		m.kind = "essay"
		m.editSlug = post.Slug
		m.busy = true
		return m, m.openVim(sheet(post.Title, post.Body))
	case scrLanding:
		return m.openField()
	case scrPages:
		if len(m.pages) == 0 {
			return m, nil
		}
		page := m.pages[m.cursor]
		m.kind = "page"
		m.editSlug = page.Slug
		m.busy = true
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
	switch choice {
	case 0:
		m.screen = scrEssays
		m.busy = true
		return m, m.loadEssays()
	case 1:
		m.screen = scrLanding
		m.busy = true
		return m, m.loadHome()
	case 2:
		m.screen = scrPages
		m.busy = true
		return m, m.loadPages()
	default:
		m.screen = scrKeys
		m.busy = true
		return m, m.loadKeys()
	}
}

func (m model) openField() (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	row := m.rows[m.cursor]
	if row.key == "portrait" || row.key == "cover" {
		m.status = "p sets the portrait, c sets the cover, from a file path"
		return m, nil
	}
	m.editKey = row.key
	if row.long {
		m.kind = "field"
		m.busy = true
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
		m.busy = true
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

func (m model) askDelete() (tea.Model, tea.Cmd) {
	switch m.screen {
	case scrEssays:
		if len(m.posts) == 0 {
			return m, nil
		}
		m.mode = modeConfirm
		m.kind = "essay"
		m.prompt = "delete " + m.posts[m.cursor].Title + "? y/n"
		return m, nil
	case scrPages:
		if len(m.pages) == 0 {
			return m, nil
		}
		if stays(m.pages[m.cursor].Slug) {
			m.status = "that page stays"
			return m, nil
		}
		m.mode = modeConfirm
		m.kind = "page"
		m.prompt = "delete " + m.pages[m.cursor].Slug + "? y/n"
		return m, nil
	case scrKeys:
		if len(m.keys) == 0 {
			return m, nil
		}
		m.mode = modeConfirm
		m.kind = "key"
		m.prompt = "revoke " + m.keys[m.cursor].Name + "? y/n"
		return m, nil
	default:
		return m, nil
	}
}

func (m model) onConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = modeList
	if msg.String() != "y" {
		m.status = ""
		return m, nil
	}
	m.busy = true
	client := m.client
	switch m.kind {
	case "essay":
		slug := m.posts[m.cursor].Slug
		return m, func() tea.Msg {
			if err := client.DeletePost(slug); err != nil {
				return failedMsg{err.Error()}
			}
			return savedMsg{"deleted"}
		}
	case "page":
		slug := m.pages[m.cursor].Slug
		return m, func() tea.Msg {
			if err := client.DeletePage(slug); err != nil {
				return failedMsg{err.Error()}
			}
			return savedMsg{"deleted"}
		}
	case "key":
		id := m.keys[m.cursor].ID
		return m, func() tea.Msg {
			if err := client.DeleteKey(id); err != nil {
				return failedMsg{err.Error()}
			}
			return savedMsg{"revoked"}
		}
	default:
		m.busy = false
		return m, nil
	}
}

func (m model) shift(way string) (tea.Model, tea.Cmd) {
	if m.screen != scrEssays || len(m.posts) == 0 {
		return m, nil
	}
	post := m.posts[m.cursor]
	if !futurePost(post, time.Now()) {
		m.status = "a published essay stays where it is"
		return m, nil
	}
	m.busy = true
	client := m.client
	return m, func() tea.Msg {
		if err := client.MovePost(post.Slug, way); err != nil {
			return failedMsg{err.Error()}
		}
		return savedMsg{"moved"}
	}
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
	labels := []string{"weekday", "time", "zone"}
	values := []string{weekdayName(m.sched.Weekday), m.sched.Time, m.sched.Zone}
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
	switch m.kind {
	case "field":
		setField(&m.home, m.editKey, value)
		m.busy = true
		m.status = ""
		return m, m.saveHome()
	case "slug":
		if value == "" {
			m.status = "the slug is empty"
			return m, nil
		}
		m.kind = "page"
		m.editSlug = ""
		m.editKey = value
		m.busy = true
		return m, m.openVim(sheet("Title", ""))
	case "key":
		if value == "" {
			m.status = "the name is empty"
			return m, nil
		}
		m.busy = true
		client := m.client
		return m, func() tea.Msg {
			key, err := client.CreateKey(value)
			if err != nil {
				return failedMsg{err.Error()}
			}
			return savedMsg{"new key " + key.Token}
		}
	case "portrait", "cover":
		return m.upload(m.kind, value)
	case "weekday", "time", "zone":
		return m.saveSchedule(m.kind, value)
	default:
		return m, nil
	}
}

func (m model) upload(name, path string) (tea.Model, tea.Cmd) {
	raw, err := os.ReadFile(path)
	if err != nil {
		m.status = "could not read the photo"
		return m, nil
	}
	m.busy = true
	client := m.client
	return m, func() tea.Msg {
		if err := client.PutMedia(name, raw); err != nil {
			return failedMsg{err.Error()}
		}
		return savedMsg{name + " saved"}
	}
}

func (m model) saveSchedule(field, value string) (tea.Model, tea.Cmd) {
	sched := m.sched
	switch field {
	case "weekday":
		day, ok := parseWeekday(value)
		if !ok {
			m.status = "weekday is Sunday through Saturday, or 0 through 6"
			return m, nil
		}
		sched.Weekday = day
	case "time":
		sched.Time = value
	case "zone":
		sched.Zone = value
	}
	m.busy = true
	client := m.client
	return m, func() tea.Msg {
		saved, err := client.PutSchedule(sched)
		if err != nil {
			return failedMsg{err.Error()}
		}
		return schedMsg{saved}
	}
}

func (m model) afterVim(msg vimDone) (tea.Model, tea.Cmd) {
	m.busy = false
	text, ok := m.finishVim(msg)
	if !ok {
		return m, nil
	}
	switch m.kind {
	case "essay":
		title, body := readSheet(text)
		if title == "" || body == "" {
			m.status = "an essay needs a title and a body"
			return m, nil
		}
		m.busy = true
		return m, m.saveEssay(m.editSlug, title, body)
	case "page":
		title, body := readSheet(text)
		slug := m.editSlug
		if slug == "" {
			slug = m.editKey
		}
		if title == "" {
			m.status = "a page needs a title"
			return m, nil
		}
		m.busy = true
		client := m.client
		page := Page{Slug: slug, Title: title, Body: body}
		return m, func() tea.Msg {
			if _, err := client.PutPage(page); err != nil {
				return failedMsg{err.Error()}
			}
			return savedMsg{"saved"}
		}
	case "field":
		setField(&m.home, m.editKey, oneParagraph(text))
		m.busy = true
		m.status = ""
		return m, m.saveHome()
	default:
		return m, nil
	}
}

func openSite(origin string) tea.Cmd {
	return func() tea.Msg {
		cmd := execCommand("xdg-open", origin)
		if err := cmd.Start(); err != nil {
			return failedMsg{"could not open the site"}
		}
		go func() { _ = cmd.Wait() }()
		return savedMsg{"opened " + origin}
	}
}
