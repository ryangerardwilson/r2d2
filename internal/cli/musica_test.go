package cli

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// stubMusicaCenter records each launcher script r2d2 musica starts.
func stubMusicaCenter(t *testing.T, err error) *[]string {
	t.Helper()
	started := &[]string{}
	previous := startMusicaCenter
	startMusicaCenter = func(script string) error {
		*started = append(*started, script)
		return err
	}
	t.Cleanup(func() { startMusicaCenter = previous })
	return started
}

func TestBareMusicaOpensTheTesseractCenter(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	started := stubMusicaCenter(t, nil)

	var out, errb bytes.Buffer
	if code := Run([]string{"r2d2", "musica"}, &out, &errb); code != 0 {
		t.Fatalf("code %d\n%s%s", code, out.String(), errb.String())
	}
	want := filepath.Join("/xdg", "quickshell", "omarchy-launcher", "open.sh")
	if len(*started) != 1 || (*started)[0] != want {
		t.Fatalf("started %q, want %q", *started, want)
	}
	if out.Len() != 0 {
		t.Fatalf("bare r2d2 musica printed %q", out.String())
	}
}

func TestBareMusicaUsesHomeConfigWithoutAnAbsoluteXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "relative")
	t.Setenv("HOME", "/home/someone")
	started := stubMusicaCenter(t, nil)

	var out, errb bytes.Buffer
	if code := Run([]string{"r2d2", "musica"}, &out, &errb); code != 0 || len(*started) != 1 || (*started)[0] != "/home/someone/.config/quickshell/omarchy-launcher/open.sh" {
		t.Fatalf("code %d started %q\n%s", code, *started, errb.String())
	}
}

func TestBareMusicaNamesTheLauncherWhenItCannotStart(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	stubMusicaCenter(t, errors.New("no such file or directory"))

	var out, errb bytes.Buffer
	if code := Run([]string{"r2d2", "musica"}, &out, &errb); code != 1 || !strings.Contains(errb.String(), "/xdg/quickshell/omarchy-launcher/open.sh musica") {
		t.Fatalf("code %d\n%s", code, errb.String())
	}
}

func TestTheRealMusicaStarterReportsAMissingLauncher(t *testing.T) {
	if err := startMusicaCenter(filepath.Join(t.TempDir(), "open.sh")); err == nil {
		t.Fatal("starting a missing launcher script succeeded")
	}
}

func TestMusicaWithArgumentsStaysOnThePlayer(t *testing.T) {
	started := stubMusicaCenter(t, nil)
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	var out, errb bytes.Buffer
	if code := Run([]string{"r2d2", "musica", "help"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "musica terminal music player") {
		t.Fatalf("help: code %d\n%s%s", code, out.String(), errb.String())
	}
	out.Reset()
	errb.Reset()
	if code := Run([]string{"r2d2", "musica", "end"}, &out, &errb); code != 1 || !strings.Contains(errb.String(), "musica is not running") {
		t.Fatalf("end with no session: code %d\n%s%s", code, out.String(), errb.String())
	}
	if len(*started) != 0 {
		t.Fatalf("a command with arguments opened the center: %q", *started)
	}
}
