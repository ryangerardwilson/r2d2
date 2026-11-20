package health

import tea "github.com/charmbracelet/bubbletea"

type model struct {
	rows    []Row
	cursor  int
	help    bool
	loading bool
	gen     int
	width   int
	height  int
	check   func() []Row
}

type loaded struct {
	gen  int
	rows []Row
}

func newModel(check func() []Row) model {
	return model{check: check, loading: true}
}

func (m model) Init() tea.Cmd { return m.load() }

// load checks once. Only the newest load is shown.
func (m model) load() tea.Cmd {
	gen, check := m.gen, m.check
	return func() tea.Msg { return loaded{gen, check()} }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.key(msg.String())
	case loaded:
		if msg.gen == m.gen {
			m.rows, m.loading = msg.rows, false
			m.cursor = min(m.cursor, max(len(m.rows)-1, 0))
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	default:
		return m, nil
	}
}

func (m model) key(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.help = !m.help
		return m, nil
	case "esc":
		if m.help {
			m.help = false
			return m, nil
		}
		return m, tea.Quit
	}
	if m.help {
		return m, nil
	}
	switch key {
	case "r":
		m.gen++
		m.loading = true
		return m, m.load()
	case "j", "down":
		m.cursor = cycled(m.cursor, 1, len(m.rows))
	case "k", "up":
		m.cursor = cycled(m.cursor, -1, len(m.rows))
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = max(len(m.rows)-1, 0)
	}
	return m, nil
}

func cycled(cursor, delta, n int) int {
	if n <= 0 {
		return 0
	}
	cursor = min(max(cursor, 0), n-1)
	return (cursor + delta%n + n) % n
}
