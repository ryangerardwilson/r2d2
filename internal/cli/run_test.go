package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpNamesWebAndWrap(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2"}, &out, &err); code != 0 {
		t.Fatalf("code %d", code)
	}
	for _, want := range []string{"r2d2 web", "r2d2 slack", "cyclotree", ":Wrap", "nebula crypt", "flowing paragraph"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q\n%s", want, out.String())
		}
	}
}

func TestUnknownScreenExits2(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2", "nlp"}, &out, &err); code != 2 || !strings.Contains(err.String(), "unknown screen") {
		t.Fatalf("code %d\n%s", code, err.String())
	}
}

func TestWebHelpWithoutATerminal(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2", "web", "help"}, &out, &err); code != 0 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(out.String(), ":Wrap") || !strings.Contains(out.String(), "RGW_API_KEY") {
		t.Fatalf("%s", out.String())
	}
}

func TestVersion(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2", "version"}, &out, &err); code != 0 || !strings.HasPrefix(out.String(), "r2d2 ") {
		t.Fatalf("%d %q", code, out.String())
	}
}

func TestHelpNamesHealthAndMusica(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2", "help"}, &out, &err); code != 0 {
		t.Fatalf("code %d", code)
	}
	for _, want := range []string{"r2d2 health", "r2d2 musica", "background jobs and nebula backups", "player, cassettes, and downloads"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q\n%s", want, out.String())
		}
	}
}

func TestHealthHelpWithoutATerminal(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2", "health", "help"}, &out, &err); code != 0 || !strings.Contains(out.String(), "r2d2 health") || !strings.Contains(out.String(), "never restarts") {
		t.Fatalf("code %d\n%s", code, out.String())
	}
}

func TestMusicaHelpIsThePlayerHelp(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2", "musica", "help"}, &out, &err); code != 0 || !strings.Contains(out.String(), "musica terminal music player") || !strings.Contains(out.String(), "r2d2 musica") {
		t.Fatalf("code %d\n%s%s", code, out.String(), err.String())
	}
}

func TestHelpNamesRec(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2", "help"}, &out, &err); code != 0 {
		t.Fatalf("code %d", code)
	}
	for _, want := range []string{"r2d2 rec", "rec          screen recorder and camera"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q\n%s", want, out.String())
		}
	}
}

func TestRecHelpWithoutATerminal(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2", "rec", "help"}, &out, &err); code != 0 || !strings.Contains(out.String(), "r2d2 rec") || !strings.Contains(out.String(), "rec is not open") {
		t.Fatalf("code %d\n%s", code, out.String())
	}
}

func TestLooptabRunsInR2d2(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2", "looptab", "help"}, &out, &err); code != 0 || !strings.Contains(out.String(), "e edits the jobs file in vim") || !strings.Contains(out.String(), "enter runs the selected job now") {
		t.Fatalf("looptab %d\n%s%s", code, out.String(), err.String())
	}
	if !strings.Contains(Help, "run one now or edit the jobs file") || !strings.Contains(Help, "run a Looptab job now, or edit the jobs file in vim") {
		t.Fatalf("help\n%s", Help)
	}
}

func TestSkillsIsListedAndExplained(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"r2d2", "help"}, &out, &err); code != 0 {
		t.Fatalf("code %d", code)
	}
	_, after, _ := strings.Cut(out.String(), "screens:\n")
	if !strings.Contains(after, "skills       shared skills for Claude Code, Grok Build, and Codex") {
		t.Fatalf("screens\n%s", after)
	}
	out.Reset()
	if code := Run([]string{"r2d2", "skills", "help"}, &out, &err); code != 0 || !strings.Contains(out.String(), "~/.local/share/r2d2/skills") {
		t.Fatalf("skills help %d\n%s%s", code, out.String(), err.String())
	}
	out.Reset()
	err.Reset()
	if code := Run([]string{"r2d2", "skils"}, &out, &err); code != 2 || !strings.Contains(err.String(), "unknown screen") {
		t.Fatalf("unknown screen %d\n%s", code, err.String())
	}
}
