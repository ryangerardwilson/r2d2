package screen

import "testing"

func TestCommandKeepsTheScreenArguments(t *testing.T) {
	got := Command("o", []string{"--resume", "abc"})
	if len(got) != 4 || got[0] != "tui" || got[1] != "o" || got[2] != "--resume" || got[3] != "abc" {
		t.Fatalf("%q", got)
	}
}

func TestWebIsAScreenAndNlpIsNot(t *testing.T) {
	if !Known("web") || !Known("looptab") || Known("nlp") {
		t.Fatal("screen list")
	}
}
