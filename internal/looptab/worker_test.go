package looptab

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSession is a session whose writes the test can hold. Its first Place
// signals started and waits for gate.
type fakeSession struct {
	mu      sync.Mutex
	calls   []string
	started chan struct{}
	gate    chan struct{}
	once    sync.Once
}

func newSession() *fakeSession {
	return &fakeSession{started: make(chan struct{}), gate: make(chan struct{})}
}

func (s *fakeSession) Place(path, cursor string) error {
	s.record("place " + path + "/" + cursor)
	first := false
	s.once.Do(func() { first = true })
	if first {
		close(s.started)
		<-s.gate
	}
	return nil
}

func (s *fakeSession) Close() error {
	s.record("close")
	return nil
}

func (s *fakeSession) record(call string) {
	s.mu.Lock()
	s.calls = append(s.calls, call)
	s.mu.Unlock()
}

func (s *fakeSession) log() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.calls, ",")
}

func waitClosed(t *testing.T, ch chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s did not happen", what)
	}
}

// A reload that hangs after a run leaves the list painted and the keys live.
func TestASlowReloadDoesNotBlockThePaintOrTheKeys(t *testing.T) {
	var loads atomic.Int32
	release := make(chan struct{})
	var ran []int
	m := &model{
		load: func(ctx context.Context) ([]job, error) {
			if loads.Add(1) == 1 {
				return twoJobs(), nil
			}
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return append(twoJobs(), job{Index: 2, ID: "feed0001", Schedule: "hourly", Kind: "command", Action: "c3po gmail tidy"}), nil
		},
		run: func(j job) error {
			ran = append(ran, j.Index)
			return nil
		},
	}
	screen := newScreen(80, 24,
		step{want: "> daily 5am", key: "enter"},
		step{want: "looptab" + styleReset + "  " + styleMuted + "loading", key: "j"},
		step{want: "> hourly at 15", key: "?"},
		step{want: "enter  run now", key: "?"},
		step{want: "> hourly at 15", key: "enter"},
		step{want: "still loading", key: "k", before: func() { close(release) }},
		step{want: "c3po gmail tidy", key: "q"},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(screen.last(), "> daily 5am") {
		t.Fatalf("the answer moved the cursor off its job\n%s", screen.last())
	}
	if len(ran) != 1 || ran[0] != 0 || loads.Load() != 2 {
		t.Fatalf("ran %v loads %d", ran, loads.Load())
	}
	if screen.showed("> hourly at 15", "loading", "still loading") == "" {
		t.Fatal("no frame moved the cursor while the reload hung")
	}
}

func TestAHungReloadTimesOut(t *testing.T) {
	m := &model{
		timeout: 20 * time.Millisecond,
		load: func(ctx context.Context) ([]job, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	screen := newScreen(80, 24, step{want: "looptab now timed out after 20ms", key: "q"})
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if last := screen.last(); strings.Contains(last, "loading") || strings.Contains(last, "no jobs") {
		t.Fatalf("last\n%s", last)
	}
}

// Keys move the cursor while a session write is held. The writer keeps only
// the newest job, skips a job it already wrote, and closes after the last.
func TestThePlaceIsWrittenOffTheLoopOnlyWhenItChanges(t *testing.T) {
	sess := newSession()
	m := &model{load: listed(twoJobs()), session: sess}
	wait := func() { waitClosed(t, sess.started, "the first place write") }
	screen := newScreen(80, 24,
		step{want: "> daily 5am", key: "j", before: wait},
		step{want: "> hourly at 15", key: "k"},
		step{want: "> daily 5am", key: "j"},
		step{want: "> hourly at 15", key: "?"},
		step{want: "enter  run now", key: "?"},
		step{want: "> hourly at 15", key: "q", before: func() { close(sess.gate) }},
	)
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if got := sess.log(); got != "place /abcd1234,place /00abcdef,close" {
		t.Fatalf("session calls %s", got)
	}
}

func TestQuitGivesUpOnASlowPlaceWrite(t *testing.T) {
	sess := newSession()
	defer close(sess.gate)
	m := &model{load: listed(twoJobs()), session: sess, saveWait: 20 * time.Millisecond}
	screen := newScreen(80, 24, step{want: "> daily 5am", key: "q", before: func() {
		waitClosed(t, sess.started, "the first place write")
	}})
	start := time.Now()
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("quit waited %s on a held write", time.Since(start))
	}
	if last := screen.last(); !strings.Contains(last, "saving place") {
		t.Fatalf("last\n%s", last)
	}
}

func TestAResumedScreenStartsOnItsJobWithoutRewritingIt(t *testing.T) {
	sess := newSession()
	close(sess.gate)
	m := &model{load: listed(twoJobs()), session: sess, selected: "00abcdef"}
	screen := newScreen(80, 24, step{want: "> hourly at 15", key: "q"})
	if err := m.loop(screen); err != nil {
		t.Fatal(err)
	}
	if got := sess.log(); got != "close" {
		t.Fatalf("session calls %s", got)
	}
}

// The worker runs one looptab now at a time and folds a second ask into the
// one already queued.
func TestReloadsRunOneAtATime(t *testing.T) {
	var running, most atomic.Int32
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	r := startReloader(func(context.Context) ([]job, error) {
		n := running.Add(1)
		if n > most.Load() {
			most.Store(n)
		}
		started <- struct{}{}
		<-release
		running.Add(-1)
		return twoJobs(), nil
	}, time.Second)
	defer r.stop()
	if !r.ask() {
		t.Fatal("first ask")
	}
	<-started
	if !r.ask() || r.ask() {
		t.Fatal("a second ask is queued once")
	}
	close(release)
	for range 2 {
		select {
		case a := <-r.answers:
			if a.err != nil || len(a.rows) != 2 {
				t.Fatalf("%+v", a)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("no answer")
		}
	}
	if most.Load() != 1 {
		t.Fatalf("%d loads ran at once", most.Load())
	}
}
