package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ryangerardwilson/nebula/s3"
)

// live reads systemd, nebula s3, nebula keep, and nebula run. It only reads.
type live struct{}

const unitProperties = "LoadState,ActiveState,SubState,Result,ExecMainStatus,LastTriggerUSec,NextElapseUSecRealtime,NextElapseUSecMonotonic"

func (live) Unit(ctx context.Context, name string) (Unit, error) {
	out, err := exec.CommandContext(ctx, "systemctl", "--user", "show", name, "--timestamp=unix", "-p", unitProperties).Output()
	if err != nil {
		return Unit{}, fmt.Errorf("systemctl show %s: %w", name, err)
	}
	return parseUnit(string(out)), nil
}

func (live) Stat(ctx context.Context, bucket, key string) (time.Time, error) {
	mtime, _, err := s3.Stat(ctx, bucket, key)
	if errors.Is(err, s3.ErrNotFound) {
		return time.Time{}, ErrMissing
	}
	return mtime, err
}

// Kept reads nebula keep status --json. A nebula that predates keep exits 2
// with its usage, which is ErrNoKeep.
func (live) Kept(ctx context.Context) ([]Copy, error) {
	out, err := nebula(ctx, "keep", "status", "--json")
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 2 {
		return nil, ErrNoKeep
	}
	if err != nil {
		return nil, err
	}
	return parseKept(out)
}

// Hosted runs nebula run list.
func (live) Hosted(ctx context.Context) ([]Process, error) {
	out, err := nebula(ctx, "run", "list")
	if err != nil {
		return nil, err
	}
	return parseHosted(out), nil
}

// nebula runs one read-only nebula command in its own process group, so a
// timeout also stops the ssh it starts. A failure names the command and
// carries nebula's stderr.
func nebula(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "nebula", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), nil
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	name := "nebula " + strings.Join(args, " ")
	if text := strings.TrimSpace(stderr.String()); text != "" {
		return "", commandError{name + ": " + text, err}
	}
	return "", fmt.Errorf("%s: %w", name, err)
}

// commandError is a failed command's stderr. It unwraps to the exit error.
type commandError struct {
	text string
	err  error
}

func (e commandError) Error() string { return e.text }

func (e commandError) Unwrap() error { return e.err }

// keepRecord is one object of nebula keep status --json. Only the fields the
// screen judges pass this edge.
type keepRecord struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	DirtySince string `json:"dirty_since"`
	KeptAt     string `json:"kept_at"`
	Error      string `json:"error"`
}

// parseKept reads the JSON array nebula keep status --json prints. Its
// stamps are RFC 3339 in UTC.
func parseKept(out string) ([]Copy, error) {
	var records []keepRecord
	if err := json.Unmarshal([]byte(out), &records); err != nil {
		return nil, fmt.Errorf("nebula keep status: %w", err)
	}
	copies := make([]Copy, 0, len(records))
	for _, r := range records {
		copies = append(copies, Copy{
			Name:       r.Name,
			State:      r.State,
			DirtySince: rfcStamp(r.DirtySince),
			KeptAt:     rfcStamp(r.KeptAt),
			Error:      r.Error,
		})
	}
	return copies, nil
}

func rfcStamp(value string) time.Time {
	stamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return stamp
}

// parseHosted reads the name, hostname, port, and release columns of
// nebula run list. "no apps" is an empty list.
func parseHosted(out string) []Process {
	var processes []Process
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.TrimSpace(line) == "no apps" {
			continue
		}
		p := Process{Name: fields[0]}
		if len(fields) >= 4 {
			p.Host, p.Port, p.Release = fields[1], fields[2], strings.Join(fields[3:], " ")
		}
		processes = append(processes, p)
	}
	return processes
}

// parseUnit reads systemctl show output taken with --timestamp=unix.
func parseUnit(out string) Unit {
	values := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		key, value, found := strings.Cut(line, "=")
		if found {
			values[key] = strings.TrimSpace(value)
		}
	}
	mono := values["NextElapseUSecMonotonic"]
	return Unit{
		Load:        values["LoadState"],
		Active:      values["ActiveState"],
		Sub:         values["SubState"],
		Result:      values["Result"],
		Status:      values["ExecMainStatus"],
		LastTrigger: unixStamp(values["LastTriggerUSec"]),
		Scheduled:   values["NextElapseUSecRealtime"] != "" || mono != "" && mono != "infinity" && mono != "0",
	}
}

func unixStamp(value string) time.Time {
	seconds, err := strconv.ParseInt(strings.TrimPrefix(value, "@"), 10, 64)
	if err != nil || seconds <= 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0)
}
