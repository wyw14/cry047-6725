package domain

import (
	"testing"
	"time"
)

// TestMaintenancePlan_OccurrenceDate verifies that an occurrence is identified by
// its due date: OccurrenceDate follows NextDueDate as it advances across
// occurrences, rather than being pinned to the plan creation day.
func TestMaintenancePlan_OccurrenceDate(t *testing.T) {
	created := mustParseTime("2026-01-01T00:00:00Z")
	first := mustParseTime("2026-02-01T00:00:00Z")
	second := mustParseTime("2026-03-01T00:00:00Z")

	p := MaintenancePlan{ID: "p1", CreatedAt: created, NextDueDate: first}
	if got := p.OccurrenceDate(); !got.Equal(first) {
		t.Fatalf("OccurrenceDate should follow NextDueDate, got %v want %v", got, first)
	}

	// Advancing the due date to the next occurrence moves the occurrence anchor.
	p.NextDueDate = second
	if got := p.OccurrenceDate(); !got.Equal(second) {
		t.Fatalf("OccurrenceDate should advance with NextDueDate, got %v want %v", got, second)
	}

	// A plan with no scheduled due date falls back to the creation timestamp so
	// callers still receive a deterministic value.
	p2 := MaintenancePlan{ID: "p2", CreatedAt: created}
	if got := p2.OccurrenceDate(); !got.Equal(created) {
		t.Fatalf("OccurrenceDate with zero due date should fall back to CreatedAt, got %v", got)
	}
}

// TestMaintenanceTodoKey_PerOccurrence verifies the two required properties of
// the auto-todo idempotency key: distinct occurrences of the same plan produce
// distinct keys (each gets its own todo), while repeated observations of the
// same occurrence produce the same key (scans are idempotent). It also checks
// the key is stable across timezones.
func TestMaintenanceTodoKey_PerOccurrence(t *testing.T) {
	created := mustParseTime("2026-01-01T00:00:00Z")
	first := mustParseTime("2026-02-01T00:00:00Z")
	second := mustParseTime("2026-03-01T00:00:00Z")

	p := &MaintenancePlan{ID: "plan-1", CreatedAt: created, NextDueDate: first}
	k1 := MaintenanceTodoKey(p)
	k2 := MaintenanceTodoKey(p) // same occurrence -> same key (idempotent)

	if k1 != k2 {
		t.Fatalf("repeated scans of the same occurrence must produce the same key, got %q and %q", k1, k2)
	}

	// A later occurrence (advanced due date) must produce a different key so it
	// gets its own todo.
	p.NextDueDate = second
	k3 := MaintenanceTodoKey(p)
	if k3 == k1 {
		t.Fatalf("distinct occurrences must produce distinct keys, both were %q", k1)
	}

	// Two occurrences of two different plans that share a due date must not
	// collide: the key carries the plan id.
	other := &MaintenancePlan{ID: "plan-2", CreatedAt: created, NextDueDate: first}
	if MaintenanceTodoKey(other) == k1 {
		t.Fatalf("keys for different plans must not collide")
	}
}

// TestMaintenanceTodoKey_TimezoneStable verifies the key is rendered in UTC so
// the same occurrence resolves to the same key regardless of the process
// timezone. A due date near midnight in different zones must not shift the
// occurrence day used in the key.
func TestMaintenanceTodoKey_TimezoneStable(t *testing.T) {
	// 2026-02-01 00:30 UTC is 2026-01-31 19:30 in America/Chicago (UTC-6); the
	// key must use the UTC day so both representations agree.
	utc := mustParseTime("2026-02-01T00:30:00Z")
	chicago := utc.In(time.FixedZone("CST", -6*60*60))

	pUTC := &MaintenancePlan{ID: "p1", CreatedAt: utc, NextDueDate: utc}
	pChicago := &MaintenancePlan{ID: "p1", CreatedAt: utc, NextDueDate: chicago}

	if MaintenanceTodoKey(pUTC) != MaintenanceTodoKey(pChicago) {
		t.Fatalf("key must be timezone-stable: utc=%q chicago=%q",
			MaintenanceTodoKey(pUTC), MaintenanceTodoKey(pChicago))
	}
}
