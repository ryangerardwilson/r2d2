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
