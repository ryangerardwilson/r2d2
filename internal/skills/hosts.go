package skills

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
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
	// pluginName is the one plugin every host installs from the marketplace.
	pluginName = "skills"
	pluginID   = "skills@skills"
	// pluginSkills is the skill root inside an installed plugin directory.
	pluginSkills = "skills"
	// hostTimeout bounds one host command. A marketplace update clones.
	hostTimeout = 3 * time.Minute
	maxHostErr  = 300
)

// runFunc runs one host command and returns its standard output.
type runFunc func(ctx context.Context, args ...string) (string, error)

// host is one agent host that installs the skills plugin from main. steps
// replace its installed copy. installed names the directories that copy
// lives in, read after the steps ran; out is the last step's output.
type host struct {
	name      string
	steps     [][]string
	installed func(out string) ([]string, error)
}

// hostsFor is Grok Build, Claude Code, and Codex, in the order they refresh.
func hostsFor(home, claudeDir string) []host {
	return []host{
		{
			name: "Grok Build",
			steps: [][]string{
				{"grok", "plugin", "marketplace", "update", pluginName},
				{"grok", "plugin", "update", pluginName},
			},
			installed: func(string) ([]string, error) {
				return grokInstalled(filepath.Join(home, ".grok", "installed-plugins", "registry.json"))
			},
		},
		{
			name: "Claude Code",
			steps: [][]string{
				{"claude", "plugin", "marketplace", "update", pluginName},
				{"claude", "plugin", "update", pluginID, "-y"},
			},
			installed: func(string) ([]string, error) {
				return claudeInstalled(filepath.Join(claudeDir, "plugins", "installed_plugins.json"))
			},
		},
		{
			name: "Codex",
			steps: [][]string{
				{"codex", "plugin", "marketplace", "upgrade", pluginName},
				{"codex", "plugin", "add", pluginID, "--json"},
			},
			installed: codexInstalled,
		},
	}
}

// claudeConfigDir is where Claude Code keeps its plugins.
func claudeConfigDir(home string) string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(home, ".claude")
}

// runHost runs a host command with stdin closed. Its output never reaches
// the screen; an error carries the command and what it printed.
func runHost(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, hostTimeout)
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), nil
	}
	if errors.Is(err, exec.ErrNotFound) {
		return "", fmt.Errorf("%s is not on PATH", args[0])
	}
	text := oneLine(stderr.String())
	if text == "" {
		text = oneLine(stdout.String())
	}
	if text == "" {
		text = err.Error()
	}
	return "", fmt.Errorf("%s: %s", strings.Join(args, " "), text)
}

// oneLine folds output to one printable line of at most maxHostErr bytes.
func oneLine(text string) string {
	text = strings.Map(func(r rune) rune {
		if r < ' ' || r == 0x7f {
			return ' '
		}
		return r
	}, text)
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > maxHostErr {
		text = text[:maxHostErr] + "..."
	}
	return text
}

// refreshHost runs the host's steps and then checks every installed
// directory against want. A step that exits 0 and leaves the old files is
// a failure.
func refreshHost(ctx context.Context, run runFunc, h host, want map[string]string) error {
	var out string
	for _, step := range h.steps {
		var err error
		if out, err = run(ctx, step...); err != nil {
			return err
		}
	}
	dirs, err := h.installed(out)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if err := sameTree(filepath.Join(dir, pluginSkills), want); err != nil {
			return err
		}
	}
	return nil
}

// grokInstalled is the installed copy of skills in Grok's registry, and
// the marketplace directory it was copied from when that is local.
func grokInstalled(registry string) ([]string, error) {
	var reg struct {
		Repos map[string]struct {
			Kind struct {
				Type       string `json:"type"`
				SourcePath string `json:"source_path"`
			} `json:"kind"`
			Path        string                     `json:"path"`
			Plugins     map[string]json.RawMessage `json:"plugins"`
			Marketplace struct {
				Source string `json:"source_url_or_path"`
			} `json:"marketplace"`
		} `json:"repos"`
	}
	if err := readJSON(registry, &reg); err != nil {
		return nil, err
	}
	for _, repo := range reg.Repos {
		if _, ok := repo.Plugins[pluginName]; !ok || remoteKey(repo.Marketplace.Source) != remoteKey(remoteURL) {
			continue
		}
		dirs := []string{repo.Path}
		if repo.Kind.Type == "Local" && repo.Kind.SourcePath != "" {
			dirs = append(dirs, repo.Kind.SourcePath)
		}
		return dirs, nil
	}
	return nil, fmt.Errorf("%s is not installed in %s", pluginName, registry)
}

// claudeInstalled is the install path of skills@skills, the user scope first.
func claudeInstalled(path string) ([]string, error) {
	var file struct {
		Plugins map[string][]struct {
			Scope       string `json:"scope"`
			InstallPath string `json:"installPath"`
		} `json:"plugins"`
	}
	if err := readJSON(path, &file); err != nil {
		return nil, err
	}
	entries := file.Plugins[pluginID]
	for _, entry := range entries {
		if entry.Scope == "user" {
			return []string{entry.InstallPath}, nil
		}
	}
	if len(entries) > 0 {
		return []string{entries[0].InstallPath}, nil
	}
	return nil, fmt.Errorf("%s is not installed in %s", pluginID, path)
}

// codexInstalled is the installedPath that codex plugin add --json printed.
func codexInstalled(out string) ([]string, error) {
	var result struct {
		InstalledPath string `json:"installedPath"`
	}
	start := strings.Index(out, "{")
	if start < 0 || json.Unmarshal([]byte(out[start:]), &result) != nil || result.InstalledPath == "" {
		return nil, fmt.Errorf("codex plugin add did not name the installed path of %s", pluginID)
	}
	return []string{result.InstalledPath}, nil
}

func readJSON(path string, into any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// tree maps each file under the skill root at HEAD, relative to that root,
// to its blob id.
func (r *repo) tree(ctx context.Context) (map[string]string, error) {
	out, err := r.git(ctx, r.dir, "ls-tree", "-r", "-z", "HEAD", "--", skillRoot)
	if err != nil {
		return nil, err
	}
	tree := map[string]string{}
	for _, entry := range strings.Split(out, "\x00") {
		meta, path, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			continue
		}
		tree[strings.TrimPrefix(path, skillRoot+"/")] = fields[2]
	}
	return tree, nil
}

// sameTree is nil when dir holds exactly the files in want, byte for byte.
func sameTree(dir string, want map[string]string) error {
	got, err := installedTree(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s is missing", dir)
	}
	if err != nil {
		return err
	}
	var diff []string
	for path, id := range want {
		if got[path] != id {
			diff = append(diff, path)
		}
	}
	for path := range got {
		if _, ok := want[path]; !ok {
			diff = append(diff, path)
		}
	}
	if len(diff) == 0 {
		return nil
	}
	sort.Strings(diff)
	if len(diff) > 3 {
		diff = append(diff[:3], "and "+strconv.Itoa(len(diff)-3)+" more")
	}
	return fmt.Errorf("%s still differs from main at %s", dir, strings.Join(diff, ", "))
}

// installedTree maps each file under dir to the blob id git gives it.
func installedTree(dir string) (map[string]string, error) {
	tree := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		var body []byte
		if entry.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			body = []byte(target)
		} else if body, err = os.ReadFile(path); err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		tree[filepath.ToSlash(rel)] = blobID(body)
		return nil
	})
	return tree, err
}

func blobID(body []byte) string {
	sum := sha1.New()
	fmt.Fprintf(sum, "blob %d\x00", len(body))
	sum.Write(body)
	return hex.EncodeToString(sum.Sum(nil))
}
