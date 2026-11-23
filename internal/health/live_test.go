package health

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeNebula puts a nebula on PATH that answers keep status --json with
// body. PATH holds nothing else, so the real nebula cannot run.
func fakeNebula(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$*\" != \"keep status --json\" ] && [ \"$*\" != \"run list\" ]; then echo \"unexpected nebula $*\" >&2; exit 3; fi\n" +
		body
	if err := os.WriteFile(filepath.Join(dir, "nebula"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestKeptReadsTheJSONStatus(t *testing.T) {
	fakeNebula(t, `printf '%s\n' '[
  {"name": "c3po", "path": "/run/user/1000/c3po/c3po.sqlite", "marked": 4, "kept": 4, "kept_at": "2026-11-20T11:57:00Z", "kept_size": 4096, "source": "4096 1795132105", "state": "ok"},
  {"name": "tesseract", "path": "/run/user/1000/tesseract.sqlite", "marked": 9, "kept": 7, "dirty_since": "2026-11-20T11:40:00Z", "kept_at": "2026-11-20T11:39:00Z", "error": "put: connection refused", "error_at": "2026-11-20T11:41:00Z", "failures": 2, "state": "failing"}
]'
`)
	copies, err := live{}.Kept(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(copies) != 2 {
		t.Fatalf("%+v", copies)
	}
	if got := copies[0]; got.Name != "c3po" || got.State != "ok" || !got.KeptAt.Equal(time.Date(2026, 11, 20, 11, 57, 0, 0, time.UTC)) || !got.DirtySince.IsZero() {
		t.Fatalf("%+v", got)
	}
	if got := copies[1]; got.State != "failing" || got.Error != "put: connection refused" || !got.DirtySince.Equal(time.Date(2026, 11, 20, 11, 40, 0, 0, time.UTC)) {
		t.Fatalf("%+v", got)
	}
	if rows := copyRows(copies, nil, now); rows[0] != (Row{"keep c3po", OK, "kept 3m ago"}) || rows[1].State != Down {
		t.Fatalf("%+v", rows)
	}
}

// The installed nebula may predate keep: it prints its usage and exits 2.
func TestKeptOnANebulaWithoutKeepIsUnavailable(t *testing.T) {
	fakeNebula(t, "echo 'usage: nebula help' >&2\nexit 2\n")
	if _, err := (live{}).Kept(context.Background()); !errors.Is(err, ErrNoKeep) {
		t.Fatalf("err %v", err)
	}
	_, err := (live{}).Kept(context.Background())
	rows := copyRows(nil, err, now)
	if len(rows) != 1 || rows[0].Name != "nebula keep" || rows[0].State != Waiting {
		t.Fatalf("%+v", rows)
	}
}

func TestKeptFailureCarriesStderr(t *testing.T) {
	fakeNebula(t, "echo 'keep record c3po: unexpected end of JSON input' >&2\nexit 1\n")
	_, err := live{}.Kept(context.Background())
	if err == nil || errors.Is(err, ErrNoKeep) || err.Error() != "nebula keep status --json: keep record c3po: unexpected end of JSON input" {
		t.Fatalf("err %v", err)
	}
	fakeNebula(t, "printf 'not json\\n'\n")
	if _, err := (live{}).Kept(context.Background()); err == nil || !strings.HasPrefix(err.Error(), "nebula keep status: ") {
		t.Fatalf("err %v", err)
	}
}

func TestHostedStillReadsRunList(t *testing.T) {
	fakeNebula(t, "printf 'site  site.example.com  8080  r42\\n'\n")
	processes, err := live{}.Hosted(context.Background())
	if err != nil || len(processes) != 1 || processes[0].Port != "8080" {
		t.Fatalf("%+v %v", processes, err)
	}
	fakeNebula(t, "echo 'ssh: connect to host del-vc2 port 22: Connection refused' >&2\nexit 255\n")
	if _, err := (live{}).Hosted(context.Background()); err == nil || err.Error() != "nebula run list: ssh: connect to host del-vc2 port 22: Connection refused" {
		t.Fatalf("err %v", err)
	}
}
