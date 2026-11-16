package web

import "testing"

func TestReadSheetJoinsWrappedLines(t *testing.T) {
	text := "A quiet\nending\n\nThe chapter closes\non the last page.\n\nThen the market.\n"
	title, body := readSheet(text)
	if title != "A quiet ending" {
		t.Fatalf("title %q", title)
	}
	if body != "The chapter closes on the last page.\n\nThen the market." {
		t.Fatalf("body %q", body)
	}
}

func TestSheetRoundTrip(t *testing.T) {
	in := sheet("One line title", "First paragraph.\n\nSecond paragraph.")
	title, body := readSheet(in)
	if title != "One line title" || body != "First paragraph.\n\nSecond paragraph." {
		t.Fatalf("title %q body %q", title, body)
	}
}

func TestOneParagraphDropsTheWrap(t *testing.T) {
	got := oneParagraph("The hero\nstays one breath.\n")
	if got != "The hero stays one breath." {
		t.Fatalf("got %q", got)
	}
}

func TestBlankLinesAreParagraphBreaks(t *testing.T) {
	parts := paragraphs("one\n\n\ntwo\n")
	if len(parts) != 2 || parts[0] != "one" || parts[1] != "two" {
		t.Fatalf("%#v", parts)
	}
}
