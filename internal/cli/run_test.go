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
	for _, want := range []string{"r2d2 web", "cyclotree", ":Wrap", "nebula crypt", "flowing paragraph"} {
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
