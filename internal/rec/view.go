package rec

import (
	"fmt"
	"strings"
)

// Paint is bold, dim, and reverse only, so the screen stays black, white, and gray.
const (
	bold    = "\x1b[1m"
	dim     = "\x1b[2m"
	reverse = "\x1b[7m"
	reset   = "\x1b[0m"
)

var working = map[string]string{
	"open": "opening the camera", "start": "starting", "pause": "pausing", "resume": "resuming",
	"stop": "saving", "focus": "moving the camera", "toggle": "starting or saving", "quit": "saving and closing",
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(bold + "rec" + reset + "\n\n")
	fmt.Fprintf(&b, "status  %s\n", paintWord(m.word, m.clock))
	fmt.Fprintf(&b, "camera  %s\n", m.camera())
	if m.note != "" {
		label, text := m.kind, m.note
		if label == "error" {
			text = bold + text + reset
		}
		fmt.Fprintf(&b, "%-6s  %s\n", label, text)
	}
	if m.working != "" {
		fmt.Fprintf(&b, "\n%s%s%s\n", dim, working[m.working], reset)
	}
	b.WriteString("\n")
	rows := m.items()
	for i, row := range rows {
		if row.path != "" && (i == 0 || rows[i-1].path == "") {
			fmt.Fprintf(&b, "\n%srecordings%s\n", dim, reset)
		}
		mark := "  "
		label := row.label
		if i == m.cursor {
			mark = "> "
			label = bold + label + reset
		}
		fmt.Fprintf(&b, "%s%s\n", mark, label)
	}
	return b.String()
}

func (m model) camera() string {
	switch {
	case m.word == "closed":
		return dim + "closed" + reset
	case m.focus:
		return "fills the screen"
	default:
		return "in the corner"
	}
}

func paintWord(word, clock string) string {
	text := word
	if clock != "" {
		text += " " + clock
	}
	switch word {
	case "recording":
		return reverse + bold + " " + text + " " + reset
	case "paused", "starting", "stopping", "saving":
		return bold + text + reset
	case "closed":
		return dim + text + reset
	default:
		return text
	}
}
