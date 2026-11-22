package rec

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const recentRecordingLimit = 20

type recording struct {
	path  string
	label string
	mod   time.Time
}

func scanVideos() []recording {
	dir := videosDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	found := make([]recording, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !isRecordingName(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		found = append(found, recording{
			path:  filepath.Join(dir, name),
			label: recordingLabel(name),
			mod:   info.ModTime(),
		})
	}
	sort.Slice(found, func(i, j int) bool {
		if !found[i].mod.Equal(found[j].mod) {
			return found[i].mod.After(found[j].mod)
		}
		return found[i].path > found[j].path
	})
	if len(found) > recentRecordingLimit {
		found = found[:recentRecordingLimit]
	}
	return found
}

func isRecordingName(name string) bool {
	return strings.HasPrefix(name, "screenrecording-") &&
		strings.HasSuffix(name, ".mp4") &&
		!strings.Contains(name, "-processed")
}

func recordingLabel(name string) string {
	stem := strings.TrimSuffix(strings.TrimPrefix(name, "screenrecording-"), ".mp4")
	stamp, err := time.Parse("2006-01-02_15-04-05", stem)
	if err != nil {
		return stem
	}
	return stamp.Format("2006-01-02 15:04:05")
}

func videosDir() string {
	if dir := os.Getenv("OMARCHY_SCREENRECORD_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("XDG_VIDEOS_DIR"); dir != "" {
		return dir
	}
	if dir := userVideosDir(); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Videos")
}

func userVideosDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		value, ok := strings.CutPrefix(line, "XDG_VIDEOS_DIR=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		return os.ExpandEnv(strings.ReplaceAll(value, "$HOME", home))
	}
	return ""
}

func playWithMpv(path string) error {
	cmd := exec.Command("mpv", "--no-terminal", "--force-window=immediate", path)
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	defer devnull.Close()
	cmd.Stdin = devnull
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
