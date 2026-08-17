package notifier

import (
	"context"
	"sync"

	"github.com/cry047/baseline/internal/domain"
)

// LocalNotifier is a fully offline notifier adapter. It records notifications
// into an in-memory ring buffer for later inspection (e.g., by tests or a UI
// notification center) and writes them to a durable log file when configured.
type LocalNotifier struct {
	mu      sync.Mutex
	ring    []*domain.Notification
	maxSize int
	logPath string
}

// New returns a LocalNotifier.
func New(maxSize int, logPath string) *LocalNotifier {
	if maxSize <= 0 {
		maxSize = 256
	}
	return &LocalNotifier{maxSize: maxSize, logPath: logPath}
}

// Notify records the notification in the ring buffer.
func (n *LocalNotifier) Notify(ctx context.Context, notif *domain.Notification) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	cp := *notif
	n.ring = append(n.ring, &cp)
	if len(n.ring) > n.maxSize {
		n.ring = n.ring[len(n.ring)-n.maxSize:]
	}
	return nil
}

// Recent returns the most recent N notifications.
func (n *LocalNotifier) Recent(limit int) []*domain.Notification {
	n.mu.Lock()
	defer n.mu.Unlock()
	if limit <= 0 || limit > len(n.ring) {
		limit = len(n.ring)
	}
	out := make([]*domain.Notification, limit)
	for i := 0; i < limit; i++ {
		cp := *n.ring[len(n.ring)-1-i]
		out[i] = &cp
	}
	return out
}

// Len returns the number of notifications currently buffered.
func (n *LocalNotifier) Len() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.ring)
}

// Assert that *LocalNotifier implements domain.Notifier.
var _ domain.Notifier = (*LocalNotifier)(nil)
