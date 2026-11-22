package looptab

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	c3polooptab "github.com/ryangerardwilson/c3po/looptab"
)

func TestHelpNamesTheRunAndTheEdit(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run([]string{"help"}, &out, &errb); code != 0 {
		t.Fatalf("help %d\n%s", code, errb.String())
	}
	for _, want := range []string{"r2d2 looptab", "enter runs the selected job now", "e edits the jobs file in vim"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %q\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "j/k") {
		t.Fatalf("help lists shortcuts\n%s", out.String())
	}
	out.Reset()
	if code := Run([]string{"version"}, &out, &errb); code != 0 || !strings.HasPrefix(out.String(), "r2d2 ") {
		t.Fatalf("version %d %s", code, out.String())
	}
	errb.Reset()
	if code := Run([]string{"extra"}, &out, &errb); code != 2 || !strings.Contains(errb.String(), "r2d2 looptab") {
		t.Fatalf("extra %d %s", code, errb.String())
	}
	errb.Reset()
	if code := Run([]string{"--resume"}, &out, &errb); code != 2 || !strings.Contains(errb.String(), "r2d2 looptab") {
		t.Fatalf("resume %d %s", code, errb.String())
	}
}

func TestEditorInstallsAfterVim(t *testing.T) {
	var calls []string
	e := editor{
		lookVim:  func() (string, error) { return "/bin/vim", nil },
		jobsFile: func() (string, error) { calls = append(calls, "jobs"); return "/x/jobs", nil },
		install:  func() (int, error) { calls = append(calls, "install"); return 2, nil },
		open: func(vim, path string) error {
			calls = append(calls, "open "+vim+" "+path)
			return nil
		},
	}
	n, err := e.edit()
	if err != nil || n != 2 || strings.Join(calls, ",") != "jobs,open /bin/vim /x/jobs,install" {
		t.Fatalf("n %d err %v calls %v", n, err, calls)
	}
}

func TestEditorWithoutVimTouchesNothing(t *testing.T) {
	called := false
	e := editor{
		lookVim:  func() (string, error) { return "", errors.New("not found") },
		jobsFile: func() (string, error) { called = true; return "", nil },
		install:  func() (int, error) { called = true; return 0, nil },
		open:     func(string, string) error { called = true; return nil },
	}
	if _, err := e.edit(); !errors.Is(err, errNoVim) || err.Error() != "vim is not on PATH" || called {
		t.Fatalf("err %v called %v", err, called)
	}
}

func TestEditorDoesNotInstallWhenVimFails(t *testing.T) {
	installed := false
	e := editor{
		lookVim:  func() (string, error) { return "vim", nil },
		jobsFile: func() (string, error) { return "/x/jobs", nil },
		install:  func() (int, error) { installed = true; return 0, nil },
		open:     func(string, string) error { return errors.New("exit status 1") },
	}
	if _, err := e.edit(); err == nil || installed {
		t.Fatalf("err %v installed %v", err, installed)
	}
}

// The screen's edit creates the c3po template, registers it, and installs it
// through c3po, which writes the scheduler file.
func TestEditWithNothingRegisteredUsesTheC3poTemplate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("C3PO_SQLX", "off")
	e := editor{
		lookVim:  func() (string, error) { return "vim", nil },
		jobsFile: c3polooptab.JobsFile,
		install:  c3polooptab.Install,
		open: func(_, path string) error {
			body, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(body), "# Schedules:") {
				t.Fatalf("template %s %v", body, err)
			}
			return os.WriteFile(path, []byte("hourly c3po gmail tidy\n"), 0o644)
		},
	}
	n, err := e.edit()
	if err != nil || n != 1 {
		t.Fatalf("n %d err %v", n, err)
	}
	jobs := filepath.Join(home, ".config", "c3po", "looptab", "jobs")
	if _, err := os.Stat(jobs); err != nil {
		t.Fatal(err)
	}
	sched, err := os.ReadFile(filepath.Join(home, ".config", "looptab", "looptab"))
	if err != nil || !strings.Contains(string(sched), "hourly c3po gmail tidy") || !strings.Contains(string(sched), "Installed by c3po looptab") {
		t.Fatalf("schedule %s %v", sched, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "r2d2")); !os.IsNotExist(err) {
		t.Fatalf("r2d2 config %v", err)
	}
}
