package health

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var now = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

type fake struct {
	units    map[string]Unit
	objects  map[string]time.Time
	hang     map[string]bool
	kept     []Copy
	failKeep error
	hangKeep bool
	hosted   string
	failRun  error
	calls    atomic.Int32
}

func (f *fake) Unit(_ context.Context, name string) (Unit, error) {
	f.calls.Add(1)
	if u, ok := f.units[name]; ok {
		return u, nil
	}
	return Unit{Load: "not-found", Active: "inactive", Sub: "dead"}, nil
}

func (f *fake) Stat(ctx context.Context, bucket, key string) (time.Time, error) {
	if f.hang[bucket+"/"+key] {
		<-ctx.Done()
		time.Sleep(time.Second)
		return time.Time{}, ctx.Err()
	}
	if mtime, ok := f.objects[bucket+"/"+key]; ok {
		return mtime, nil
	}
	return time.Time{}, fmt.Errorf("s3://%s/%s: %w", bucket, key, ErrMissing)
}

func (f *fake) Kept(ctx context.Context) ([]Copy, error) {
	if f.hangKeep {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.kept, f.failKeep
}

func (f *fake) Hosted(context.Context) ([]Process, error) {
	if f.failRun != nil {
		return nil, f.failRun
	}
	return parseHosted(f.hosted), nil
}

func running() Unit { return Unit{Load: "loaded", Active: "active", Sub: "running", Result: "success"} }

func fired(ago time.Duration) Unit {
	return Unit{Load: "loaded", Active: "active", Sub: "waiting", Result: "success", LastTrigger: now.Add(-ago), Scheduled: true}
}

func succeeded() Unit {
	return Unit{Load: "loaded", Active: "inactive", Sub: "dead", Result: "success", Status: "0"}
}

// healthy is every catalog check passing.
func healthy() *fake {
	f := &fake{units: map[string]Unit{}, objects: map[string]time.Time{}}
	for _, item := range catalog {
		switch item.kind {
		case service:
			f.units[item.name] = running()
		case timer:
			f.units[item.name] = fired(item.every / 2)
			f.units[item.unit] = succeeded()
		case boot:
			f.units[item.name] = succeeded()
		default:
			f.objects[item.bucket+"/"+item.key] = now.Add(-time.Minute)
		}
	}
	f.kept = []Copy{{Name: "c3po", State: "ok", KeptAt: now.Add(-3 * time.Minute)}}
	f.hosted = "site             site.example.com                          8080  r42\n"
	return f
}

func rowNamed(t *testing.T, rows []Row, name string) Row {
	t.Helper()
	for _, row := range rows {
		if row.Name == name {
			return row
		}
	}
	t.Fatalf("no row %q in %v", name, rows)
	return Row{}
}

func TestEveryCheckIsOkWhenHealthy(t *testing.T) {
	rows := Check(context.Background(), healthy(), now)
	if len(rows) != len(catalog)+2 {
		t.Fatalf("%d rows", len(rows))
	}
	for _, row := range rows {
		if row.State != OK {
			t.Fatalf("%+v", row)
		}
	}
	if got := rowNamed(t, rows, "run site"); !strings.Contains(got.Reason, "site.example.com port 8080, release r42") {
		t.Fatalf("%+v", got)
	}
	if got := rowNamed(t, rows, "keep c3po"); got.Reason != "kept 3m ago" {
		t.Fatalf("%+v", got)
	}
}

// sqlx-backup.timer and s3://sqlx-backups were retired. The daily dump is
// nebula-sql-backup.timer and stays on the host.
func TestTheRetiredBackupRowsAreGone(t *testing.T) {
	rows := Check(context.Background(), healthy(), now)
	for _, row := range rows {
		if strings.Contains(row.Name, "sqlx-backup") {
			t.Fatalf("retired row %+v", row)
		}
	}
	if got := rowNamed(t, rows, "nebula-sql-backup.timer"); got.State != OK {
		t.Fatalf("%+v", got)
	}
}

func TestKeepRows(t *testing.T) {
	f := healthy()
	f.kept = []Copy{
		{Name: "c3po", State: "ok", KeptAt: now.Add(-3 * time.Minute)},
		{Name: "crusade", State: "ok"},
		{Name: "musica", State: "waiting", DirtySince: now.Add(-2 * time.Minute)},
		{Name: "tesseract", State: "waiting", DirtySince: now.Add(-20 * time.Minute)},
		{Name: "antibloat", State: "failing", DirtySince: now.Add(-time.Hour), Error: "put s3://backups/antibloat.sqlite: ssh: connect to host del-vc2 port 22: Connection refused\nmore"},
		{Name: "odd", State: "stuck"},
	}
	rows := Check(context.Background(), f, now)
	want := map[string]Row{
		"keep c3po":      {"keep c3po", OK, "kept 3m ago"},
		"keep crusade":   {"keep crusade", OK, "up to date"},
		"keep musica":    {"keep musica", OK, "unkept for 2m"},
		"keep tesseract": {"keep tesseract", Waiting, "unkept for 20m"},
		"keep antibloat": {"keep antibloat", Down, "put s3://backups/antibloat.sqlite: ssh: connect to host del-vc2 port 22: Connection refused"},
		"keep odd":       {"keep odd", Down, "keep state is stuck"},
	}
	for name, row := range want {
		if got := rowNamed(t, rows, name); got != row {
			t.Fatalf("%+v, want %+v", got, row)
		}
	}
	if got := Plain(rows); !strings.Contains(got, "health  2 down") {
		t.Fatalf("%s", got)
	}
}

func TestKeepWithoutDatabasesIsOneOkRow(t *testing.T) {
	f := healthy()
	f.kept = nil
	if got := rowNamed(t, Check(context.Background(), f, now), "nebula keep"); got.State != OK || got.Reason != "no databases kept" {
		t.Fatalf("%+v", got)
	}
}

// An installed nebula that predates keep does not make the screen fail.
func TestKeepUnavailableIsOneWaitingRow(t *testing.T) {
	f := healthy()
	f.failKeep = ErrNoKeep
	rows := Check(context.Background(), f, now)
	if len(rows) != len(catalog)+2 {
		t.Fatalf("%d rows", len(rows))
	}
	if got := rowNamed(t, rows, "nebula keep"); got.State != Waiting || !strings.Contains(got.Reason, "keep status is unavailable") {
		t.Fatalf("%+v", got)
	}
	if downCount(rows) != 0 {
		t.Fatalf("%s", Plain(rows))
	}
}

func TestKeepFailureIsDown(t *testing.T) {
	f := healthy()
	f.failKeep = errors.New("nebula keep status --json: keep record c3po: unexpected end of JSON input")
	if got := rowNamed(t, Check(context.Background(), f, now), "nebula keep"); got.State != Down || !strings.Contains(got.Reason, "keep record c3po") {
		t.Fatalf("%+v", got)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	f = healthy()
	f.hangKeep = true
	if got := rowNamed(t, Check(ctx, f, now), "nebula keep"); got.State != Down || !strings.Contains(got.Reason, "timed out") {
		t.Fatalf("%+v", got)
	}
}

func TestUnits(t *testing.T) {
	f := healthy()
	f.units["looptab.service"] = Unit{Load: "loaded", Active: "inactive", Sub: "dead", Result: "success"}
	f.units["gdrive.timer"] = fired(3 * time.Hour)
	f.units["nebula-keep.timer"] = Unit{Load: "loaded", Active: "active", Sub: "waiting", Result: "success", Scheduled: true}
	f.units["genesys.service"] = Unit{Load: "loaded", Active: "failed", Sub: "failed", Result: "exit-code", Status: "1"}
	f.units["tesseract-restore.service"] = Unit{Load: "loaded", Active: "failed", Sub: "failed", Result: "exit-code", Status: "2"}
	delete(f.units, "antibloat-observer.service")
	rows := Check(context.Background(), f, now)
	want := map[string]State{
		"tunnel.service":             OK,
		"looptab.service":            Down,
		"antibloat-observer.service": Down,
		"gmail.timer":                OK,
		"gdrive.timer":               Down,
		"nebula-keep.timer":          Waiting,
		"genesys.timer":              Down,
		"tesseract-restore.service":  Down,
	}
	for name, state := range want {
		if got := rowNamed(t, rows, name); got.State != state {
			t.Fatalf("%s: %+v, want %s", name, got, state)
		}
	}
	if got := rowNamed(t, rows, "genesys.timer"); got.Reason != "last run failed (exit-code, status 1)" {
		t.Fatalf("%+v", got)
	}
	if got := rowNamed(t, rows, "gdrive.timer"); got.Reason != "last run 3h ago, every 1h" {
		t.Fatalf("%+v", got)
	}
}

func TestTimerNeverRunAndNothingScheduledIsDown(t *testing.T) {
	f := healthy()
	f.units["gmail.timer"] = Unit{Load: "loaded", Active: "active", Sub: "waiting", Result: "success"}
	if got := rowNamed(t, Check(context.Background(), f, now), "gmail.timer"); got.State != Down {
		t.Fatalf("%+v", got)
	}
}

func TestObjects(t *testing.T) {
	f := healthy()
	f.objects["backups/tesseract.sqlite"] = now.Add(-2 * time.Hour)
	f.objects["backups/musica.sqlite"] = now.Add(-30 * day)
	delete(f.objects, "backups/antibloat.sqlite")
	delete(f.objects, "backups/crusade.sqlite")
	rows := Check(context.Background(), f, now)
	want := map[string]State{
		"s3://backups/c3po.sqlite":      OK,
		"s3://backups/tesseract.sqlite": Down,
		"s3://backups/musica.sqlite":    OK,
		"s3://backups/antibloat.sqlite": Down,
		"s3://backups/crusade.sqlite":   Waiting,
	}
	for name, state := range want {
		if got := rowNamed(t, rows, name); got.State != state {
			t.Fatalf("%s: %+v, want %s", name, got, state)
		}
	}
	if got := rowNamed(t, rows, "s3://backups/antibloat.sqlite"); got.Reason != "missing" {
		t.Fatalf("%+v", got)
	}
}

func TestStatTimeoutIsDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	f := healthy()
	f.hang = map[string]bool{"backups/c3po.sqlite": true, "backups/crusade.sqlite": true}
	start := time.Now()
	rows := Check(ctx, f, now)
	if time.Since(start) > 500*time.Millisecond {
		t.Fatalf("waited %s for a hung stat", time.Since(start))
	}
	for _, name := range []string{"s3://backups/c3po.sqlite", "s3://backups/crusade.sqlite"} {
		if got := rowNamed(t, rows, name); got.State != Down || !strings.Contains(got.Reason, "timed out") {
			t.Fatalf("%+v", got)
		}
	}
	if got := rowNamed(t, rows, "tunnel.service"); got.State != OK {
		t.Fatalf("%+v", got)
	}
}

func TestNebulaRun(t *testing.T) {
	f := healthy()
	f.hosted = "no apps\n"
	if got := rowNamed(t, Check(context.Background(), f, now), "nebula run"); got.State != OK || got.Reason != "no hosted processes" {
		t.Fatalf("%+v", got)
	}
	f.failRun = errors.New("nebula run list: ssh: connect to host del-vc2 port 22: Connection refused")
	if got := rowNamed(t, Check(context.Background(), f, now), "nebula run"); got.State != Down || !strings.Contains(got.Reason, "Connection refused") {
		t.Fatalf("%+v", got)
	}
}

func TestParseUnitReadsUnixStamps(t *testing.T) {
	u := parseUnit("LoadState=loaded\nActiveState=active\nSubState=waiting\nResult=success\nLastTriggerUSec=@1795132105\nNextElapseUSecRealtime=\nNextElapseUSecMonotonic=22h 3min\n")
	if u.Load != "loaded" || u.Active != "active" || u.LastTrigger.Unix() != 1795132105 || !u.Scheduled {
		t.Fatalf("%+v", u)
	}
	u = parseUnit("LoadState=loaded\nLastTriggerUSec=\nNextElapseUSecRealtime=\nNextElapseUSecMonotonic=infinity\n")
	if !u.LastTrigger.IsZero() || u.Scheduled {
		t.Fatalf("%+v", u)
	}
}

func TestPipePrintsRowsAndExits0(t *testing.T) {
	f := healthy()
	f.units["genesys.service"] = Unit{Load: "loaded", Active: "failed", Result: "exit-code", Status: "1"}
	var out, errb bytes.Buffer
	if code := run(nil, &out, &errb, f, func() time.Time { return now }, false); code != 0 {
		t.Fatalf("code %d %s", code, errb.String())
	}
	text := out.String()
	for _, want := range []string{"health  1 down", "tunnel.service", "genesys.timer", "down", "s3://backups/crusade.sqlite", "run site"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q\n%s", want, text)
		}
	}
	if strings.Contains(text, "\x1b[") {
		t.Fatalf("paint on a pipe\n%q", text)
	}
}

func TestHelpAndExtraArguments(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"help"}, &out, &errb, healthy(), time.Now, false); code != 0 || !strings.Contains(out.String(), "never restarts") {
		t.Fatalf("code %d\n%s", code, out.String())
	}
	if code := run([]string{"now"}, &out, &errb, healthy(), time.Now, false); code != 2 {
		t.Fatalf("code %d", code)
	}
}

func loadedModel(f *fake) model {
	m := newModel(func() []Row { return Check(context.Background(), f, now) })
	next, _ := m.Update(m.Init()())
	return next.(model)
}

func press(m model, keys ...string) (model, tea.Cmd) {
	var cmd tea.Cmd
	for _, k := range keys {
		var next tea.Model
		next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		m = next.(model)
	}
	return m, cmd
}

func TestRReloadsAndOtherKeysDoNot(t *testing.T) {
	f := healthy()
	m := loadedModel(f)
	before := f.calls.Load()
	m, cmd := press(m, "j", "k", "g", "G", "?", "?")
	if cmd != nil || f.calls.Load() != before {
		t.Fatalf("checked on a move: %d -> %d", before, f.calls.Load())
	}
	f.units["tunnel.service"] = Unit{Load: "loaded", Active: "failed", Sub: "failed", Result: "exit-code"}
	m, cmd = press(m, "r")
	if cmd == nil || !m.loading || !strings.Contains(m.View(), "checking") {
		t.Fatalf("r did not reload\n%s", m.View())
	}
	next, _ := m.Update(cmd())
	m = next.(model)
	if f.calls.Load() == before || !strings.Contains(m.View(), "health  1 down") {
		t.Fatalf("%s", m.View())
	}
}

func TestStaleLoadIsIgnored(t *testing.T) {
	m := loadedModel(healthy())
	m, _ = press(m, "r")
	next, _ := m.Update(loaded{gen: m.gen - 1, rows: []Row{{"old", Down, "x"}}})
	if strings.Contains(next.(model).View(), "old") {
		t.Fatal("an older load replaced the rows")
	}
}

func TestShortcutsHiddenUntilQuestionMark(t *testing.T) {
	m := loadedModel(healthy())
	if strings.Contains(m.View(), "reload") || strings.Contains(m.View(), "j/k") {
		t.Fatalf("shortcuts on the screen\n%s", m.View())
	}
	m, _ = press(m, "?")
	if !strings.Contains(m.View(), "r      reload") || strings.Contains(m.View(), "tunnel.service") {
		t.Fatalf("%s", m.View())
	}
	m, _ = press(m, "j")
	m, _ = press(m, "?")
	if m.cursor != 0 || strings.Contains(m.View(), "reload") || !strings.Contains(m.View(), "> tunnel.service") {
		t.Fatalf("cursor %d\n%s", m.cursor, m.View())
	}
}

func TestKeysWrapAndJump(t *testing.T) {
	m := loadedModel(healthy())
	last := len(m.rows) - 1
	m, _ = press(m, "k")
	if m.cursor != last {
		t.Fatalf("k on the first row: %d", m.cursor)
	}
	m, _ = press(m, "j")
	if m.cursor != 0 {
		t.Fatalf("j on the last row: %d", m.cursor)
	}
	m, _ = press(m, "G")
	if m.cursor != last {
		t.Fatalf("G %d", m.cursor)
	}
	m, _ = press(m, "g")
	if m.cursor != 0 {
		t.Fatalf("g %d", m.cursor)
	}
	_, cmd := press(m, "q")
	if cmd == nil {
		t.Fatal("q did not quit")
	}
}

func TestPaintIsBoldAndDimOnly(t *testing.T) {
	m := loadedModel(healthy())
	m.rows = []Row{{"a", OK, "fine"}, {"b", Waiting, "soon"}, {"c", Down, "broken"}}
	view := m.View()
	if !strings.Contains(view, "\x1b[2m  b") || !strings.Contains(view, "\x1b[1m  c") || strings.Contains(view, "\x1b[0m> a") {
		t.Fatalf("%q", view)
	}
	for _, code := range []string{"\x1b[3", "\x1b[9", "38;", "48;"} {
		if strings.Contains(view, code) {
			t.Fatalf("color %q in %q", code, view)
		}
	}
}

// lockedBuffer is the program's output, read while the program writes it.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func waitOutput(t *testing.T, out *lockedBuffer, text string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(out.String(), text) {
		if time.Now().After(deadline) {
			t.Fatalf("no %q in\n%q", text, out.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// The first frame and the keys do not wait for the check, which runs in a
// tea.Cmd.
func TestTheScreenPaintsBeforeTheCheckReturns(t *testing.T) {
	release := make(chan struct{})
	var checks atomic.Int32
	m := newModel(func() []Row {
		checks.Add(1)
		<-release
		return []Row{{"tunnel.service", OK, "running"}, {"looptab.service", Down, "inactive/dead"}}
	})
	in, keys := io.Pipe()
	defer keys.Close()
	var out lockedBuffer
	program := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(&out))
	done := make(chan error, 1)
	go func() {
		_, err := program.Run()
		done <- err
	}()
	waitOutput(t, &out, "health  checking")
	if _, err := keys.Write([]byte("?")); err != nil {
		t.Fatal(err)
	}
	waitOutput(t, &out, "r      reload")
	if checks.Load() != 1 {
		t.Fatalf("%d checks", checks.Load())
	}
	close(release)
	if _, err := keys.Write([]byte("?")); err != nil {
		t.Fatal(err)
	}
	waitOutput(t, &out, "health  1 down")
	if _, err := keys.Write([]byte("q")); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("q did not quit")
	}
}

func TestAnIdleObjectKeepCopiesIsNotDown(t *testing.T) {
	f := healthy()
	f.objects["backups/rgw_slack_quickshell_plugin.sqlite"] = now.Add(-6 * time.Hour)
	f.objects["backups/tesseract.sqlite"] = now.Add(-6 * time.Hour)
	f.kept = []Copy{
		{Name: "rgw_slack_quickshell_plugin", State: "ok", KeptAt: now.Add(-6 * time.Hour)},
		{Name: "tesseract", State: "failing", Error: "connection refused"},
	}
	rows := Check(context.Background(), f, now)
	if got := rowNamed(t, rows, "s3://backups/rgw_slack_quickshell_plugin.sqlite"); got.State != OK || !strings.Contains(got.Reason, "keep has every change") {
		t.Fatalf("an idle object keep vouches for: %+v", got)
	}
	if got := rowNamed(t, rows, "s3://backups/tesseract.sqlite"); got.State != Down {
		t.Fatalf("a stale object keep cannot vouch for: %+v", got)
	}
}

func TestAnObjectKeepIsStillUploadingIsNotDown(t *testing.T) {
	f := healthy()
	f.objects["backups/c3po.sqlite"] = now.Add(-3 * time.Hour)
	f.kept = []Copy{{Name: "c3po", State: "waiting", DirtySince: now.Add(-time.Minute)}}
	rows := Check(context.Background(), f, now)
	if got := rowNamed(t, rows, "s3://backups/c3po.sqlite"); got.State != OK {
		t.Fatalf("an idle object whose copy lags a minute: %+v", got)
	}
	if got := rowNamed(t, rows, "keep c3po"); got.State != OK {
		t.Fatalf("%+v", got)
	}
	f.kept = []Copy{{Name: "c3po", State: "waiting", DirtySince: now.Add(-time.Hour)}}
	rows = Check(context.Background(), f, now)
	if got := rowNamed(t, rows, "s3://backups/c3po.sqlite"); got.State != Down {
		t.Fatalf("a copy an hour behind cannot vouch: %+v", got)
	}
}
