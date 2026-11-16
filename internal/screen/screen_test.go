package screen

import (
	"bytes"
	"testing"
)

func TestLaunchPassesTheScreenArguments(t *testing.T) {
	var out, errb bytes.Buffer
	code := Launch("o", []string{"help"}, &out, &errb)
	text := out.String() + errb.String()
	if code != 0 || !bytes.Contains([]byte(text), []byte("r2d2 o")) {
		t.Fatalf("code %d\n%s", code, text)
	}
	code = Launch("nlp", nil, &out, &errb)
	if code != 2 {
		t.Fatalf("nlp %d", code)
	}
}

func TestWebIsAScreenAndNlpIsNot(t *testing.T) {
	if !Known("web") || !Known("looptab") || Known("nlp") {
		t.Fatal("screen list")
	}
}
