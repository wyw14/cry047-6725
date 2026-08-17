package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestLocalScheduler_FiresCallback verifies the scheduler invokes the callback
// at the scheduled time.
func TestLocalScheduler_FiresCallback(t *testing.T) {
	now := time.Now()
	mockNow := now
	s := New(
		WithTickInterval(10*time.Millisecond),
		WithClock(func() time.Time { return mockNow }),
	)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop(context.Background())

	var fired atomic.Int32
	var wg sync.WaitGroup
	wg.Add(1)
	cancel, err := s.Schedule(context.Background(), mockNow.Add(50*time.Millisecond), "fire-once", func(ctx context.Context) {
		fired.Add(1)
		wg.Done()
	})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	_ = cancel

	// Advance mock time past the scheduled time.
	go func() {
		time.Sleep(20 * time.Millisecond)
		mockNow = mockNow.Add(100 * time.Millisecond)
	}()

	// Wait for callback to fire or timeout.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("callback did not fire in time")
	}
	if fired.Load() != 1 {
		t.Errorf("expected 1 fire, got %d", fired.Load())
	}
}

// TestLocalScheduler_Cancel verifies that cancelling a task prevents it from
// firing.
func TestLocalScheduler_Cancel(t *testing.T) {
	now := time.Now()
	mockNow := now
	s := New(
		WithTickInterval(10*time.Millisecond),
		WithClock(func() time.Time { return mockNow }),
	)
	_ = s.Start(context.Background())
	defer s.Stop(context.Background())

	var fired atomic.Int32
	cancel, _ := s.Schedule(context.Background(), mockNow.Add(50*time.Millisecond), "to-cancel", func(ctx context.Context) {
		fired.Add(1)
	})
	cancel()
	go func() {
		time.Sleep(20 * time.Millisecond)
		mockNow = mockNow.Add(200 * time.Millisecond)
	}()
	time.Sleep(100 * time.Millisecond)
	if fired.Load() != 0 {
		t.Errorf("expected 0 fires after cancel, got %d", fired.Load())
	}
}

// TestLocalScheduler_Now returns the configured clock's time.
func TestLocalScheduler_Now(t *testing.T) {
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := New(WithClock(func() time.Time { return fixed }))
	if !s.Now().Equal(fixed) {
		t.Errorf("expected %v, got %v", fixed, s.Now())
	}
}
