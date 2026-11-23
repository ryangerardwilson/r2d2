package skills

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// fixture is a bare remote that stands in for GitHub, and a working copy
// path that does not exist yet.
type fixture struct {
	t      *testing.T
	remote string
	dir    string
}

func newFixture(t *testing.T, skills ...string) *fixture {
	t.Helper()
	root := t.TempDir()
	f := &fixture{t: t, remote: filepath.Join(root, "remote.git"), dir: filepath.Join(root, "share", "skills")}
	f.git(root, "init", "--quiet", "--bare", "--initial-branch=main", f.remote)
	seed := filepath.Join(root, "seed")
	f.git(root, "clone", "--quiet", f.remote, seed)
	f.write(seed, ".gitkeep", "")
	for _, name := range skills {
		f.write(seed, name+"/SKILL.md", "---\nname: "+name+"\ndescription: d\n---\n")
	}
	f.git(seed, "add", "--all")
	f.git(seed, "commit", "--quiet", "-m", "seed")
	f.git(seed, "push", "--quiet", "origin", "HEAD:refs/heads/main")
	return f
}

func (f *fixture) git(dir string, args ...string) string {
	f.t.Helper()
	out, err := runGit(context.Background(), dir, args...)
	if err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return out
}

func (f *fixture) write(dir, rel, body string) {
	f.t.Helper()
	path := filepath.Join(dir, skillRoot, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) repo() *repo {
	return &repo{dir: f.dir, remote: f.remote, git: runGit}
}

// model opens the working copy as the screen does. vim appends text to the
// file it opens, unless text is empty.
func (f *fixture) model(text string) *model {
	m := &model{
		repo:    f.repo(),
		lookVim: func() (string, error) { return "vim", nil },
		openVim: func(_, path string) error { return appendText(path, text) },
		busy:    func(string) {},
	}
	m.apply(m.repo.open(context.Background()))
	return m
}

func appendText(path, text string) error {
	if text == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(text)
	return err
}

// subjects are the commit subjects on the remote main, newest first.
func (f *fixture) subjects() []string {
	return strings.Split(f.git(f.remote, "log", "--format=%s", "main"), "\n")
}

func press(m *model, keys ...string) {
	for _, key := range keys {
		m.handleKey(key)
	}
}

func typeKeys(m *model, text string) {
	for _, r := range text {
		m.handleKey(string(r))
	}
}

func TestOpenClonesAndLists(t *testing.T) {
	f := newFixture(t, "beta", "alpha")
	m := f.model("")
	if m.broken != nil || !m.writable || m.note != "cloned" {
		t.Fatalf("state = %q writable %v broken %v", m.note, m.writable, m.broken)
	}
	if got := strings.Join(m.rows, ","); got != "alpha,beta" {
		t.Fatalf("rows = %q", got)
	}
	press(m, "k")
	if m.cursor != 1 {
		t.Fatalf("k from the first row = %d, want the last", m.cursor)
	}
	press(m, "j")
	if m.cursor != 0 {
		t.Fatalf("j from the last row = %d, want the first", m.cursor)
	}
	press(m, "G")
	if m.cursor != 1 {
		t.Fatalf("G = %d", m.cursor)
	}
	press(m, "g")
	if m.cursor != 0 {
		t.Fatalf("g = %d", m.cursor)
	}
}

func TestReopenFetchesAndFastForwards(t *testing.T) {
	f := newFixture(t, "alpha")
	f.model("")
	other := filepath.Join(t.TempDir(), "other")
	f.git(filepath.Dir(other), "clone", "--quiet", f.remote, other)
	f.write(other, "beta/SKILL.md", "---\nname: beta\n---\n")
	f.git(other, "add", "--all")
	f.git(other, "commit", "--quiet", "-m", "Add beta")
	f.git(other, "push", "--quiet", "origin", "main")
	m := f.model("")
	if m.note != "fast-forwarded 1 commit" || strings.Join(m.rows, ",") != "alpha,beta" {
		t.Fatalf("note %q rows %v", m.note, m.rows)
	}
}

func TestEmptyListSaysNoSkills(t *testing.T) {
	m := newFixture(t).model("")
	press(m, "j", "k", "G", "g")
	if m.cursor != 0 || len(m.rows) != 0 {
		t.Fatalf("cursor %d rows %v", m.cursor, m.rows)
	}
	if !strings.Contains(m.view(), "no skills") {
		t.Fatalf("view = %q", m.view())
	}
	press(m, "enter")
	if m.status != "no skills" {
		t.Fatalf("enter on an empty list = %q", m.status)
	}
}

func TestShortcutsHideUntilQuestionMark(t *testing.T) {
	m := newFixture(t, "alpha").model("")
	if strings.Contains(m.view(), "new skill") {
		t.Fatal("shortcuts before ?")
	}
	press(m, "?")
	if !strings.Contains(m.view(), "new skill") {
		t.Fatal("? did not show the shortcuts")
	}
	press(m, "?")
	if strings.Contains(m.view(), "new skill") {
		t.Fatal("a second ? did not hide the shortcuts")
	}
	if !m.handleKey("q") {
		t.Fatal("q did not quit")
	}
}

var sgr = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

func TestViewIsBlackWhiteAndGray(t *testing.T) {
	f := newFixture(t, "alpha", "beta")
	m := f.model("")
	m.status, m.alarm = "push failed", true
	frames := []string{m.view()}
	m.alarm = false
	frames = append(frames, m.view())
	press(m, "?")
	frames = append(frames, m.view())
	allowed := map[string]bool{"0": true, "1": true, "1;97": true, "90": true, "7": true}
	for _, frame := range frames {
		for _, match := range sgr.FindAllStringSubmatch(frame, -1) {
			if !allowed[match[1]] {
				t.Fatalf("style %q is not black, white, or gray", match[1])
			}
		}
	}
}

func TestCreateCommitsAndPushes(t *testing.T) {
	f := newFixture(t, "alpha")
	m := f.model("\nUse it to deploy.\n")
	press(m, "n")
	typeKeys(m, "deploy")
	press(m, "enter")
	if m.naming != nil || m.alarm {
		t.Fatalf("status %q", m.status)
	}
	if got := f.subjects()[0]; got != "Add deploy" {
		t.Fatalf("remote head = %q", got)
	}
	saved, err := os.ReadFile(filepath.Join(f.dir, skillRoot, "deploy", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != "\nUse it to deploy.\n" {
		t.Fatalf("SKILL.md = %q", saved)
	}
	author := f.git(f.remote, "log", "-1", "--format=%an <%ae>|%cn <%ce>", "main")
	want := authorName + " <" + authorMail + ">"
	if author != want+"|"+want {
		t.Fatalf("author = %q", author)
	}
	if m.rows[m.clamp()] != "deploy" {
		t.Fatalf("cursor on %q", m.rows[m.clamp()])
	}
}

func TestCreateRejectsBadAndTakenNames(t *testing.T) {
	f := newFixture(t, "alpha")
	m := f.model("x")
	for _, name := range []string{"Deploy", "-deploy", "deploy-", "de_ploy", "alpha", strings.Repeat("a", 65), ""} {
		press(m, "n")
		typeKeys(m, name)
		press(m, "enter")
		if m.naming == nil || !m.alarm {
			t.Fatalf("%q: naming %v status %q", name, m.naming, m.status)
		}
		press(m, "esc")
		if m.naming != nil {
			t.Fatal("esc did not cancel")
		}
	}
	if got := f.subjects(); len(got) != 1 {
		t.Fatalf("remote = %v", got)
	}
	entries, _ := os.ReadDir(filepath.Join(f.dir, skillRoot))
	if len(entries) != 2 {
		t.Fatalf("skill root has %d entries", len(entries))
	}
}

func TestCreateEmptiedFileAddsNothing(t *testing.T) {
	f := newFixture(t)
	m := f.model("")
	m.openVim = func(_, path string) error { return os.WriteFile(path, nil, 0o644) }
	press(m, "n")
	typeKeys(m, "deploy")
	press(m, "enter")
	if _, err := os.Stat(m.repo.skillDir("deploy")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("directory left: %v", err)
	}
	if len(f.subjects()) != 1 || len(m.rows) != 0 {
		t.Fatalf("committed or listed: %v %v", f.subjects(), m.rows)
	}
}

func TestMissingVimLeavesTheScreenUp(t *testing.T) {
	f := newFixture(t, "alpha")
	m := f.model("x")
	m.lookVim = func() (string, error) { return "", errors.New("not found") }
	press(m, "n")
	typeKeys(m, "deploy")
	press(m, "enter")
	if m.status != "vim is not on PATH" {
		t.Fatalf("status = %q", m.status)
	}
	if _, err := os.Stat(m.repo.skillDir("deploy")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a file was written without vim")
	}
	press(m, "enter")
	if m.status != "vim is not on PATH" || m.handleKey("j") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestEditCommitsOnlyAChange(t *testing.T) {
	f := newFixture(t, "alpha")
	m := f.model("")
	press(m, "enter")
	if m.status != "no change to alpha" || len(f.subjects()) != 1 {
		t.Fatalf("status %q remote %v", m.status, f.subjects())
	}
	m.openVim = func(_, path string) error { return appendText(path, "More.\n") }
	press(m, "enter")
	if got := f.subjects()[0]; got != "Update alpha" {
		t.Fatalf("remote head = %q status %q", got, m.status)
	}
}

func TestRemoveAsksFirst(t *testing.T) {
	f := newFixture(t, "alpha", "beta")
	m := f.model("")
	press(m, "d", "n")
	if len(m.rows) != 2 || m.status != "kept alpha" || len(f.subjects()) != 1 {
		t.Fatalf("cancel: rows %v status %q", m.rows, m.status)
	}
	press(m, "d")
	if !strings.Contains(m.view(), "remove alpha? y/n") {
		t.Fatalf("view = %q", m.view())
	}
	press(m, "y")
	if strings.Join(m.rows, ",") != "beta" {
		t.Fatalf("rows = %v", m.rows)
	}
	if got := f.subjects()[0]; got != "Remove alpha" {
		t.Fatalf("remote head = %q", got)
	}
	if _, err := os.Stat(m.repo.skillDir("alpha")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("alpha is still there")
	}
}

func TestRefusedPushKeepsTheCommit(t *testing.T) {
	f := newFixture(t)
	m := f.model("x")
	hook := filepath.Join(f.remote, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho refused >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	press(m, "n")
	typeKeys(m, "deploy")
	press(m, "enter")
	if !m.alarm || !strings.Contains(m.status, "push failed") {
		t.Fatalf("status = %q", m.status)
	}
	if got := f.git(f.dir, "log", "-1", "--format=%s"); got != "Add deploy" {
		t.Fatalf("local head = %q", got)
	}
	if len(f.subjects()) != 1 {
		t.Fatal("the remote took the push")
	}
	if strings.Join(m.rows, ",") != "deploy" {
		t.Fatalf("rows = %v", m.rows)
	}
	press(m, "r")
	if m.note != "1 unpushed commit; left as it is" {
		t.Fatalf("note after r = %q", m.note)
	}
}

func TestDirtyWorktreeIsLeftInPlace(t *testing.T) {
	f := newFixture(t, "alpha")
	m := f.model("")
	other := filepath.Join(t.TempDir(), "other")
	f.git(filepath.Dir(other), "clone", "--quiet", f.remote, other)
	f.write(other, "beta/SKILL.md", "---\nname: beta\n---\n")
	f.git(other, "add", "--all")
	f.git(other, "commit", "--quiet", "-m", "Add beta")
	f.git(other, "push", "--quiet", "origin", "main")
	head := f.git(f.dir, "rev-parse", "HEAD")
	f.write(f.dir, "alpha/SKILL.md", "local edit\n")
	m = f.model("")
	if m.note != "dirty worktree; left as it is" {
		t.Fatalf("note = %q", m.note)
	}
	press(m, "r")
	if m.note != "dirty worktree; left as it is" || f.git(f.dir, "rev-parse", "HEAD") != head {
		t.Fatalf("r moved main or lost the note: %q", m.note)
	}
	body, _ := os.ReadFile(m.repo.skillFile("alpha"))
	if string(body) != "local edit\n" {
		t.Fatalf("the local edit is gone: %q", body)
	}
}

func TestAnotherBranchIsReadOnly(t *testing.T) {
	f := newFixture(t, "alpha")
	f.model("")
	f.git(f.dir, "switch", "--quiet", "-c", "draft")
	m := f.model("x")
	if m.writable || m.note != "on draft, not main; left as it is" {
		t.Fatalf("writable %v note %q", m.writable, m.note)
	}
	press(m, "n")
	if m.naming != nil || !strings.HasPrefix(m.status, "writes are off") {
		t.Fatalf("n on another branch: %q", m.status)
	}
}

func TestAnotherDirectoryIsRefused(t *testing.T) {
	f := newFixture(t, "alpha")
	if err := os.MkdirAll(f.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(f.dir, "notes.txt")
	if err := os.WriteFile(keep, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := f.model("x")
	if m.broken == nil || len(m.rows) != 0 {
		t.Fatalf("broken %v rows %v", m.broken, m.rows)
	}
	press(m, "n", "d", "enter", "r")
	if m.naming != nil || m.removing != "" {
		t.Fatal("a write prompt opened")
	}
	if body, err := os.ReadFile(keep); err != nil || string(body) != "mine" {
		t.Fatalf("the directory changed: %v", err)
	}
	if !strings.Contains(m.view(), "is not the skills repository") {
		t.Fatalf("view = %q", m.view())
	}
}

func TestAnotherCloneIsRefused(t *testing.T) {
	f := newFixture(t, "alpha")
	g := newFixture(t, "beta")
	if err := os.MkdirAll(filepath.Dir(f.dir), 0o755); err != nil {
		t.Fatal(err)
	}
	f.git(filepath.Dir(f.dir), "clone", "--quiet", g.remote, f.dir)
	m := f.model("x")
	if m.broken == nil {
		t.Fatal("a clone of another remote opened")
	}
}

func TestRemoteKeyFoldsGitHubForms(t *testing.T) {
	want := remoteKey(remoteURL)
	for _, url := range []string{
		"https://github.com/ryangerardwilson/skills.git",
		"git@github.com:ryangerardwilson/skills.git",
		"ssh://git@github.com/ryangerardwilson/skills",
		"https://github.com/ryangerardwilson/skills/",
	} {
		if remoteKey(url) != want {
			t.Fatalf("%s folds to %q, want %q", url, remoteKey(url), want)
		}
	}
	if remoteKey("https://github.com/ryangerardwilson/r2d2") == want {
		t.Fatal("another repository matched")
	}
}

func TestHelpAndPipe(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"help"}, &out, &errOut, true); code != 0 || out.String() != Help {
		t.Fatalf("help: %d %q", code, out.String())
	}
	out.Reset()
	if code := run(nil, &out, &errOut, false); code != 0 || out.String() != Help {
		t.Fatalf("pipe: %d", code)
	}
	for _, word := range []string{"r2d2 skills", "~/.local/share/r2d2/skills", "vim", "main"} {
		if !strings.Contains(Help, word) {
			t.Fatalf("help does not name %q", word)
		}
	}
	out.Reset()
	if code := run([]string{"--resume", "0b6a2f4e-7c1d-4e8a-9f3b-2d5c6e7f8a9b"}, &out, &errOut, false); code != 0 {
		t.Fatalf("--resume: %d", code)
	}
	if code := run([]string{"extra"}, &out, &errOut, false); code != 2 {
		t.Fatalf("extra: %d", code)
	}
}
