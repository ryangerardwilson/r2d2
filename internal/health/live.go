package health

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ryangerardwilson/nebula/s3x"
)

// live reads systemd, nebula s3, and nebula run. It only reads.
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
	mtime, _, err := s3x.Stat(ctx, bucket, key)
	return mtime, err
}

func (live) Keys(ctx context.Context, bucket string) ([]string, error) {
	objects, err := s3x.List(ctx, bucket, "")
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(objects))
	for _, object := range objects {
		keys = append(keys, object.Key)
	}
	return keys, nil
}

// Hosted runs nebula run list in its own process group, so a timeout
// also stops the ssh it starts.
func (live) Hosted(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "nebula", "run", "list")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if text := strings.TrimSpace(stderr.String()); text != "" {
			return "", errors.New("nebula run list: " + text)
		}
		return "", fmt.Errorf("nebula run list: %w", err)
	}
	return stdout.String(), nil
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
