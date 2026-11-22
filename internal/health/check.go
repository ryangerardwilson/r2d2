// Package health is the r2d2 health screen. It reads units, nebula s3
// objects, and nebula run, and never changes any of them.
package health

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// State is the word a row shows.
type State string

const (
	OK      State = "ok"
	Waiting State = "waiting"
	Down    State = "down"
)

// Row is one check.
type Row struct {
	Name   string
	State  State
	Reason string
}

// Unit is what systemctl reports for one unit.
type Unit struct {
	Load        string
	Active      string
	Sub         string
	Result      string
	Status      string
	LastTrigger time.Time
	Scheduled   bool
}

// Process is one process nebula hosts. Host, Port, and Release are empty
// when nebula did not list them.
type Process struct {
	Name    string
	Host    string
	Port    string
	Release string
}

// Checker reads what the screen shows. It must not change anything.
type Checker interface {
	Unit(ctx context.Context, name string) (Unit, error)
	Stat(ctx context.Context, bucket, key string) (time.Time, error)
	Keys(ctx context.Context, bucket string) ([]string, error)
	Hosted(ctx context.Context) ([]Process, error)
}

// timeout bounds each systemctl, s3, and ssh call. A caller's earlier deadline wins.
const timeout = 5 * time.Second

var errTimeout = errors.New("timed out")

// ErrMissing is what a Checker's Stat returns when the object does not exist.
var ErrMissing = errors.New("missing")

// Check runs every check at once and returns the rows in catalog order,
// then one row per hosted process.
func Check(ctx context.Context, c Checker, now time.Time) []Row {
	rows := make([]Row, len(catalog))
	var hosted []Row
	var wg sync.WaitGroup
	for i, item := range catalog {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows[i] = item.run(ctx, c, now)
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		hosted = hostedRows(ctx, c)
	}()
	wg.Wait()
	return append(rows, hosted...)
}

// bounded returns when fn does or when the timeout passes, whichever is
// first, so a hung call cannot hold the screen.
func bounded[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	type result struct {
		value T
		err   error
	}
	done := make(chan result, 1)
	go func() {
		value, err := fn(ctx)
		done <- result{value, err}
	}()
	select {
	case r := <-done:
		return r.value, r.err
	case <-ctx.Done():
		var zero T
		return zero, errTimeout
	}
}

func (c check) run(ctx context.Context, checker Checker, now time.Time) Row {
	switch c.kind {
	case service:
		return serviceRow(ctx, checker, c)
	case timer:
		return timerRow(ctx, checker, c, now)
	case boot:
		return bootRow(ctx, checker, c)
	case backupDay:
		return dayRow(ctx, checker, c, now)
	default:
		return objectRow(ctx, checker, c, now)
	}
}

func unit(ctx context.Context, c Checker, name string) (Unit, error) {
	u, err := bounded(ctx, func(ctx context.Context) (Unit, error) { return c.Unit(ctx, name) })
	if err == nil && u.Load != "loaded" {
		err = fmt.Errorf("%s is %s", name, orUnknown(u.Load))
	}
	return u, err
}

func serviceRow(ctx context.Context, c Checker, item check) Row {
	u, err := unit(ctx, c, item.name)
	if err != nil {
		return down(item, err)
	}
	if u.Active == "active" && u.Sub == "running" {
		return Row{item.name, OK, "running"}
	}
	return Row{item.name, Down, orUnknown(u.Active) + "/" + orUnknown(u.Sub) + failure(u)}
}

func timerRow(ctx context.Context, c Checker, item check, now time.Time) Row {
	t, err := unit(ctx, c, item.name)
	if err != nil {
		return down(item, err)
	}
	s, err := unit(ctx, c, item.unit)
	if err != nil {
		return down(item, err)
	}
	if t.Active != "active" {
		return Row{item.name, Down, "timer is " + orUnknown(t.Active)}
	}
	if failed(s) {
		return Row{item.name, Down, "last run failed" + failure(s)}
	}
	if t.LastTrigger.IsZero() {
		if t.Scheduled {
			return Row{item.name, Waiting, "not run yet, next run scheduled"}
		}
		return Row{item.name, Down, "never run and nothing scheduled"}
	}
	age := now.Sub(t.LastTrigger)
	if age > 2*item.every {
		return Row{item.name, Down, "last run " + ago(age) + ", every " + span(item.every)}
	}
	return Row{item.name, OK, "last run " + ago(age)}
}

func bootRow(ctx context.Context, c Checker, item check) Row {
	u, err := unit(ctx, c, item.name)
	if err != nil {
		return down(item, err)
	}
	if failed(u) {
		return Row{item.name, Down, "last run failed" + failure(u)}
	}
	if u.Active == "activating" {
		return Row{item.name, OK, "running"}
	}
	return Row{item.name, OK, "last run succeeded"}
}

func objectRow(ctx context.Context, c Checker, item check, now time.Time) Row {
	mtime, err := bounded(ctx, func(ctx context.Context) (time.Time, error) {
		return c.Stat(ctx, item.bucket, item.key)
	})
	if errors.Is(err, ErrMissing) {
		if item.kind == skippable {
			return Row{item.title(), Waiting, "missing; " + item.why}
		}
		return Row{item.title(), Down, "missing"}
	}
	if err != nil {
		return down(item, err)
	}
	age := now.Sub(mtime)
	if item.kind == fresh && age > 2*item.every {
		return Row{item.title(), Down, "written " + ago(age) + ", every " + span(item.every)}
	}
	return Row{item.title(), OK, "written " + ago(age)}
}

func dayRow(ctx context.Context, c Checker, item check, now time.Time) Row {
	keys, err := bounded(ctx, func(ctx context.Context) ([]string, error) { return c.Keys(ctx, item.bucket) })
	if err != nil {
		return down(item, err)
	}
	newest, name := newestDay(keys)
	if newest.IsZero() {
		return Row{item.title(), Down, "no backup day"}
	}
	if now.Sub(newest) > 2*item.every {
		return Row{item.title(), Down, "newest day " + name + ", every " + span(item.every)}
	}
	return Row{item.title(), OK, "newest day " + name}
}

// newestDay reads the UTC day prefix of each key, as sql backup names them.
func newestDay(keys []string) (time.Time, string) {
	var newest time.Time
	var name string
	for _, key := range keys {
		prefix, _, _ := strings.Cut(key, "/")
		day, err := time.ParseInLocation("2006-01-02", prefix, time.UTC)
		if err == nil && day.After(newest) {
			newest, name = day, prefix
		}
	}
	return newest, name
}

func hostedRows(ctx context.Context, c Checker) []Row {
	processes, err := bounded(ctx, c.Hosted)
	if err != nil {
		return []Row{{"nebula run", Down, reason(err)}}
	}
	if len(processes) == 0 {
		return []Row{{"nebula run", OK, "no hosted processes"}}
	}
	rows := make([]Row, 0, len(processes))
	for _, p := range processes {
		rows = append(rows, Row{"run " + p.Name, OK, hostedReason(p)})
	}
	return rows
}

func hostedReason(p Process) string {
	if p.Port == "" {
		return "listed"
	}
	return p.Host + " port " + p.Port + ", release " + p.Release
}

func failed(u Unit) bool {
	return u.Active == "failed" || u.Result != "" && u.Result != "success"
}

func failure(u Unit) string {
	if u.Result == "" || u.Result == "success" {
		return ""
	}
	if u.Status != "" && u.Status != "0" {
		return " (" + u.Result + ", status " + u.Status + ")"
	}
	return " (" + u.Result + ")"
}

func down(item check, err error) Row {
	return Row{item.title(), Down, reason(err)}
}

func reason(err error) string {
	if errors.Is(err, errTimeout) || errors.Is(err, context.DeadlineExceeded) {
		return "timed out after " + span(timeout)
	}
	text, _, _ := strings.Cut(strings.TrimSpace(err.Error()), "\n")
	if len(text) > 120 {
		text = text[:120]
	}
	return text
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

func ago(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	return span(d) + " ago"
}

func span(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d/time.Second))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	case d < 2*day:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	default:
		return fmt.Sprintf("%dd", int(d/day))
	}
}
