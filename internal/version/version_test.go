package version

import "testing"

func TestStringAddsTheCommit(t *testing.T) {
	old := Commit
	Commit = "abc1234"
	t.Cleanup(func() { Commit = old })
	if got := String(); got != "0.1.0+abc1234" {
		t.Fatalf("%s", got)
	}
}

func TestStringOmitsAnEmptyCommit(t *testing.T) {
	old := Commit
	Commit = ""
	t.Cleanup(func() { Commit = old })
	if got := String(); got != Version {
		t.Fatalf("%s", got)
	}
}
