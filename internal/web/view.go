package web

import (
	"fmt"
	"strings"
	"time"
)

func (m model) View() string {
	if m.help {
		return m.helpView()
	}
	var b strings.Builder
	sched := m.schedView()
	fmt.Fprintf(&b, "web  %s\n", m.origin)
	if line := rule(sched); line != "" && (m.screen == scrEssays || m.screen == scrSchedule) {
		fmt.Fprintf(&b, "%s\n", line)
	}
	b.WriteString("\n")
	switch m.screen {
	case scrMenu:
		m.writeChoices(&b, []string{"Essays", "Landing", "Pages", "API keys"})
	case scrEssays:
		m.writeEssays(&b)
	case scrLanding:
		m.writeRows(&b)
	case scrPages:
		m.writePages(&b)
	case scrKeys:
		m.writeKeys(&b)
	case scrSchedule:
		m.writeChoices(&b, []string{
			"weekday  " + weekdayName(sched.Weekday),
			"time     " + sched.Time,
			"zone     " + sched.Zone,
		})
	}
	if m.mode == modePrompt {
		fmt.Fprintf(&b, "\n%s: %s", m.prompt, m.draft)
	}
	if m.mode == modeConfirm {
		fmt.Fprintf(&b, "\n%s", m.prompt)
	}
	if m.status != "" {
		fmt.Fprintf(&b, "\n%s", m.status)
	}
	if line := m.workLine(); line != "" {
		fmt.Fprintf(&b, "\n%s", line)
	}
	b.WriteString("\n")
	return b.String()
}

func (m model) writeChoices(b *strings.Builder, labels []string) {
	width := m.rowWidth()
	for i, label := range labels {
		mark := "  "
		if i == m.cursor {
			mark = "> "
		}
		fmt.Fprintf(b, "%s%s\n", mark, cut(label, width))
	}
}

func (m model) writeEssays(b *strings.Builder) {
	posts := m.essays()
	if len(posts) == 0 {
		b.WriteString("  no essays\n")
		return
	}
	loc, now := loadZone(m.schedView().Zone), time.Now()
	start, end := m.window(len(posts))
	width := m.rowWidth()
	for i := start; i < end; i++ {
		post := posts[i]
		mark := "  "
		if i == m.cursor {
			mark = "> "
		}
		line := whenLine(post, loc, now) + "  " + post.Title
		fmt.Fprintf(b, "%s%s\n", mark, cut(line, width))
	}
}

func (m model) writeRows(b *strings.Builder) {
	rows := m.rows()
	if len(rows) == 0 {
		b.WriteString("  loading\n")
		return
	}
	start, end := m.window(len(rows))
	width := m.rowWidth()
	for i := start; i < end; i++ {
		row := rows[i]
		mark := "  "
		if i == m.cursor {
			mark = "> "
		}
		value := row.value
		if value == "" {
			value = "—"
		}
		fmt.Fprintf(b, "%s%-16s  %s\n", mark, cut(row.label, 16), cut(value, width-18))
	}
}

func (m model) writePages(b *strings.Builder) {
	pages := m.pageList()
	if len(pages) == 0 {
		b.WriteString("  no pages\n")
		return
	}
	start, end := m.window(len(pages))
	width := m.rowWidth()
	for i := start; i < end; i++ {
		page := pages[i]
		mark := "  "
		if i == m.cursor {
			mark = "> "
		}
		fmt.Fprintf(b, "%s%-16s  %s\n", mark, cut(page.Slug, 16), cut(page.Title, width-18))
	}
}

func (m model) writeKeys(b *strings.Builder) {
	keys := m.keyList()
	if len(keys) == 0 {
		b.WriteString("  no keys\n")
		return
	}
	start, end := m.window(len(keys))
	for i := start; i < end; i++ {
		key := keys[i]
		mark := "  "
		if i == m.cursor {
			mark = "> "
		}
		fmt.Fprintf(b, "%s%-16s  %s\n", mark, cut(key.Name, 16), key.Prefix)
	}
}

func (m model) window(n int) (int, int) {
	height := m.height - 6
	if height < 3 {
		height = 3
	}
	if n <= height {
		return 0, n
	}
	start := m.cursor - height/2
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > n {
		end = n
		start = end - height
	}
	return start, end
}

func (m model) rowWidth() int {
	if m.width > 4 {
		return m.width - 2
	}
	return 78
}

func (m model) helpView() string {
	lines := []string{"web", ""}
	lines = append(lines, m.shortcutLines()...)
	lines = append(lines, "?      shortcuts")
	return strings.Join(lines, "\n") + "\n"
}

func (m model) shortcutLines() []string {
	switch m.screen {
	case scrMenu:
		return []string{"j/k    move", "enter  open", "v      view site", "q      quit"}
	case scrEssays:
		return []string{"enter  edit in vim", "n      new", "u/d    move the queue", "x      delete", "s      schedule", "q      back"}
	case scrLanding:
		return []string{"enter  edit", "p      portrait", "c      cover", "q      back"}
	case scrPages:
		return []string{"enter  edit in vim", "n      new", "x      delete", "q      back"}
	case scrKeys:
		return []string{"n      new key", "x      revoke", "q      back"}
	case scrSchedule:
		return []string{"enter  edit the field", "q      back"}
	default:
		return []string{"q      back"}
	}
}
