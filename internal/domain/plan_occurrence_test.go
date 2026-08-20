package domain

import (
	"testing"
	"time"
)

// TestMaintenanceTodoKey_PerOccurrence verifies the idempotency key for an
// auto-generated maintenance todo is scoped to a single occurrence:
//   - it is stable for the same NextDueDate (so repeated scheduler scans are
//     idempotent);
//   - it changes when NextDueDate advances to a later cycle (so a plan that
//     falls overdue again gets its own todo, not the original reminder).
func TestMaintenanceTodoKey_PerOccurrence(t *testing.T) {
	base := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	plan := &MaintenancePlan{ID: "plan-1", NextDueDate: base}

	first := MaintenanceTodoKey(plan)
	if want := "auto-todo-plan-1-2026-01-10"; first != want {
		t.Fatalf("first key = %q, want %q", first, want)
	}
	// Re-deriving the key for the same occurrence must be stable: this is what
	// makes repeated scheduler scans a no-op for an already-tracked occurrence.
	if got := MaintenanceTodoKey(plan); got != first {
		t.Fatalf("same occurrence yielded unstable key: %q != %q", got, first)
	}
	// A later overdue cycle has a new NextDueDate, so it is a distinct
	// occurrence and must get a distinct key (and therefore a distinct todo).
	plan.NextDueDate = base.AddDate(0, 0, 30)
	second := MaintenanceTodoKey(plan)
	if second == first {
		t.Fatalf("distinct occurrences must yield distinct keys, both %q", first)
	}
	if want := "auto-todo-plan-1-2026-02-09"; second != want {
		t.Fatalf("second key = %q, want %q", second, want)
	}
}

// TestOccurrenceDate_UsesNextDueDate verifies the occurrence is identified by
// the current cycle's due date, not the plan's creation day.
func TestOccurrenceDate_UsesNextDueDate(t *testing.T) {
	due := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	p := MaintenancePlan{ID: "p", NextDueDate: due, CreatedAt: due.AddDate(0, 0, -30)}
	if got := p.OccurrenceDate(); !got.Equal(due) {
		t.Fatalf("OccurrenceDate = %v, want NextDueDate %v", got, due)
	}
}

// TestOccurrenceDate_FallsBackToCreatedAt verifies a plan with no due date yet
// still derives a stable key off its creation day.
func TestOccurrenceDate_FallsBackToCreatedAt(t *testing.T) {
	created := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	p := MaintenancePlan{ID: "p", CreatedAt: created} // NextDueDate is zero
	if got := p.OccurrenceDate(); !got.Equal(created) {
		t.Fatalf("OccurrenceDate = %v, want CreatedAt %v (fallback)", got, created)
	}
}
