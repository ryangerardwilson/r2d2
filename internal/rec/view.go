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

const shortcuts = `rec

1      start
2      pause
3      resume
4      stop and save, camera stays open
5      camera in the corner or filling the screen
space  start or stop
6      save and close the camera and this screen
q      save and close the camera and this screen
r      reload
?      shortcuts
ctrl+c leave this screen, camera as it is
`

var working = map[string]string{
	"open": "opening the camera", "start": "starting", "pause": "pausing", "resume": "resuming",
	"stop": "saving", "focus": "moving the camera", "toggle": "starting or saving", "quit": "saving and closing",
}

func (m model) View() string {
	if m.help {
		return shortcuts
	}
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
