package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TestSchedulerCreatesTodoForEachDueOccurrence verifies the core fix: when a
// maintenance plan becomes overdue again on a later cycle (its NextDueDate
// has advanced), the scheduler generates a SECOND responsible-person todo
// instead of showing only the original one.
func TestSchedulerCreatesTodoForEachDueOccurrence(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, -5))
	scheduler := NewSchedulerService(ports)
	if _, err := scheduler.Run(context.Background()); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	current, err := ports.Plans.GetPlan(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	current.NextDueDate = now.AddDate(0, 0, -1)
	current.Version++
	if err := ports.Plans.UpdatePlan(context.Background(), current); err != nil {
		t.Fatalf("advance occurrence: %v", err)
	}
	if _, err := scheduler.Run(context.Background()); err != nil {
		t.Fatalf("second scan: %v", err)
	}
	todos, err := ports.Todos.ListByUser(context.Background(), personID, domain.PageQuery{Limit: 20})
	if err != nil {
		t.Fatalf("list todos: %v", err)
	}
	if len(todos.Items) != 2 {
		t.Fatalf("each due occurrence needs its own todo, got %d", len(todos.Items))
	}
	// The two todos must be for distinct occurrences (distinct idempotency
	// keys), proving they are not accidental duplicates of one reminder.
	if todos.Items[0].IdempotencyKey == todos.Items[1].IdempotencyKey {
		t.Fatalf("todos share an idempotency key: %q", todos.Items[0].IdempotencyKey)
	}
}

// TestScheduler_PerOccurrenceIdempotency verifies the two requirements together:
// every overdue occurrence gets its own todo, and re-scanning the same
// occurrence (or a later occurrence without further change) never duplicates.
func TestScheduler_PerOccurrenceIdempotency(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, -5))
	scheduler := NewSchedulerService(ports)

	// Scan #1: first overdue occurrence -> one todo.
	if _, err := scheduler.Run(context.Background()); err != nil {
		t.Fatalf("scan 1: %v", err)
	}
	// Scan #2 (no change): same occurrence -> must NOT duplicate.
	if _, err := scheduler.Run(context.Background()); err != nil {
		t.Fatalf("scan 2: %v", err)
	}
	// Advance to a new overdue occurrence and scan again.
	current, err := ports.Plans.GetPlan(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	current.NextDueDate = now.AddDate(0, 0, -2)
	current.Version++
	if err := ports.Plans.UpdatePlan(context.Background(), current); err != nil {
		t.Fatalf("advance occurrence: %v", err)
	}
	if _, err := scheduler.Run(context.Background()); err != nil {
		t.Fatalf("scan 3: %v", err)
	}
	// Scan #4 (no change after the new occurrence): again no duplicate.
	if _, err := scheduler.Run(context.Background()); err != nil {
		t.Fatalf("scan 4: %v", err)
	}
	todos, err := ports.Todos.ListByUser(context.Background(), personID, domain.PageQuery{Limit: 20})
	if err != nil {
		t.Fatalf("list todos: %v", err)
	}
	if len(todos.Items) != 2 {
		t.Fatalf("expected 2 todos (one per occurrence) after 4 scans, got %d", len(todos.Items))
	}
}

// TestScheduler_PerOccurrenceNotification verifies that each overdue occurrence
// surfaces exactly one reminder notification to the responsible person, and
// that re-scanning the same occurrence does not duplicate it.
func TestScheduler_PerOccurrenceNotification(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, -5))
	scheduler := NewSchedulerService(ports)

	// First overdue occurrence -> one reminder.
	if _, err := scheduler.Run(context.Background()); err != nil {
		t.Fatalf("scan 1: %v", err)
	}
	// Re-scan same occurrence -> no new reminder.
	if _, err := scheduler.Run(context.Background()); err != nil {
		t.Fatalf("scan 2: %v", err)
	}
	if got := countOverdueNotifs(t, ports, personID); got != 1 {
		t.Fatalf("expected 1 overdue notification after re-scan, got %d", got)
	}

	// New overdue occurrence -> one more reminder.
	current, err := ports.Plans.GetPlan(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	current.NextDueDate = now.AddDate(0, 0, -1)
	current.Version++
	if err := ports.Plans.UpdatePlan(context.Background(), current); err != nil {
		t.Fatalf("advance occurrence: %v", err)
	}
	if _, err := scheduler.Run(context.Background()); err != nil {
		t.Fatalf("scan 3: %v", err)
	}
	if got := countOverdueNotifs(t, ports, personID); got != 2 {
		t.Fatalf("expected 2 overdue notifications (one per occurrence), got %d", got)
	}
}

// countOverdueNotifs counts maintenance_overdue notifications for a user.
func countOverdueNotifs(t *testing.T, ports *domain.Ports, userID string) int {
	t.Helper()
	notifs, err := ports.Notifications.ListByUser(context.Background(), userID, domain.PageQuery{Limit: 50})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	n := 0
	for _, item := range notifs.Items {
		if item.Type == domain.NotificationMaintenanceOverdue {
			n++
		}
	}
	return n
}
