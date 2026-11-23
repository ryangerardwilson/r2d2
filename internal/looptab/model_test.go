package looptab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// step waits until the newest frame shows want, runs before, then presses key.
type step struct {
	want   string
	key    string
	before func()
}

// fakeScreen plays a script of steps. The loop paints on its goroutine and
// reads keys on another, as on a terminal.
type fakeScreen struct {
	mu      sync.Mutex
	frames  []string
	script  []step
	painted chan struct{}
	width   int
	height  int
}

func newScreen(width, height int, script ...step) *fakeScreen {
	return &fakeScreen{width: width, height: height, script: script, painted: make(chan struct{}, 1)}
}

func (f *fakeScreen) size() (int, int) { return f.width, f.height }

func (f *fakeScreen) paint(frame string) {
	f.mu.Lock()
	f.frames = append(f.frames, frame)
	f.mu.Unlock()
	select {
	case f.painted <- struct{}{}:
	default:
	}
}

func (f *fakeScreen) readKey() (string, error) {
	f.mu.Lock()
	if len(f.script) == 0 {
		f.mu.Unlock()
		return "", io.EOF
	}
	s := f.script[0]
	f.script = f.script[1:]
	f.mu.Unlock()
	deadline := time.After(2 * time.Second)
	for !strings.Contains(f.last(), s.want) {
		select {
		case <-f.painted:
		case <-deadline:
			return "", fmt.Errorf("no frame showed %q before %q; last\n%s", s.want, s.key, f.last())
		}
	}
	if s.before != nil {
		s.before()
	}
	return s.key, nil
}

func (f *fakeScreen) first() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.frames[0]
}

func (f *fakeScreen) last() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.frames) == 0 {
		return ""
	}
	return f.frames[len(f.frames)-1]
}

// showed is the first frame that holds every text, or "".
func (f *fakeScreen) showed(texts ...string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, frame := range f.frames {
		all := true
		for _, text := range texts {
			all = all && strings.Contains(frame, text)
		}
		if all {
			return frame
		}
	}
	return ""
}

func twoJobs() []job {
	return []job{
		{Index: 0, ID: "abcd1234", Schedule: "daily 5am", Kind: "grok", Action: `@grok "Check mail"`},
		{Index: 1, ID: "00abcdef", Schedule: "hourly at 15", Kind: "command", Action: "c3po gmail spam clean"},
	}
}

func listed(rows []job) func(context.Context) ([]job, error) {
	return func(context.Context) ([]job, error) { return rows, nil }
}

func TestJKMovesAcrossJobs(t *testing.T) {
	m := &model{load: listed(twoJobs())}
	screen := newScreen(80, 24,
		step{want: "> daily 5am  grok", key: "j"},
		step{want: "> hourly at 15  command", key: "k"},
		step{want: "> daily 5am  grok", key: "q"},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if first := screen.first(); strings.Contains(first, "j/k move") || !strings.Contains(first, "loading") {
		t.Fatalf("first\n%s", first)
	}
}

func TestJKWrapsTheJobs(t *testing.T) {
	m := &model{rows: twoJobs()}
	if m.handleKey("k") || m.cursor != 1 {
		t.Fatalf("k %d", m.cursor)
	}
	if m.handleKey("j") || m.cursor != 0 {
		t.Fatalf("j %d", m.cursor)
	}
	if m.handleKey("G") || m.cursor != 1 || m.handleKey("g") || m.cursor != 0 {
		t.Fatalf("ends %d", m.cursor)
	}
	empty := &model{}
	empty.handleKey("j")
	empty.handleKey("k")
	if empty.cursor != 0 {
		t.Fatalf("empty %d", empty.cursor)
	}
}

func TestEnterRunsTheSelectedIndex(t *testing.T) {
	var loads atomic.Int32
	var ran []int
	m := &model{
		load: func(context.Context) ([]job, error) {
			loads.Add(1)
			return twoJobs(), nil
		},
		run: func(j job) error {
			ran = append(ran, j.Index)
			return nil
		},
	}
	screen := newScreen(80, 24,
		step{want: "> daily 5am", key: "j"},
		step{want: "> hourly at 15", key: "enter"},
		step{want: "ran 00abcdef", key: "q"},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != 1 {
		t.Fatalf("ran %v", ran)
	}
	if screen.showed("ran 00abcdef", "loading") == "" {
		t.Fatal("the reload after the run did not show loading")
	}
}

func TestQuestionMarkBlocksTheRun(t *testing.T) {
	var ran []int
	m := &model{
		load: listed(twoJobs()),
		run: func(j job) error {
			ran = append(ran, j.Index)
			return nil
		},
	}
	screen := newScreen(80, 24,
		step{want: "> daily 5am", key: "?"},
		step{want: "enter  run now", key: "j"},
		step{want: "enter  run now", key: "enter"},
		step{want: "enter  run now", key: "esc"},
		step{want: "> daily 5am", key: "q"},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 || m.cursor != 0 {
		t.Fatalf("cursor %d ran %v", m.cursor, ran)
	}
	if strings.Contains(screen.first(), "j/k move") || screen.showed("enter", "quit") == "" {
		t.Fatalf("first\n%s", screen.first())
	}
	if last := screen.last(); !strings.Contains(last, "> daily 5am") || strings.Contains(last, "j/k move") {
		t.Fatalf("restored\n%s", last)
	}
}

func TestEnterOnAnEmptyListDoesNotRun(t *testing.T) {
	var ran []int
	m := &model{
		load: listed(nil),
		run: func(j job) error {
			ran = append(ran, j.Index)
			return nil
		},
	}
	screen := newScreen(40, 20,
		step{want: "no jobs", key: "enter"},
		step{want: "no jobs", key: "q"},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 || m.status != "no jobs" {
		t.Fatalf("status %q ran %v", m.status, ran)
	}
}

func TestEnterWhenLooptabIsMissingDoesNotRun(t *testing.T) {
	var ran []int
	m := &model{
		load: func(context.Context) ([]job, error) { return nil, errors.New("looptab is not on PATH") },
		run: func(j job) error {
			ran = append(ran, j.Index)
			return nil
		},
	}
	screen := newScreen(40, 20,
		step{want: "looptab is not on PATH", key: "enter"},
		step{want: "looptab is not on PATH", key: "q"},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 || strings.Contains(screen.last(), "no jobs") {
		t.Fatalf("ran %v\n%s", ran, screen.last())
	}
}

func TestAFailedRunStaysOnTheList(t *testing.T) {
	m := &model{
		load: listed(twoJobs()),
		run:  func(job) error { return errors.New("exit status 1") },
	}
	screen := newScreen(80, 24,
		step{want: "> daily 5am", key: "enter"},
		step{want: "abcd1234 failed", key: "q"},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if last := screen.last(); !strings.Contains(last, styleError+"abcd1234 failed") || !strings.Contains(last, "> daily 5am") {
		t.Fatalf("last\n%q", last)
	}
}

func TestEEditsAndReloads(t *testing.T) {
	var loads atomic.Int32
	edits := 0
	m := &model{
		load: func(context.Context) ([]job, error) {
			if loads.Add(1) == 1 {
				return twoJobs(), nil
			}
			return append(twoJobs(), job{Index: 2, ID: "feed0001", Schedule: "hourly", Kind: "command", Action: "c3po gmail tidy"}), nil
		},
		edit: func() (int, error) {
			edits++
			return 1, nil
		},
	}
	screen := newScreen(80, 24,
		step{want: "> daily 5am", key: "e"},
		step{want: "c3po gmail tidy", key: "q"},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if edits != 1 || loads.Load() != 2 {
		t.Fatalf("edits %d loads %d", edits, loads.Load())
	}
	if !strings.Contains(screen.last(), "installed 1 jobs files") {
		t.Fatalf("last\n%s", screen.last())
	}
}

func TestMissingVimLeavesTheScreenUp(t *testing.T) {
	var loads atomic.Int32
	m := &model{
		load: func(context.Context) ([]job, error) {
			loads.Add(1)
			return twoJobs(), nil
		},
		edit: func() (int, error) { return 0, errNoVim },
	}
	screen := newScreen(80, 24,
		step{want: "> daily 5am", key: "e"},
		step{want: "vim is not on PATH", key: "j"},
		step{want: "> hourly", key: "q"},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if loads.Load() != 1 || m.cursor != 1 {
		t.Fatalf("loads %d cursor %d", loads.Load(), m.cursor)
	}
	if screen.showed("vim is not on PATH", "> daily 5am") == "" {
		t.Fatal("no frame showed the vim error over the list")
	}
}

func TestQuestionMarkBlocksTheEdit(t *testing.T) {
	edits := 0
	m := &model{
		load: listed(twoJobs()),
		edit: func() (int, error) {
			edits++
			return 1, nil
		},
	}
	screen := newScreen(80, 24,
		step{want: "> daily 5am", key: "?"},
		step{want: "e      edit jobs file", key: "e"},
		step{want: "e      edit jobs file", key: "q"},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if edits != 0 || strings.Contains(screen.first(), "edit jobs file") {
		t.Fatalf("edits %d\n%s", edits, screen.first())
	}
}
