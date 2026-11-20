package rec

import (
	"bytes"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Widget reaches the rec camera widget, the GTK window in the corner of the recording.
type Widget interface {
	// Send writes one command to the widget socket and returns the reply line.
	// An error means the widget is not open.
	Send(command string, timeout time.Duration) (string, error)
	// Start runs the rec program with REC_INTERNAL=1 in its own session and does not wait.
	Start(args []string) error
	// Internal runs the rec program with REC_INTERNAL=1 and waits.
	Internal(args []string) (stdout, stderr string, code int)
}

// Result is what one rec command prints and its exit code.
type Result struct {
	Out  string
	Err  string
	Code int
}

// Client runs rec commands against the widget.
type Client struct {
	Widget Widget
	Sleep  func(time.Duration)
	// Flags are --size and --device for a widget that r2d2 starts.
	Flags []string
}

const (
	step     = 100 * time.Millisecond
	openWait = 10 * time.Second
	takeWait = 30 * time.Second
)

const notOpen = "rec is not open"

// Do runs one command. A closed widget is started for open, start, toggle, and focus.
func (c Client) Do(command string) Result {
	wire := command
	if command == "open" {
		wire = "raise"
	}
	reply, err := c.Widget.Send(wire, timeout(command))
	if err == nil {
		if command == "open" {
			return Result{}
		}
		return Result{Out: reply}
	}
	switch command {
	case "quit":
		// The widget is gone. rec quit still saves a recorder that is running.
		out, errText, code := c.Widget.Internal(c.args("quit"))
		return Result{Out: strings.TrimSpace(out), Err: strings.TrimSpace(errText), Code: code}
	case "status", "pause", "resume", "stop":
		return Result{Err: notOpen, Code: 1}
	}
	return c.open(command)
}

func (c Client) open(command string) Result {
	if err := c.Widget.Start(c.args(command)); err != nil {
		return Result{Err: "rec: " + err.Error(), Code: 1}
	}
	status, ok := c.await(openWait, func(string) bool { return true })
	if !ok {
		return Result{Err: "rec did not open", Code: 1}
	}
	switch command {
	case "open":
		return Result{}
	case "start", "toggle":
		// The widget starts the take after it is placed. Report it once it settles.
		status, _ = c.await(takeWait, settled)
	}
	return Result{Out: status}
}

// await asks for the status until done accepts it or wait runs out.
func (c Client) await(wait time.Duration, done func(string) bool) (string, bool) {
	last, seen := "", false
	for spent := time.Duration(0); spent <= wait; spent += step {
		if status, err := c.Widget.Send("status", 2*time.Second); err == nil {
			last, seen = status, true
			if done(status) {
				return status, true
			}
		}
		c.sleep(step)
	}
	return last, seen
}

func (c Client) sleep(d time.Duration) {
	if c.Sleep != nil {
		c.Sleep(d)
		return
	}
	time.Sleep(d)
}

func (c Client) args(command string) []string {
	return append(append([]string{}, c.Flags...), command)
}

// settled is true once a take has started or failed to start.
func settled(status string) bool {
	return status != "idle" && !strings.HasPrefix(status, "starting")
}

func timeout(command string) time.Duration {
	switch command {
	case "start", "toggle", "stop", "quit":
		return 600 * time.Second
	}
	return 2 * time.Second
}

// SocketPath is where the widget listens. rec.py uses the same path.
func SocketPath() string {
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = filepath.Join("/tmp", "rec-"+strconv.Itoa(os.Getuid()))
	}
	return filepath.Join(runtime, "rec.sock")
}

type live struct{}

func (live) Send(command string, timeout time.Duration) (string, error) {
	conn, err := net.DialTimeout("unix", SocketPath(), timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}
	if _, err := conn.Write([]byte(command + "\n")); err != nil {
		return "", err
	}
	var data []byte
	buf := make([]byte, 4096)
	for !bytes.Contains(data, []byte("\n")) {
		n, err := conn.Read(buf)
		data = append(data, buf[:n]...)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return "", err
			}
			break
		}
	}
	return strings.TrimSpace(string(data)), nil
}

func (live) Start(args []string) error {
	cmd := internalCommand(args)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	// Errors from the widget go to a log beside the socket, not to this terminal.
	if log, err := os.Create(filepath.Join(filepath.Dir(SocketPath()), "rec.log")); err == nil {
		cmd.Stdout, cmd.Stderr = log, log
		defer log.Close()
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func (live) Internal(args []string) (string, string, int) {
	cmd := internalCommand(args)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return out.String(), errb.String(), 0
	case errors.As(err, &exit):
		return out.String(), errb.String(), exit.ExitCode()
	default:
		return out.String(), "rec: " + err.Error(), 1
	}
}

// internalCommand runs the rec launcher with REC_INTERNAL=1, so it runs rec.py and does not call r2d2.
func internalCommand(args []string) *exec.Cmd {
	cmd := exec.Command(program(), args...)
	cmd.Env = append(os.Environ(), "REC_INTERNAL=1")
	return cmd
}

func program() string {
	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, ".local", "bin", "rec")
		if info, err := os.Stat(path); err == nil && info.Mode()&0o111 != 0 {
			return path
		}
	}
	if path, err := exec.LookPath("rec"); err == nil {
		return path
	}
	return "rec"
}
