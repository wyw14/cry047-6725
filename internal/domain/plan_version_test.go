package domain

import (
	"testing"
	"time"
)

// minute is a small constant used to build deterministic timestamps in the
// temporal-helper tests.
func mustVersion(id string, versionNumber, cycle int, changedAt time.Time) *PlanVersion {
	return &PlanVersion{
		ID:            id,
		PlanID:        "plan-1",
		VersionNumber: versionNumber,
		CycleDays:     cycle,
		ChangedAt:     changedAt,
		ChangedBy:     "u1",
		Reason:        "test",
	}
}

func TestNextVersionNumber(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		versions []*PlanVersion
		want     int
	}{
		{"empty", nil, 1},
		{"single", []*PlanVersion{mustVersion("a", 1, 30, t0)}, 2},
		{"sequential", []*PlanVersion{
			mustVersion("a", 1, 30, t0),
			mustVersion("b", 2, 60, t0.Add(48*time.Hour)),
		}, 3},
		{"gap-robust", []*PlanVersion{
			mustVersion("a", 1, 30, t0),
			mustVersion("b", 3, 90, t0.Add(96*time.Hour)),
		}, 4},
		{"nil-entries-ignored", []*PlanVersion{nil, mustVersion("a", 5, 30, t0), nil}, 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NextVersionNumber(tc.versions); got != tc.want {
				t.Fatalf("NextVersionNumber = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestLatestPlanVersion(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(48 * time.Hour)
	t2 := t0.Add(96 * time.Hour)

	v1 := mustVersion("a", 1, 30, t0)
	v2 := mustVersion("b", 2, 60, t1)
	v3 := mustVersion("c", 3, 90, t2)

	if got, ok := LatestPlanVersion(nil); ok {
		t.Fatalf("empty history should return false, got %v", got)
	}
	if got, ok := LatestPlanVersion([]*PlanVersion{nil, nil}); ok {
		t.Fatalf("all-nil history should return false, got %v", got)
	}
	got, ok := LatestPlanVersion([]*PlanVersion{v1, v2, v3})
	if !ok || got.ID != v3.ID {
		t.Fatalf("expected %s, got %v (ok=%v)", v3.ID, got, ok)
	}
	// Out-of-order input must still pick the latest by ChangedAt.
	got, _ = LatestPlanVersion([]*PlanVersion{v3, v1, v2})
	if got.ID != v3.ID {
		t.Fatalf("expected latest %s, got %s", v3.ID, got.ID)
	}
}

func TestLatestPlanVersion_TimestampTieBreaksByVersion(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	low := mustVersion("a", 1, 30, t0)
	high := mustVersion("b", 2, 60, t0) // same ChangedAt, higher VersionNumber
	got, _ := LatestPlanVersion([]*PlanVersion{low, high})
	if got.ID != high.ID {
		t.Fatalf("expected tie-break to higher version %s, got %s", high.ID, got.ID)
	}
}

func TestEarliestPlanVersion(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v1 := mustVersion("a", 1, 30, t0)
	v2 := mustVersion("b", 2, 60, t0.Add(48*time.Hour))

	if _, ok := EarliestPlanVersion(nil); ok {
		t.Fatalf("empty history should return false")
	}
	got, _ := EarliestPlanVersion([]*PlanVersion{v2, v1})
	if got.ID != v1.ID {
		t.Fatalf("expected earliest %s, got %s", v1.ID, got.ID)
	}
}

func TestEffectiveCycleAt(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(48 * time.Hour) // first change
	t2 := t0.Add(96 * time.Hour) // second change
	v1 := mustVersion("a", 1, 30, t0)
	v2 := mustVersion("b", 2, 60, t1)
	v3 := mustVersion("c", 3, 90, t2)
	hist := []*PlanVersion{v1, v2, v3}

	cases := []struct {
		name   string
		at     time.Time
		want   int
		wantOk bool
	}{
		{"before-first", t0.Add(-1 * time.Hour), 30, true},
		{"at-first", t0, 30, true},
		{"between-1-and-2", t0.Add(24 * time.Hour), 30, true},
		{"at-second", t1, 60, true},
		{"between-2-and-3", t0.Add(72 * time.Hour), 60, true},
		{"after-last", t2.Add(1 * time.Hour), 90, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := EffectiveCycleAt(hist, tc.at)
			if ok != tc.wantOk {
				t.Fatalf("EffectiveCycleAt ok = %v, want %v", ok, tc.wantOk)
			}
			if got != tc.want {
				t.Fatalf("EffectiveCycleAt = %d, want %d", got, tc.want)
			}
		})
	}

	if got, ok := EffectiveCycleAt(nil, t0); ok || got != 0 {
		t.Fatalf("empty history should return (0, false), got (%d, %v)", got, ok)
	}
}

// TestEffectiveCycleAt_PreservesHistoryIndependently confirms that calling the
// read-only helper does not mutate any snapshot (a regression guard for the old
// RewriteHistoricalVersions behavior which overwrote historical entries).
func TestEffectiveCycleAt_PreservesHistoryIndependently(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v1 := mustVersion("a", 1, 30, t0)
	v2 := mustVersion("b", 2, 60, t0.Add(48*time.Hour))
	hist := []*PlanVersion{v1, v2}

	_, _ = EffectiveCycleAt(hist, t0.Add(72*time.Hour))
	if v1.CycleDays != 30 {
		t.Fatalf("historical v1 cycle rewritten to %d", v1.CycleDays)
	}
	if v2.CycleDays != 60 {
		t.Fatalf("v2 cycle rewritten to %d", v2.CycleDays)
	}
}
