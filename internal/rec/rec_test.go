package rec

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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
	if code != 0 || !strings.Contains(out, "r2d2 rec") || !strings.Contains(out, "Super+R") || !strings.Contains(out, "never draws it") {
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

func TestShortcutsStayHiddenUntilQuestionMark(t *testing.T) {
	var asked []string
	m := newModel(func(command string) Result {
		asked = append(asked, command)
		return Result{Out: "recording 0:00"}
	})
	if text := m.View(); strings.Contains(text, "ctrl+c") || strings.Contains(text, "1      start") {
		t.Fatalf("shortcuts on the screen:\n%s", text)
	}
	m = press(t, m, "?")
	if text := m.View(); !strings.Contains(text, "1      start") || !strings.Contains(text, "space  start or stop") || strings.Contains(text, "status") {
		t.Fatalf("shortcuts:\n%s", text)
	}
	for _, s := range []string{"1", "2", "4", "5", "6", " ", "r"} {
		m = press(t, m, s)
	}
	if len(asked) != 0 || m.quitting {
		t.Fatalf("keys acted under the shortcut list: %v", asked)
	}
	m = press(t, m, "?")
	if text := m.View(); strings.Contains(text, "1      start") || !strings.Contains(text, "status") {
		t.Fatalf("after the second ?:\n%s", text)
	}
}

func TestKeysSendTheCommands(t *testing.T) {
	var asked []string
	replies := map[string]string{"start": "recording 0:00", "focus": "recording 0:03 focus", "stop": "/home/ryan/Videos/c.mp4", "pause": "busy"}
	m := newModel(func(command string) Result {
		asked = append(asked, command)
		return Result{Out: replies[command]}
	})
	m.working = ""
	m = press(t, m, "1")
	if text := m.View(); !strings.Contains(text, "recording 0:00") || !strings.Contains(text, "in the corner") {
		t.Fatalf("after 1:\n%s", text)
	}
	m = press(t, m, "5")
	if text := m.View(); !strings.Contains(text, "fills the screen") {
		t.Fatalf("after 5:\n%s", text)
	}
	m = press(t, m, "2")
	if text := m.View(); !strings.Contains(text, "busy") {
		t.Fatalf("after 2:\n%s", text)
	}
	m = press(t, m, "4")
	if text := m.View(); !strings.Contains(text, "saved   /home/ryan/Videos/c.mp4") {
		t.Fatalf("after 4:\n%s", text)
	}
	m = press(t, m, "j")
	if strings.Join(asked, " ") != "start focus pause stop" {
		t.Fatalf("asked %v", asked)
	}
}

func TestAFailedCommandShowsItsText(t *testing.T) {
	m := newModel(func(string) Result { return Result{Err: notOpen, Code: 1} })
	m = press(t, m, "3")
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
	for _, s := range []string{"q", "6"} {
		var asked []string
		m := newModel(func(command string) Result {
			asked = append(asked, command)
			return Result{Out: "/home/ryan/Videos/d.mp4"}
		})
		next, cmd := m.Update(key(s))
		m = next.(model)
		if !m.quitting || cmd == nil {
			t.Fatalf("%s did not quit", s)
		}
		next, cmd = m.Update(cmd())
		m = next.(model)
		if cmd == nil || m.final.Out != "/home/ryan/Videos/d.mp4" || strings.Join(asked, " ") != "quit" {
			t.Fatalf("%s: final %+v asked %v", s, m.final, asked)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%s did not leave", s)
		}
	}
}

func TestStatusPollsOnTheTickAndOnR(t *testing.T) {
	polls := 0
	m := newModel(func(command string) Result {
		if command == "status" {
			polls++
		}
		return Result{Out: "paused 0:07"}
	})
	m.working = ""
	for _, s := range []string{"x", "j", "?", "?"} {
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
