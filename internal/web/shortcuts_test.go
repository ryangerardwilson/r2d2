package web

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWebHidesShortcutsUntilQuestionMark(t *testing.T) {
	start := model{origin: "https://example.test", screen: scrEssays}.View()
	if strings.Contains(start, "j/k") || strings.Contains(start, "delete") {
		t.Fatalf("shortcuts on the screen:\n%s", start)
	}
	var m tea.Model = model{origin: "https://example.test", screen: scrEssays}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	shown := m.View()
	if !strings.Contains(shown, "move the queue") || !strings.Contains(shown, "?      shortcuts") || strings.Contains(shown, "https://example.test") {
		t.Fatalf("help:\n%s", shown)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	back := m.View()
	if strings.Contains(back, "move the queue") || !strings.Contains(back, "https://example.test") {
		t.Fatalf("after toggle:\n%s", back)
	}
	got := m.(model)
	if got.cursor != 0 || got.help {
		t.Fatalf("cursor %d help %v", got.cursor, got.help)
	}
}

func TestEssayListWraps(t *testing.T) {
	m := model{
		origin: "https://example.test",
		screen: scrEssays,
		width:  80,
		height: 24,
		posts:  []Post{{Title: "First essay"}, {Title: "Second essay"}},
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	got := next.(model)
	if got.cursor != 1 || !strings.Contains(markedLine(got.View()), "Second essay") {
		t.Fatalf("k %d\n%s", got.cursor, got.View())
	}
	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	got = next.(model)
	if got.cursor != 0 || !strings.Contains(markedLine(got.View()), "First essay") {
		t.Fatalf("j %d\n%s", got.cursor, got.View())
	}
	empty := model{origin: "https://example.test", screen: scrEssays}
	next, _ = empty.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if next.(model).cursor != 0 {
		t.Fatalf("empty %d", next.(model).cursor)
	}
}

func markedLine(view string) string {
	for _, line := range strings.Split(view, "\n") {
		if strings.HasPrefix(line, "> ") {
			return line
		}
	}
	return ""
}
