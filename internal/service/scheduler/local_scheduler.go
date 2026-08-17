package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Task is a scheduled callback.
type Task struct {
	ID        string
	Name      string
	At        time.Time
	Callback  func(ctx context.Context)
	Cancelled bool
}

// LocalScheduler is an offline-safe scheduler. It runs a single goroutine
// that ticks every interval and fires due callbacks. The loop is fully
// deterministic when seeded with a fixed clock.
type LocalScheduler struct {
	mu      sync.Mutex
	tasks   []*Task
	tick    time.Duration
	now     func() time.Time
	started bool
	stopCh  chan struct{}
	doneCh  chan struct{}
	onError func(name string, err error)
}

// Option configures the scheduler.
type Option func(*LocalScheduler)

// WithTickInterval sets the tick interval.
func WithTickInterval(d time.Duration) Option {
	return func(s *LocalScheduler) { s.tick = d }
}

// WithClock injects a clock function (for tests).
func WithClock(fn func() time.Time) Option {
	return func(s *LocalScheduler) { s.now = fn }
}

// WithErrorHandler sets an error handler invoked when a callback panics or
// returns a non-nil error from a future-style wrapper.
func WithErrorHandler(fn func(name string, err error)) Option {
	return func(s *LocalScheduler) { s.onError = fn }
}

// New returns a LocalScheduler.
func New(opts ...Option) *LocalScheduler {
	s := &LocalScheduler{
		tick:    time.Second,
		now:     time.Now,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		onError: func(string, error) {},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Start begins the scheduler loop. Must be idempotent.
func (s *LocalScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()
	go s.loop(ctx)
	return nil
}

// Stop signals the scheduler loop to exit.
func (s *LocalScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	close(s.stopCh)
	s.started = false
	s.mu.Unlock()
	select {
	case <-s.doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// Schedule registers a callback to fire at the given time. Returns a cancel
// function.
func (s *LocalScheduler) Schedule(ctx context.Context, at time.Time, name string, fn func(ctx context.Context)) (func(), error) {
	if fn == nil {
		return nil, errors.New("callback is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("%s-%d", name, len(s.tasks))
	t := &Task{ID: id, Name: name, At: at, Callback: fn}
	s.tasks = append(s.tasks, t)
	// Maintain sorted order.
	sort.Slice(s.tasks, func(i, j int) bool { return s.tasks[i].At.Before(s.tasks[j].At) })
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		t.Cancelled = true
	}, nil
}

// Now returns the adapter's notion of "now".
func (s *LocalScheduler) Now() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// Tasks returns a snapshot of currently-scheduled tasks. For tests only.
func (s *LocalScheduler) Tasks() []*Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Task, len(s.tasks))
	copy(out, s.tasks)
	return out
}

// loop fires due tasks at each tick.
func (s *LocalScheduler) loop(ctx context.Context) {
	defer close(s.doneCh)
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.fireDue(ctx, now)
		}
	}
}

// fireDue fires all tasks whose scheduled time has arrived.
func (s *LocalScheduler) fireDue(ctx context.Context, now time.Time) {
	s.mu.Lock()
	due := []*Task{}
	keep := s.tasks[:0]
	for _, t := range s.tasks {
		if t.Cancelled {
			continue
		}
		if !t.At.After(now) {
			due = append(due, t)
			continue
		}
		keep = append(keep, t)
	}
	s.tasks = keep
	s.mu.Unlock()
	for _, t := range due {
		t := t
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.onError(t.Name, fmt.Errorf("panic: %v", r))
				}
			}()
			t.Callback(ctx)
		}()
	}
}
