package health

import (
	"fmt"
	"strings"
)

const (
	bold  = "\x1b[1m"
	dim   = "\x1b[2m"
	reset = "\x1b[0m"
)

const shortcuts = "health\n\nj/k    move\ng/G    first and last\nr      reload\n?      shortcuts\nq      quit\n"

func (m model) View() string {
	if m.help {
		return shortcuts
	}
	var b strings.Builder
	b.WriteString(title(m.rows, m.loading) + "\n\n")
	width := nameWidth(m.rows)
	start, end := m.window()
	for i := start; i < end; i++ {
		mark := "  "
		if i == m.cursor {
			mark = "> "
		}
		b.WriteString(paint(m.rows[i].State, clip(mark+line(m.rows[i], width), m.width)) + "\n")
	}
	return b.String()
}

// window keeps the cursor row on screen when the rows outgrow the terminal.
func (m model) window() (int, int) {
	visible := len(m.rows)
	if m.height > 3 && m.height-3 < visible {
		visible = m.height - 3
	}
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	return start, start + visible
}

func title(rows []Row, loading bool) string {
	if loading && len(rows) == 0 {
		return "health  checking"
	}
	text := fmt.Sprintf("health  %d down", downCount(rows))
	if loading {
		text += "  checking"
	}
	return text
}

func downCount(rows []Row) int {
	n := 0
	for _, row := range rows {
		if row.State == Down {
			n++
		}
	}
	return n
}

func nameWidth(rows []Row) int {
	width := 0
	for _, row := range rows {
		width = max(width, len(row.Name))
	}
	return min(width, 48)
}

func line(row Row, width int) string {
	return fmt.Sprintf("%-*s  %-7s  %s", width, row.Name, row.State, row.Reason)
}

// paint uses bold and dim only, so the screen stays black, white, and gray.
func paint(state State, text string) string {
	switch state {
	case Down:
		return bold + text + reset
	case Waiting:
		return dim + text + reset
	default:
		return text
	}
}

func clip(text string, width int) string {
	runes := []rune(text)
	if width <= 0 || len(runes) <= width {
		return text
	}
	return string(runes[:width])
}

// Plain is the rows without paint, for a pipe.
func Plain(rows []Row) string {
	var b strings.Builder
	b.WriteString(title(rows, false) + "\n\n")
	width := nameWidth(rows)
	for _, row := range rows {
		b.WriteString(strings.TrimRight(line(row, width), " ") + "\n")
	}
	return b.String()
}
