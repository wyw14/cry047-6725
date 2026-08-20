package memory

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TestPlanRepository_AppendPlanVersionIsImmutable confirms that appending a new
// plan version snapshot never rewrites previously stored snapshots. This is the
// storage-level guard for the "older audit records must keep showing the cycle
// that was actually in effect" invariant.
func TestPlanRepository_AppendPlanVersionIsImmutable(t *testing.T) {
	store := NewStore()
	repo := NewPlanRepository(store)
	ctx := context.Background()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	v1 := &domain.PlanVersion{
		ID: "pv-1", PlanID: "plan-1", VersionNumber: 1,
		CycleDays: 30, TemplateID: "tpl-1",
		ChangedBy: "u1", ChangedAt: t0, Reason: "init",
	}
	if err := repo.AppendPlanVersion(ctx, v1); err != nil {
		t.Fatalf("append v1: %v", err)
	}
	// Append a second snapshot with a different cycle.
	v2 := &domain.PlanVersion{
		ID: "pv-2", PlanID: "plan-1", VersionNumber: 2,
		CycleDays: 60, TemplateID: "tpl-1",
		ChangedBy: "u1", ChangedAt: t0.Add(48 * time.Hour), Reason: "extend",
	}
	if err := repo.AppendPlanVersion(ctx, v2); err != nil {
		t.Fatalf("append v2: %v", err)
	}

	got, err := repo.ListPlanVersions(ctx, "plan-1")
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(got))
	}
	// The first snapshot must retain its original cycle, not be rewritten to 60.
	if got[0].CycleDays != 30 {
		t.Errorf("snapshot[0] cycle = %d, want 30 (history was rewritten)", got[0].CycleDays)
	}
	if got[1].CycleDays != 60 {
		t.Errorf("snapshot[1] cycle = %d, want 60", got[1].CycleDays)
	}
}

// TestPlanRepository_AppendPlanVersionAssignsVersionNumber confirms the
// repository defensively assigns the next sequential version number when a
// caller leaves it unset (0).
func TestPlanRepository_AppendPlanVersionAssignsVersionNumber(t *testing.T) {
	store := NewStore()
	repo := NewPlanRepository(store)
	ctx := context.Background()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// First snapshot explicitly carries version 1.
	if err := repo.AppendPlanVersion(ctx, &domain.PlanVersion{
		ID: "pv-1", PlanID: "plan-1", VersionNumber: 1, CycleDays: 30,
		ChangedBy: "u1", ChangedAt: t0, Reason: "init",
	}); err != nil {
		t.Fatalf("append v1: %v", err)
	}
	// Second snapshot leaves VersionNumber unset; the repo must derive 2.
	if err := repo.AppendPlanVersion(ctx, &domain.PlanVersion{
		ID: "pv-2", PlanID: "plan-1", VersionNumber: 0, CycleDays: 60,
		ChangedBy: "u1", ChangedAt: t0.Add(48 * time.Hour), Reason: "extend",
	}); err != nil {
		t.Fatalf("append v2: %v", err)
	}
	got, _ := repo.ListPlanVersions(ctx, "plan-1")
	if len(got) != 2 || got[1].VersionNumber != 2 {
		t.Fatalf("expected version numbers [1,2], got %v", versions(got))
	}
}

func versions(vs []*domain.PlanVersion) []int {
	out := make([]int, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.VersionNumber)
	}
	return out
}
