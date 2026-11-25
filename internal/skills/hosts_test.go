package skills

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var hostNames = []string{"Grok Build", "Claude Code", "Codex"}

// fakeHosts stand in for Grok Build, Claude Code, and Codex. A host's one
// step copies the working copy's skills into its installed directory,
// unless fail gives it an error or stale makes it exit 0 and copy nothing.
type fakeHosts struct {
	t        *testing.T
	m        *model
	src      string
	dirs     map[string]string
	fail     map[string]error
	stale    map[string]bool
	calls    []string
	statuses []string // the status when each host ran
}

func newFakeHosts(t *testing.T, m *model, workdir string) *fakeHosts {
	f := &fakeHosts{t: t, m: m, src: filepath.Join(workdir, skillRoot), dirs: map[string]string{},
		fail: map[string]error{}, stale: map[string]bool{}}
	root := t.TempDir()
	for _, name := range hostNames {
		f.dirs[name] = filepath.Join(root, strings.ReplaceAll(name, " ", "-"))
	}
	return f
}

func (f *fakeHosts) hosts() []host {
	var hosts []host
	for _, name := range hostNames {
		dir := f.dirs[name]
		hosts = append(hosts, host{
			name:      name,
			steps:     [][]string{{name}},
			installed: func(string) ([]string, error) { return []string{dir}, nil },
		})
	}
	return hosts
}

func (f *fakeHosts) run(_ context.Context, args ...string) (string, error) {
	name := args[0]
	f.calls = append(f.calls, name)
	f.statuses = append(f.statuses, f.m.status)
	if err := f.fail[name]; err != nil {
		return "", err
	}
	if f.stale[name] {
		return "", nil
	}
	return "", copyTree(f.src, filepath.Join(f.dirs[name], pluginSkills))
}

// installed is the body of rel in the host's installed copy, or "" when
// it is not there.
func (f *fakeHosts) installed(name, rel string) string {
	body, err := os.ReadFile(filepath.Join(f.dirs[name], pluginSkills, rel))
	if err != nil {
		return ""
	}
	return string(body)
}

func copyTree(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), body, 0o644)
	})
}

func addSkill(m *model, name string) {
	press(m, "n")
	typeKeys(m, name)
	press(m, "enter")
}

func TestPushInstallsForEveryHost(t *testing.T) {
	f := newFixture(t, "alpha")
	m := f.model("\nUse it to deploy.\n")
	addSkill(m, "deploy")
	if !reflect.DeepEqual(f.hosts.calls, hostNames) {
		t.Fatalf("calls = %v", f.hosts.calls)
	}
	for _, status := range f.hosts.statuses {
		if strings.Contains(status, "pushed") {
			t.Fatalf("status said pushed before every host was installed: %q", status)
		}
	}
	if m.alarm || !strings.HasPrefix(m.status, "pushed Add deploy") {
		t.Fatalf("status = %q", m.status)
	}
	for _, name := range hostNames {
		if !strings.Contains(m.status, name) {
			t.Fatalf("status %q does not name %s", m.status, name)
		}
		if f.hosts.installed(name, "deploy/SKILL.md") != "\nUse it to deploy.\n" {
			t.Fatalf("%s did not install deploy", name)
		}
	}
	press(m, "d", "y")
	if m.alarm || !strings.HasPrefix(m.status, "pushed Remove deploy") {
		t.Fatalf("remove status = %q", m.status)
	}
	for _, name := range hostNames {
		if f.hosts.installed(name, "deploy/SKILL.md") != "" {
			t.Fatalf("%s still has deploy", name)
		}
	}
}

func TestEmptyCommitInstallsNothing(t *testing.T) {
	f := newFixture(t, "alpha")
	m := f.model("")
	m.commitAndPush("alpha", "Update alpha", func(context.Context) error { return errNothing })
	if len(f.hosts.calls) != 0 || m.status != "nothing to commit for alpha" {
		t.Fatalf("calls %v status %q", f.hosts.calls, m.status)
	}
	press(m, "enter")
	if len(f.hosts.calls) != 0 {
		t.Fatalf("an unchanged edit refreshed %v", f.hosts.calls)
	}
}

func TestFailedInstallNamesTheHostAndKeepsThePush(t *testing.T) {
	f := newFixture(t)
	m := f.model("x")
	f.hosts.fail["Claude Code"] = errors.New("claude plugin update r2d2-skills@skills -y: no network")
	addSkill(m, "deploy")
	if !reflect.DeepEqual(f.hosts.calls, hostNames) {
		t.Fatalf("calls = %v", f.hosts.calls)
	}
	if !m.alarm || strings.Contains(m.status, "pushed") {
		t.Fatalf("status = %q", m.status)
	}
	if !strings.Contains(m.status, "Claude Code: claude plugin update r2d2-skills@skills -y: no network") {
		t.Fatalf("status does not name the host and error: %q", m.status)
	}
	if strings.Contains(m.status, "Grok Build:") || strings.Contains(m.status, "Codex:") {
		t.Fatalf("status blames a host that worked: %q", m.status)
	}
	if got := f.subjects()[0]; got != "Add deploy" {
		t.Fatalf("remote head = %q", got)
	}
	if got := f.git(f.dir, "log", "-1", "--format=%s"); got != "Add deploy" {
		t.Fatalf("local head = %q", got)
	}
}

func TestStaleInstallIsAFailure(t *testing.T) {
	f := newFixture(t)
	m := f.model("x")
	f.hosts.stale["Codex"] = true
	if err := copyTree(f.hosts.src, filepath.Join(f.hosts.dirs["Codex"], pluginSkills)); err != nil {
		t.Fatal(err)
	}
	addSkill(m, "deploy")
	if !m.alarm || strings.Contains(m.status, "pushed") {
		t.Fatalf("status = %q", m.status)
	}
	if !strings.Contains(m.status, "Codex: ") || !strings.Contains(m.status, "deploy/SKILL.md") {
		t.Fatalf("status = %q", m.status)
	}
	if got := f.subjects()[0]; got != "Add deploy" {
		t.Fatalf("remote head = %q", got)
	}
}

func TestLastHostDecidesPushed(t *testing.T) {
	f := newFixture(t)
	m := f.model("x")
	f.hosts.fail["Codex"] = errors.New("codex is not on PATH")
	addSkill(m, "deploy")
	if strings.Contains(m.status, "pushed") || !strings.Contains(m.status, "Codex: codex is not on PATH") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestHostCommands(t *testing.T) {
	hosts := hostsFor("/home/u", "/home/u/.claude")
	want := map[string][][]string{
		"Grok Build": {
			{"grok", "plugin", "marketplace", "update", "skills"},
			{"grok", "plugin", "update", "r2d2-skills"},
		},
		"Claude Code": {
			{"claude", "plugin", "marketplace", "update", "skills"},
			{"claude", "plugin", "update", "r2d2-skills@skills", "-y"},
		},
		"Codex": {
			{"codex", "plugin", "marketplace", "upgrade", "skills"},
			{"codex", "plugin", "add", "r2d2-skills@skills", "--json"},
		},
	}
	var names []string
	for _, h := range hosts {
		names = append(names, h.name)
		if !reflect.DeepEqual(h.steps, want[h.name]) {
			t.Fatalf("%s steps = %v", h.name, h.steps)
		}
	}
	if !reflect.DeepEqual(names, hostNames) {
		t.Fatalf("hosts = %v", names)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledPaths(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "installed_plugins.json")
	writeFile(t, claude, `{"version":2,"plugins":{"r2d2-skills@skills":[
		{"scope":"project","installPath":"/p"},
		{"scope":"user","installPath":"/c/cache/skills/skills/972f75e1bd58"}]}}`)
	if got, err := claudeInstalled(claude); err != nil || !reflect.DeepEqual(got, []string{"/c/cache/skills/skills/972f75e1bd58"}) {
		t.Fatalf("claude = %v %v", got, err)
	}
	writeFile(t, claude, `{"plugins":{}}`)
	if _, err := claudeInstalled(claude); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("claude without skills: %v", err)
	}

	grok := filepath.Join(root, "registry.json")
	writeFile(t, grok, `{"version":1,"repos":{
		"figma-1":{"kind":{"type":"Git"},"path":"/g/figma","plugins":{"figma":{}},
			"marketplace":{"source_url_or_path":"https://github.com/xai-org/plugin-marketplace.git"}},
		"skills-b91c08ab":{"kind":{"type":"Local","source_path":"/g/cache/plugins/skills"},
			"path":"/g/installed-plugins/skills-b91c08ab","plugins":{"r2d2-skills":{}},
			"marketplace":{"source_url_or_path":"https://github.com/ryangerardwilson/skills.git"}}}}`)
	want := []string{"/g/installed-plugins/skills-b91c08ab", "/g/cache/plugins/skills"}
	if got, err := grokInstalled(grok); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("grok = %v %v", got, err)
	}
	if _, err := grokInstalled(filepath.Join(root, "missing.json")); err == nil {
		t.Fatal("a missing grok registry passed")
	}
	writeFile(t, grok, `{"version":1,"repos":{
		"r2d2-skills-git":{"kind":{"type":"Git","url":"https://github.com/ryangerardwilson/skills","subdir":"plugins/r2d2-skills"},
			"path":"/g/installed-plugins/r2d2-skills","plugins":{"r2d2-skills":{}}}}}`)
	if got, err := grokInstalled(grok); err != nil || !reflect.DeepEqual(got, []string{"/g/installed-plugins/r2d2-skills/plugins/r2d2-skills"}) {
		t.Fatalf("grok git = %v %v", got, err)
	}
	writeFile(t, grok, `{"repos":{"other":{"kind":{"type":"Git","url":"https://example.com/nope"},"path":"/no","plugins":{"r2d2-skills":{}}}}}`)
	if _, err := grokInstalled(grok); err == nil {
		t.Fatal("a different git url passed")
	}

	out := `{"pluginId":"r2d2-skills@skills","installedPath":"/x/plugins/cache/skills/skills/local"}`
	if got, err := codexInstalled(out); err != nil || !reflect.DeepEqual(got, []string{"/x/plugins/cache/skills/skills/local"}) {
		t.Fatalf("codex = %v %v", got, err)
	}
	if _, err := codexInstalled("Added plugin"); err == nil {
		t.Fatal("codex without a path passed")
	}
}

func TestSameTreeMatchesGitBlobs(t *testing.T) {
	f := newFixture(t, "alpha")
	r := f.repo()
	r.open(context.Background())
	want, err := r.tree(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := want["alpha/SKILL.md"]; !ok || len(want) != 2 {
		t.Fatalf("tree = %v", want)
	}
	dir := filepath.Join(t.TempDir(), "skills")
	if err := copyTree(filepath.Join(f.dir, skillRoot), dir); err != nil {
		t.Fatal(err)
	}
	if err := sameTree(dir, want); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "alpha", "SKILL.md"), "old")
	writeFile(t, filepath.Join(dir, "gone", "SKILL.md"), "removed upstream")
	err = sameTree(dir, want)
	if err == nil || !strings.Contains(err.Error(), "alpha/SKILL.md, gone/SKILL.md") {
		t.Fatalf("changed tree: %v", err)
	}
	none := filepath.Join(t.TempDir(), "none")
	if err := sameTree(none, want); err == nil || err.Error() != none+" is missing" {
		t.Fatalf("a missing directory: %v", err)
	}
}

func TestRunHostNamesAMissingCommand(t *testing.T) {
	_, err := runHost(context.Background(), "r2d2-no-such-host-cli", "plugin")
	if err == nil || err.Error() != "r2d2-no-such-host-cli is not on PATH" {
		t.Fatalf("err = %v", err)
	}
	_, err = runHost(context.Background(), "sh", "-c", "echo out; printf 'line one\\nline two\\n' >&2; exit 3")
	if err == nil || err.Error() != "sh -c echo out; printf 'line one\\nline two\\n' >&2; exit 3: line one line two" {
		t.Fatalf("err = %v", err)
	}
}
