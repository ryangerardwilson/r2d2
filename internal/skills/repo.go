package skills

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// remoteURL is the source of truth. main on it is what every host installs.
	remoteURL = "https://github.com/ryangerardwilson/skills"
	// skillRoot holds one directory per skill, inside the r2d2-skills plugin.
	skillRoot  = "plugins/r2d2-skills/skills"
	branch     = "main"
	authorName = "Ryan Gerard Wilson"
	authorMail = "98247320+ryangerardwilson@users.noreply.github.com"
	// gitTimeout bounds one git command, a clone or a push included.
	gitTimeout = 2 * time.Minute
)

var errNothing = errors.New("nothing to commit")

// gitFunc runs git in dir and returns its trimmed output.
type gitFunc func(ctx context.Context, dir string, args ...string) (string, error)

// repo is the working copy of the skills repository. r2d2 is its only
// writer: it never resets, stashes, or force-pushes it.
type repo struct {
	dir    string
	remote string
	git    gitFunc
}

// state is what opening or refreshing the working copy found.
type state struct {
	note     string // the clone's state, shown beside the title
	writable bool   // HEAD is main, so a skill may be committed
	broken   error  // the directory is not this repository, or it cannot be cloned
}

func defaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "r2d2", "skills"), nil
}

// runGit runs git with prompts off and the owner as author and committer.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME="+authorName, "GIT_AUTHOR_EMAIL="+authorMail,
		"GIT_COMMITTER_NAME="+authorName, "GIT_COMMITTER_EMAIL="+authorMail,
	)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return text, fmt.Errorf("git %s: %s", args[0], lastLine(text))
	}
	return text, nil
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// open clones a missing working copy, or fetches the one there and
// fast-forwards it when that is safe.
func (r *repo) open(ctx context.Context) state {
	empty, err := missingOrEmpty(r.dir)
	if err != nil {
		return state{broken: err}
	}
	if empty {
		return r.clone(ctx)
	}
	if err := r.verify(ctx); err != nil {
		return state{broken: err}
	}
	return r.sync(ctx)
}

func missingOrEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func (r *repo) clone(ctx context.Context) state {
	parent := filepath.Dir(r.dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return state{broken: err}
	}
	if _, err := r.git(ctx, parent, "clone", "--quiet", "--branch", branch, r.remote, r.dir); err != nil {
		return state{broken: err}
	}
	return state{note: "cloned", writable: true}
}

// verify is nil when dir is the top of a clone of the remote. Anything else
// is left as it is.
func (r *repo) verify(ctx context.Context) error {
	notOurs := fmt.Errorf("%s is not the skills repository; it was left as it is", r.dir)
	top, err := r.git(ctx, r.dir, "rev-parse", "--show-toplevel")
	if err != nil || !samePath(top, r.dir) {
		return notOurs
	}
	origin, err := r.git(ctx, r.dir, "remote", "get-url", "origin")
	if err != nil || remoteKey(origin) != remoteKey(r.remote) {
		return notOurs
	}
	return nil
}

func samePath(a, b string) bool {
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ra == rb
}

// remoteKey folds the https, ssh, and scp forms of a GitHub URL to one key.
func remoteKey(url string) string {
	key := strings.TrimSpace(url)
	for _, prefix := range []string{"https://", "http://", "ssh://", "git@", "git://"} {
		key = strings.TrimPrefix(key, prefix)
	}
	key = strings.Replace(key, ":", "/", 1)
	key = strings.TrimSuffix(strings.TrimSuffix(key, "/"), ".git")
	return strings.ToLower(key)
}

// facts is where the working copy stands against origin.
type facts struct {
	branch string // empty when HEAD is detached
	dirty  bool
	ahead  int // commits on main that origin/main does not have
	behind int // commits on origin/main that main does not have
}

func (r *repo) look(ctx context.Context) (facts, error) {
	var f facts
	f.branch, _ = r.git(ctx, r.dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	status, err := r.git(ctx, r.dir, "status", "--porcelain")
	if err != nil {
		return f, err
	}
	f.dirty = status != ""
	counts, err := r.git(ctx, r.dir, "rev-list", "--left-right", "--count", branch+"...origin/"+branch)
	if err != nil {
		return f, err
	}
	fields := strings.Fields(counts)
	if len(fields) == 2 {
		f.ahead, _ = strconv.Atoi(fields[0])
		f.behind, _ = strconv.Atoi(fields[1])
	}
	return f, nil
}

// hold is why main is left where it is, or empty when it may move.
func (f facts) hold() string {
	switch {
	case f.branch != branch:
		head := f.branch
		if head == "" {
			head = "a detached HEAD"
		}
		return "on " + head + ", not main; left as it is"
	case f.dirty:
		return "dirty worktree; left as it is"
	case f.ahead > 0:
		return plural(f.ahead, "unpushed commit") + "; left as it is"
	}
	return ""
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

// sync fetches origin and fast-forwards main when the worktree is clean, HEAD
// is main, and nothing is unpushed. Otherwise it says why it did not.
func (r *repo) sync(ctx context.Context) state {
	_, fetchErr := r.git(ctx, r.dir, "fetch", "--quiet", "origin")
	f, err := r.look(ctx)
	if err != nil {
		return state{note: err.Error(), writable: f.branch == branch}
	}
	st := state{writable: f.branch == branch}
	switch {
	case fetchErr != nil:
		st.note = "fetch failed: " + fetchErr.Error()
	case f.hold() != "":
		st.note = f.hold()
	case f.behind == 0:
		st.note = "main is up to date"
	default:
		if _, err := r.git(ctx, r.dir, "merge", "--quiet", "--ff-only", "origin/"+branch); err != nil {
			st.note = "fast-forward failed: " + err.Error()
			break
		}
		st.note = "fast-forwarded " + plural(f.behind, "commit")
	}
	return st
}

// skills are the directories under the skill root that hold a SKILL.md.
func (r *repo) skills() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(r.dir, skillRoot))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if info, err := os.Stat(r.skillFile(entry.Name())); err == nil && info.Mode().IsRegular() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func (r *repo) skillDir(name string) string {
	return filepath.Join(r.dir, skillRoot, name)
}

func (r *repo) skillFile(name string) string {
	return filepath.Join(r.skillDir(name), "SKILL.md")
}

// scaffold writes an empty SKILL.md for a new skill. The file stays empty
// until vim saves it. It refuses a name whose directory is already there.
func (r *repo) scaffold(name string) (string, error) {
	dir := r.skillDir(name)
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", err
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%s already exists", name)
		}
		return "", err
	}
	path := r.skillFile(name)
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// commit records only the skill's own paths. Other changes in the worktree
// stay as they are.
func (r *repo) commit(ctx context.Context, name, message string) error {
	path := skillRoot + "/" + name
	if _, err := r.git(ctx, r.dir, "add", "--all", "--", path); err != nil {
		return err
	}
	status, err := r.git(ctx, r.dir, "status", "--porcelain", "--", path)
	if err != nil {
		return err
	}
	if status == "" {
		return errNothing
	}
	_, err = r.git(ctx, r.dir, "commit", "--quiet", "-m", message, "--", path)
	return err
}

// move renames one skill directory and commits that rename.
func (r *repo) move(ctx context.Context, from, to, message string) error {
	oldPath := skillRoot + "/" + from
	newPath := skillRoot + "/" + to
	if _, err := r.git(ctx, r.dir, "mv", "--", oldPath, newPath); err != nil {
		return err
	}
	_, err := r.git(ctx, r.dir, "commit", "--quiet", "-m", message, "--", oldPath, newPath)
	return err
}

// push sends main to origin/main. It never forces.
func (r *repo) push(ctx context.Context) error {
	_, err := r.git(ctx, r.dir, "push", "--quiet", "origin", "refs/heads/"+branch+":refs/heads/"+branch)
	return err
}
