package notifier

import (
	"context"
	"testing"

	"github.com/cry047/baseline/internal/domain"
)

func TestLocalNotifier_Records(t *testing.T) {
	n := New(3, "")
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := n.Notify(ctx, &domain.Notification{
			ID:     "n-" + string(rune('a'+i)),
			UserID: "u1",
			Type:   domain.NotificationAssigned,
			Title:  "t",
		}); err != nil {
			t.Fatalf("notify: %v", err)
		}
	}
	if n.Len() != 3 {
		t.Errorf("expected ring to cap at 3, got %d", n.Len())
	}
	recent := n.Recent(2)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent, got %d", len(recent))
	}
	// Most recent should be the last inserted (e/d).
	if recent[0].ID != "n-e" {
		t.Errorf("expected most-recent n-e, got %s", recent[0].ID)
	}
}
