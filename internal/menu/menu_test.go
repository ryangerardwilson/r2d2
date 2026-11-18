package menu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuMarksTheFirstScreen(t *testing.T) {
	text := model{}.View()
	if !strings.Contains(text, "> web") || !strings.Contains(text, "cyclotree") || !strings.Contains(text, "looptab") || !strings.Contains(text, "slack") {
		t.Fatalf("%s", text)
	}
	if strings.Contains(text, "> cyclotree") {
		t.Fatalf("cursor started on the second row\n%s", text)
	}
}

func TestMenuMovesDownAndStops(t *testing.T) {
	var m tea.Model = model{}
	m, _ = m.Update(key("j"))
	text := m.View()
	if !strings.Contains(text, "> cyclotree") {
		t.Fatalf("%s", text)
	}
	m, _ = m.Update(key("k"))
	if !strings.Contains(m.View(), "> web") {
		t.Fatalf("%s", m.View())
	}
	m, _ = m.Update(key("k"))
	if !strings.Contains(m.View(), "> web") {
		t.Fatalf("moved above the first row\n%s", m.View())
	}
}

func TestMenuHidesShortcutsUntilQuestionMark(t *testing.T) {
	text := model{}.View()
	if strings.Contains(text, "j/k") || strings.Contains(text, "quit") {
		t.Fatalf("shortcuts on the menu:\n%s", text)
	}
	var m tea.Model = model{}
	m, _ = m.Update(key("?"))
	shown := m.View()
	if !strings.Contains(shown, "j/k    move") || !strings.Contains(shown, "q      quit") || strings.Contains(shown, "> web") {
		t.Fatalf("help:\n%s", shown)
	}
	m, _ = m.Update(key("j"))
	m, _ = m.Update(key("?"))
	if !strings.Contains(m.View(), "> web") || strings.Contains(m.View(), "j/k") {
		t.Fatalf("after toggle:\n%s", m.View())
	}
}

func TestExitStatusKeepsACleanReturn(t *testing.T) {
	if exitStatus(nil) != "" {
		t.Fatal(exitStatus(nil))
	}
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
