package rec

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func emptyRecordings() []recording { return nil }

func recordings(rows ...recording) func() []recording {
	return func() []recording { return rows }
}

// testModel is the screen with no recordings.
func testModel(do func(string) Result) model {
	return newModel(do, emptyRecordings, playWithMpv)
}

// fake is a widget that answers from a script. Nothing here starts GTK, a camera, or the recorder.
type fake struct {
	open     bool
	replies  map[string]string
	statuses []string
	sent     []string
	started  [][]string
	internal [][]string
	quitOut  string
	quitErr  string
	quitCode int
}

func (f *fake) Send(command string, _ time.Duration) (string, error) {
	f.sent = append(f.sent, command)
	if !f.open {
		return "", os.ErrNotExist
	}
	if command == "status" && len(f.statuses) > 0 {
		reply := f.statuses[0]
		if len(f.statuses) > 1 {
			f.statuses = f.statuses[1:]
		}
		return reply, nil
	}
	return f.replies[command], nil
}

func (f *fake) Start(args []string) error {
	f.started = append(f.started, args)
	f.open = true
	return nil
}

func (f *fake) Internal(args []string) (string, string, int) {
	f.internal = append(f.internal, args)
	return f.quitOut, f.quitErr, f.quitCode
}

func noScreen(t *testing.T) func(Client, io.Writer) (Result, error) {
	return func(Client, io.Writer) (Result, error) {
		t.Fatal("the control screen started")
		return Result{}, nil
	}
}

func runFake(t *testing.T, f *fake, tty bool, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb, deps{widget: f, tty: tty, sleep: func(time.Duration) {}, screen: noScreen(t)})
	return code, out.String(), errb.String()
}

func TestNoTerminalOpensTheCameraAndExits0(t *testing.T) {
	f := &fake{statuses: []string{"idle"}}
	code, out, errText := runFake(t, f, false)
	if code != 0 || out != "" || errText != "" {
		t.Fatalf("code %d out %q err %q", code, out, errText)
	}
	if len(f.started) != 1 || strings.Join(f.started[0], " ") != "open" {
		t.Fatalf("started %v", f.started)
	}
}

func TestNoTerminalRaisesAnOpenCamera(t *testing.T) {
	f := &fake{open: true}
	if code, out, _ := runFake(t, f, false); code != 0 || out != "" {
		t.Fatalf("code %d out %q", code, out)
	}
	if len(f.started) != 0 || f.sent[0] != "raise" {
		t.Fatalf("started %v sent %v", f.started, f.sent)
	}
}

func TestATerminalOpensTheScreen(t *testing.T) {
	f := &fake{}
	opened := false
	var out, errb bytes.Buffer
	code := run(nil, &out, &errb, deps{widget: f, tty: true, screen: func(Client, io.Writer) (Result, error) {
		opened = true
		return Result{Out: "/home/ryan/Videos/a.mp4"}, nil
	}})
	if code != 0 || !opened || out.String() != "/home/ryan/Videos/a.mp4\n" {
		t.Fatalf("code %d opened %v out %q", code, opened, out.String())
	}
}

func TestACommandPrintsTheReply(t *testing.T) {
	f := &fake{open: true, replies: map[string]string{
		"status": "recording 0:01", "pause": "paused 0:01 focus", "start": "busy", "stop": "/home/ryan/Videos/a.mp4",
	}}
	for command, want := range map[string]string{
		"status": "recording 0:01\n", "pause": "paused 0:01 focus\n", "start": "busy\n", "stop": "/home/ryan/Videos/a.mp4\n",
	} {
		code, out, errText := runFake(t, f, true, command)
		if code != 0 || out != want || errText != "" {
			t.Fatalf("%s: code %d out %q err %q", command, code, out, errText)
		}
	}
	if len(f.started) != 0 {
		t.Fatalf("started %v", f.started)
	}
}

func TestAClosedCameraIsNotOpen(t *testing.T) {
	for _, command := range []string{"status", "pause", "resume", "stop"} {
		f := &fake{}
		code, out, errText := runFake(t, f, false, command)
		if code != 1 || out != "" || errText != "rec is not open\n" {
			t.Fatalf("%s: code %d out %q err %q", command, code, out, errText)
		}
		if len(f.started) != 0 || len(f.internal) != 0 {
			t.Fatalf("%s started %v internal %v", command, f.started, f.internal)
		}
	}
}

func TestQuitOnAClosedCameraStillSavesTheRecorder(t *testing.T) {
	f := &fake{quitOut: "/home/ryan/Videos/b.mp4\n"}
	code, out, _ := runFake(t, f, false, "quit")
	if code != 0 || out != "/home/ryan/Videos/b.mp4\n" || len(f.internal) != 1 || f.internal[0][0] != "quit" {
		t.Fatalf("code %d out %q internal %v", code, out, f.internal)
	}
	f = &fake{quitErr: "rec is not open\n", quitCode: 1}
	code, _, errText := runFake(t, f, false, "quit")
	if code != 1 || errText != "rec is not open\n" || len(f.started) != 0 {
		t.Fatalf("code %d err %q", code, errText)
	}
}

func TestStartOpensTheCameraAndWaitsForTheTake(t *testing.T) {
	f := &fake{statuses: []string{"idle", "starting", "recording 0:00"}}
	code, out, _ := runFake(t, f, false, "--size", "large", "start")
	if code != 0 || out != "recording 0:00\n" {
		t.Fatalf("code %d out %q", code, out)
	}
	if len(f.started) != 1 || strings.Join(f.started[0], " ") != "--size large start" {
		t.Fatalf("started %v", f.started)
	}
}

func TestFocusOpensTheCameraFillingTheScreen(t *testing.T) {
	f := &fake{statuses: []string{"idle focus"}}
	if code, out, _ := runFake(t, f, false, "focus"); code != 0 || out != "idle focus\n" || f.started[0][0] != "focus" {
		t.Fatalf("code %d out %q started %v", code, out, f.started)
	}
}

func TestSizeIsRemembered(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	f := &fake{open: true, replies: map[string]string{"status": "idle"}}
	if code, _, _ := runFake(t, f, false, "--size=medium", "status"); code != 0 {
		t.Fatalf("code %d", code)
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "rec", "size"))
	if err != nil || string(data) != "medium\n" {
		t.Fatalf("%q %v", data, err)
	}
	if code, _, _ := runFake(t, f, false, "--size", "huge"); code != 2 {
		t.Fatalf("bad size %d", code)
	}
}

func TestHelpAndUnknownWords(t *testing.T) {
	code, out, _ := runFake(t, &fake{}, false, "help")
	if code != 0 || !strings.Contains(out, "r2d2 rec") || !strings.Contains(out, "j and k") || !strings.Contains(out, "never draws it") || !strings.Contains(out, "plays it") {
		t.Fatalf("code %d\n%s", code, out)
	}
	if code, _, errText := runFake(t, &fake{}, false, "record"); code != 2 || !strings.Contains(errText, "record") {
		t.Fatalf("code %d %s", code, errText)
	}
}

func TestParseStatus(t *testing.T) {
	for line, want := range map[string]state{
		"recording 1:05":       {word: "recording", clock: "1:05"},
		"paused 0:01 focus":    {word: "paused", clock: "0:01", focus: true},
		"stopping focus":       {word: "stopping", focus: true},
		"idle":                 {word: "idle"},
		"idle focus":           {word: "idle", focus: true},
		"idle Could not start": {word: "idle", err: "Could not start"},
	} {
		got, ok := parseStatus(line)
		if !ok || got != want {
			t.Fatalf("%q: %+v %v", line, got, ok)
		}
	}
	for _, line := range []string{"", "bye", "busy", "/home/ryan/Videos/a.mp4"} {
		if _, ok := parseStatus(line); ok {
			t.Fatalf("%q parsed", line)
		}
	}
}

func key(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// press sends one key and runs the command it returns, if that command is a rec command.
func press(t *testing.T, m model, s string) model {
	t.Helper()
	next, cmd := m.Update(key(s))
	m = next.(model)
	if cmd == nil {
		return m
	}
	if msg, ok := cmd().(done); ok {
		next, _ = m.Update(msg)
		m = next.(model)
	}
	return m
}

func TestTheScreenListsTheActions(t *testing.T) {
	m := testModel(func(string) Result { return Result{Out: "idle"} })
	m.working = ""
	text := m.View()
	for _, line := range []string{"> " + bold + "start" + reset, "  pause", "  resume", "  stop and save, camera stays open", "  fill the recording with you", "  save and close", "status"} {
		if !strings.Contains(text, line) {
			t.Fatalf("missing %q\n%s", line, text)
		}
	}
	if strings.Contains(text, "1      start") || strings.Contains(text, "ctrl+c") || strings.Contains(text, "recordings") {
		t.Fatalf("shortcut page or empty recordings:\n%s", text)
	}
	m = press(t, m, "j")
	if !strings.Contains(m.View(), "> "+bold+"pause"+reset) || m.cursor != 1 {
		t.Fatalf("j:\n%s", m.View())
	}
	m = press(t, m, "k")
	m = press(t, m, "k")
	if !strings.Contains(m.View(), "> "+bold+"save and close"+reset) || m.cursor != 5 {
		t.Fatalf("k wraps:\n%s", m.View())
	}
	m = press(t, m, "j")
	if m.cursor != 0 {
		t.Fatalf("j wraps: %d", m.cursor)
	}
}

func TestEnterRunsTheSelectedAction(t *testing.T) {
	var asked []string
	replies := map[string]string{"start": "recording 0:00", "focus": "recording 0:03 focus", "stop": "/home/ryan/Videos/c.mp4", "pause": "busy"}
	m := testModel(func(command string) Result {
		asked = append(asked, command)
		return Result{Out: replies[command]}
	})
	m.working = ""
	m = press(t, m, "enter")
	if text := m.View(); !strings.Contains(text, "recording 0:00") || !strings.Contains(text, "in the corner") {
		t.Fatalf("after start:\n%s", text)
	}
	for range 4 {
		m = press(t, m, "j")
	}
	m = press(t, m, "enter")
	if text := m.View(); !strings.Contains(text, "fills the screen") || !strings.Contains(text, "put the camera back in the corner") {
		t.Fatalf("after you:\n%s", text)
	}
	for range 3 {
		m = press(t, m, "k")
	}
	m = press(t, m, "enter")
	if !strings.Contains(m.View(), "busy") {
		t.Fatalf("after pause:\n%s", m.View())
	}
	m = press(t, m, "j")
	m = press(t, m, "j")
	m = press(t, m, "enter")
	if !strings.Contains(m.View(), "saved   /home/ryan/Videos/c.mp4") {
		t.Fatalf("after stop:\n%s", m.View())
	}
	m = press(t, m, "1")
	if strings.Join(asked, " ") != "start focus pause stop" {
		t.Fatalf("asked %v", asked)
	}
}

func TestAFailedCommandShowsItsText(t *testing.T) {
	m := testModel(func(string) Result { return Result{Err: notOpen, Code: 1} })
	m = press(t, m, "j")
	m = press(t, m, "j")
	m = press(t, m, "enter")
	if text := m.View(); !strings.Contains(text, "error   ") || !strings.Contains(text, notOpen) || !strings.Contains(text, "closed") {
		t.Fatalf("%s", text)
	}
	for _, color := range []string{"\x1b[31", "\x1b[32", "\x1b[33", "\x1b[36", "\x1b[9"} {
		if strings.Contains(m.View(), color) {
			t.Fatalf("color %q", color)
		}
	}
}

func TestQuitSavesThenLeaves(t *testing.T) {
	var asked []string
	m := testModel(func(command string) Result {
		asked = append(asked, command)
		return Result{Out: "/home/ryan/Videos/d.mp4"}
	})
	m.cursor = 5
	next, cmd := m.Update(key("enter"))
	m = next.(model)
	if !m.quitting() || cmd == nil {
		t.Fatalf("enter on save and close did not quit")
	}
	next, cmd = m.Update(cmd())
	m = next.(model)
	if cmd == nil || m.final.Out != "/home/ryan/Videos/d.mp4" || strings.Join(asked, " ") != "quit" {
		t.Fatalf("final %+v asked %v", m.final, asked)
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("save and close did not leave")
	}
}

func TestQLeavesTheCameraAsItIs(t *testing.T) {
	var asked []string
	m := testModel(func(command string) Result {
		asked = append(asked, command)
		return Result{Out: "recording 0:01"}
	})
	for _, s := range []string{"q", "ctrl+c"} {
		next, cmd := m.Update(key(s))
		got := next.(model)
		if got.quitting() || len(asked) != 0 || cmd == nil {
			t.Fatalf("%s quitting %v asked %v", s, got.quitting(), asked)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%s did not leave", s)
		}
	}
}

func TestStatusPollsOnTheTickAndOnR(t *testing.T) {
	polls := 0
	m := testModel(func(command string) Result {
		if command == "status" {
			polls++
		}
		return Result{Out: "paused 0:07"}
	})
	m.working = ""
	for _, s := range []string{"x", "j", "k", "1"} {
		m = press(t, m, s)
	}
	if polls != 0 {
		t.Fatalf("polled on a key: %d", polls)
	}
	next, cmd := m.Update(tick{})
	m = next.(model)
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("tick %T", cmd())
	}
	for _, c := range batch {
		if c == nil {
			continue
		}
		if msg, ok := tryRun(c).(polled); ok {
			next, _ = m.Update(msg)
			m = next.(model)
		}
	}
	if polls != 1 || !strings.Contains(m.View(), "paused 0:07") {
		t.Fatalf("polls %d\n%s", polls, m.View())
	}
	_, cmd = m.Update(key("r"))
	if msg, ok := cmd().(polled); !ok || msg.res.Out != "paused 0:07" || polls != 2 {
		t.Fatalf("r: %v polls %d", msg, polls)
	}
}

// tryRun runs a command unless it is the one-second tick, which would block the test.
func tryRun(c tea.Cmd) tea.Msg {
	ch := make(chan tea.Msg, 1)
	go func() { ch <- c() }()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(200 * time.Millisecond):
		return nil
	}
}

func TestSendTalksToTheSocket(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	if _, err := (live{}).Send("status", time.Second); err == nil {
		t.Fatal("a missing socket answered")
	}
	ln, err := net.Listen("unix", filepath.Join(dir, "rec.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		line, _ := bufio.NewReader(conn).ReadString('\n')
		if line == "status\n" {
			_, _ = conn.Write([]byte("recording 0:09 focus\n"))
		}
	}()
	reply, err := (live{}).Send("status", time.Second)
	if err != nil || reply != "recording 0:09 focus" {
		t.Fatalf("%q %v", reply, err)
	}
}

func writeRecording(t *testing.T, dir, name string, mod time.Time) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func TestScanListsTheNewestRecordings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OMARCHY_SCREENRECORD_DIR", dir)
	base := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	writeRecording(t, dir, "screenrecording-2026-11-19_01-02-03.mp4", base)
	writeRecording(t, dir, "screenrecording-2026-11-20_08-18-49.mp4", base.Add(time.Hour))
	writeRecording(t, dir, "screenrecording-2026-11-20_08-18-49-processed.mp4", base.Add(2*time.Hour))
	writeRecording(t, dir, "notes.mp4", base.Add(3*time.Hour))
	writeRecording(t, dir, "screenrecording-weird.mp4", base.Add(30*time.Minute))
	if err := os.Mkdir(filepath.Join(dir, "screenrecording-dir.mp4"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := scanVideos()
	if len(got) != 3 {
		t.Fatalf("%+v", got)
	}
	if got[0].label != "2026-11-20 08:18:49" || got[1].label != "weird" || got[2].label != "2026-11-19 01:02:03" {
		t.Fatalf("%+v", got)
	}
}

func TestScanKeepsTwentyNewest(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OMARCHY_SCREENRECORD_DIR", dir)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 21 {
		name := fmt.Sprintf("screenrecording-2026-01-%02d_00-00-00.mp4", i+1)
		writeRecording(t, dir, name, base.Add(time.Duration(i)*time.Hour))
	}
	got := scanVideos()
	if len(got) != 20 || got[0].label != "2026-01-21 00:00:00" || got[19].label != "2026-01-02 00:00:00" {
		t.Fatalf("len %d first %s last %s", len(got), got[0].label, got[len(got)-1].label)
	}
}

func TestVideosDirPrefersTheScreenrecordDir(t *testing.T) {
	t.Setenv("OMARCHY_SCREENRECORD_DIR", "/tmp/recs")
	t.Setenv("XDG_VIDEOS_DIR", "/tmp/xdg")
	if videosDir() != "/tmp/recs" {
		t.Fatalf("%s", videosDir())
	}
}

func TestVideosDirReadsUserDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OMARCHY_SCREENRECORD_DIR", "")
	t.Setenv("XDG_VIDEOS_DIR", "")
	if err := os.MkdirAll(filepath.Join(home, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "XDG_VIDEOS_DIR=\"$HOME/Movies\"\n"
	if err := os.WriteFile(filepath.Join(home, ".config", "user-dirs.dirs"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := videosDir(); got != filepath.Join(home, "Movies") {
		t.Fatalf("%s", got)
	}
}

func TestEnterPlaysTheNewerRecording(t *testing.T) {
	var played []string
	play := func(path string) error {
		played = append(played, path)
		return nil
	}
	scan := recordings(
		recording{path: "/v/new.mp4", label: "newer"},
		recording{path: "/v/old.mp4", label: "older"},
	)
	var asked []string
	m := newModel(func(command string) Result {
		asked = append(asked, command)
		return Result{Out: "idle"}
	}, scan, play)
	m.working = ""
	if !strings.Contains(m.View(), dim+"recordings"+reset) || !strings.Contains(m.View(), "  newer") {
		t.Fatalf("list:\n%s", m.View())
	}
	m = press(t, m, "k")
	if m.cursor != 7 || !strings.Contains(m.View(), "> "+bold+"older"+reset) {
		t.Fatalf("k wrap:\n%s", m.View())
	}
	m = press(t, m, "j")
	if m.cursor != 0 {
		t.Fatalf("j wrap: %d", m.cursor)
	}
	for range 6 {
		m = press(t, m, "j")
	}
	m = press(t, m, "1")
	m = press(t, m, "enter")
	if len(played) != 1 || played[0] != "/v/new.mp4" || len(asked) != 0 || m.kind != "playing" || m.note != "newer" {
		t.Fatalf("played %v asked %v kind %s note %s", played, asked, m.kind, m.note)
	}
	next, cmd := m.Update(key("q"))
	if len(played) != 1 || len(asked) != 0 || next.(model).quitting() || cmd == nil {
		t.Fatalf("q played %v asked %v", played, asked)
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q did not leave")
	}
}

func TestAMissingPlayerShowsTheError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	scan := recordings(recording{path: "/v/new.mp4", label: "newer"})
	var asked []string
	m := newModel(func(command string) Result {
		asked = append(asked, command)
		return Result{Out: "idle"}
	}, scan, playWithMpv)
	for range 6 {
		m = press(t, m, "j")
	}
	m = press(t, m, "enter")
	if m.kind != "error" || !strings.Contains(m.note, "mpv") || len(asked) != 0 {
		t.Fatalf("kind %s note %s asked %v", m.kind, m.note, asked)
	}
	if !strings.Contains(m.View(), bold+m.note+reset) {
		t.Fatalf("%s", m.View())
	}
}

func TestTheListRefreshesAfterAPollAndASave(t *testing.T) {
	show := false
	scan := func() []recording {
		if !show {
			return nil
		}
		return []recording{{path: "/v/new.mp4", label: "2026-11-20 08:18:49"}}
	}
	m := newModel(func(string) Result { return Result{Out: "idle"} }, scan, playWithMpv)
	m.working = ""
	if strings.Contains(m.View(), "recordings") {
		t.Fatal("showed recordings too early")
	}
	show = true
	next, cmd := m.Update(key("r"))
	m = next.(model)
	msg, ok := cmd().(polled)
	if !ok {
		t.Fatalf("r: %T", cmd())
	}
	next, _ = m.Update(msg)
	m = next.(model)
	if !strings.Contains(m.View(), "2026-11-20 08:18:49") {
		t.Fatalf("after r:\n%s", m.View())
	}
	show = false
	m.reloadRecordings()
	if strings.Contains(m.View(), "recordings") {
		t.Fatal("reload kept a gone recording")
	}
	show = true
	m.seq = 1
	next, _ = m.Update(done{seq: 1, command: "stop", res: Result{Out: "/v/new.mp4"}})
	m = next.(model)
	if !strings.Contains(m.View(), "recordings") || m.cursor != 0 {
		t.Fatalf("after save cursor %d\n%s", m.cursor, m.View())
	}
	m.cursor = 6
	show = false
	m.reloadRecordings()
	if m.cursor != 5 {
		t.Fatalf("clamp %d", m.cursor)
	}
}
