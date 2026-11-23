package looptab

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	// reloadTimeout bounds one looptab now.
	reloadTimeout = 10 * time.Second
	// placeWait bounds the last session write when the screen quits.
	placeWait = 2 * time.Second
)

// answer is what one looptab now gave the worker.
type answer struct {
	rows []job
	err  error
}

// reloader is the screen's one serial worker. It runs looptab now off the
// key loop, one at a time, and hands each answer to the loop.
type reloader struct {
	asks    chan struct{}
	answers chan answer
	cancel  context.CancelFunc
}

func startReloader(load func(context.Context) ([]job, error), timeout time.Duration) *reloader {
	ctx, cancel := context.WithCancel(context.Background())
	r := &reloader{asks: make(chan struct{}, 1), answers: make(chan answer), cancel: cancel}
	go r.serve(ctx, load, timeout)
	return r
}

func (r *reloader) serve(ctx context.Context, load func(context.Context) ([]job, error), timeout time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.asks:
		}
		rows, err := loadWithin(ctx, load, timeout)
		select {
		case r.answers <- answer{rows, err}:
		case <-ctx.Done():
			return
		}
	}
}

// loadWithin runs load with the timeout and names a timeout in its error.
func loadWithin(ctx context.Context, load func(context.Context) ([]job, error), timeout time.Duration) ([]job, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	rows, err := load(ctx)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("looptab now timed out after %s", timeout)
	}
	return rows, err
}

// ask queues one reload. It is false when one is already queued: that one
// starts after this change and reads it too.
func (r *reloader) ask() bool {
	select {
	case r.asks <- struct{}{}:
		return true
	default:
		return false
	}
}

// stop drops the queued reload and cancels the one running.
func (r *reloader) stop() { r.cancel() }

// placer is the c3po screen session that keeps where the screen was.
type placer interface {
	Place(path, cursor string) error
	Close() error
}

// placeWriter is the only caller of the session once the loop runs. It
// writes the place off the key loop: the newest job id wins, an id equal to
// the last one written is skipped, and Close runs after the last write.
type placeWriter struct {
	next chan string
	done chan struct{}
}

func startPlaceWriter(s placer, last string) *placeWriter {
	p := &placeWriter{next: make(chan string, 1), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		for id := range p.next {
			if id != last && s.Place("", id) == nil {
				last = id
			}
		}
		_ = s.Close()
	}()
	return p
}

// note hands id to the writer in place of any id it has not taken yet.
// Only the key loop calls it, so the send after the drain never blocks.
func (p *placeWriter) note(id string) {
	select {
	case <-p.next:
	default:
	}
	p.next <- id
}

// finish takes no more ids. The writer still writes the newest one.
func (p *placeWriter) finish() { close(p.next) }

// wait is true once the last write and Close are done, giving up after d.
func (p *placeWriter) wait(d time.Duration) bool {
	if d <= 0 {
		select {
		case <-p.done:
			return true
		default:
			return false
		}
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-p.done:
		return true
	case <-t.C:
		return false
	}
}
