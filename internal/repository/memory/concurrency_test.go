package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TestExecutionRepository_IdempotencyUnderConcurrency verifies that two
// concurrent creates with the same idempotency_key cannot both succeed.
func TestExecutionRepository_IdempotencyUnderConcurrency(t *testing.T) {
	store := NewStore()
	repo := NewExecutionRepository(store)
	ctx := context.Background()
	const N = 16
	var wg sync.WaitGroup
	errs := make([]error, N)
	oks := make([]bool, N)
	executedAt := time.Now()
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := repo.Create(ctx, &domain.Execution{
				ID:             "e-" + string(rune('a'+idx)),
				PlanID:         "p1",
				FacilityID:     "f1",
				ExecutedBy:     "u1",
				ExecutedAt:     executedAt,
				IdempotencyKey: "same-key",
			})
			errs[idx] = err
			oks[idx] = err == nil
		}(i)
	}
	wg.Wait()
	count := 0
	for _, ok := range oks {
		if ok {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 successful create, got %d", count)
	}
}

// TestPlanRepository_ConcurrentUpdate verifies optimistic concurrency control.
// All goroutines start with the same version, so only the first can succeed;
// the rest must fail with a version-conflict error.
func TestPlanRepository_ConcurrentUpdate(t *testing.T) {
	store := NewStore()
	repo := NewPlanRepository(store)
	ctx := context.Background()
	p := &domain.MaintenancePlan{
		ID: "plan-1", FacilityID: "f1", TemplateID: "t1",
		CurrentCycleDays: 30, Active: true,
	}
	if err := repo.CreatePlan(ctx, p); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Pre-read so all goroutines start with the same version.
	base, _ := repo.GetPlan(ctx, p.ID)
	var wg sync.WaitGroup
	const N = 8
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			current := *base
			current.SkippedCount++
			current.Version = current.Version + 1
			errs[idx] = repo.UpdatePlan(ctx, &current)
		}(i)
	}
	wg.Wait()
	okCount := 0
	for _, e := range errs {
		if e == nil {
			okCount++
		}
	}
	if okCount != 1 {
		t.Errorf("expected exactly 1 successful update, got %d", okCount)
	}
}

// TestFacilityRepository_ConcurrentStatusUpdate verifies optimistic concurrency
// for status updates.
func TestFacilityRepository_ConcurrentStatusUpdate(t *testing.T) {
	store := NewStore()
	repo := NewFacilityRepository(store)
	ctx := context.Background()
	f := &domain.Facility{
		ID: "f1", PlaceID: "p1", Name: "n", Code: "C-1", Category: "c",
		ResponsiblePersonID: "rp1", Criticality: domain.CriticalityCritical,
		Status: domain.FacilityNormal, Version: 1,
	}
	if err := repo.Create(ctx, f); err != nil {
		t.Fatalf("create: %v", err)
	}
	var wg sync.WaitGroup
	const N = 5
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = repo.UpdateStatus(ctx, f.ID, domain.FacilityPendingMaintenance, 1)
		}(i)
	}
	wg.Wait()
	okCount := 0
	for _, e := range errs {
		if e == nil {
			okCount++
		}
	}
	if okCount != 1 {
		t.Errorf("expected exactly 1 successful status update, got %d", okCount)
	}
}
