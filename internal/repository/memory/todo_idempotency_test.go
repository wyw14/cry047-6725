package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TestTodoRepository_IdempotencyUnderConcurrency verifies that two concurrent
// creates with the same idempotency_key cannot both succeed: exactly one todo
// survives. This is the per-occurrence uniqueness the scheduler relies on when
// two scans race to create the todo for the same overdue occurrence.
func TestTodoRepository_IdempotencyUnderConcurrency(t *testing.T) {
	store := NewStore()
	repo := NewTodoRepository(store)
	ctx := context.Background()
	const N = 16
	var wg sync.WaitGroup
	errs := make([]error, N)
	oks := make([]bool, N)
	due := time.Now().UTC()
	// All goroutines use the same key (the same overdue occurrence) but distinct
	// todo ids; only the first create may win.
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := repo.Create(ctx, &domain.Todo{
				ID:             "t-" + string(rune('a'+idx)),
				FacilityID:     "f1",
				AssignedTo:     "u1",
				Type:           domain.TodoTypeMaintenance,
				DueDate:        due,
				Status:         domain.TodoAssigned,
				Title:          "逾期保养",
				IdempotencyKey: "auto-todo-p1-2026-02-01",
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
	// The single surviving todo must be retrievable by the idempotency key.
	got, err := repo.GetByIdempotencyKey(ctx, "auto-todo-p1-2026-02-01")
	if err != nil {
		t.Fatalf("get by idempotency key: %v", err)
	}
	if got.AssignedTo != "u1" {
		t.Errorf("retrieved todo assigned_to = %q, want u1", got.AssignedTo)
	}
}

// TestTodoRepository_DistinctOccurrenceKeys verifies that two todos for distinct
// occurrences of the same plan (distinct idempotency keys, carrying distinct
// due dates) can both coexist. The repository must no longer collapse
// per-occurrence keys onto a single plan-level key.
func TestTodoRepository_DistinctOccurrenceKeys(t *testing.T) {
	store := NewStore()
	repo := NewTodoRepository(store)
	ctx := context.Background()
	due := time.Now().UTC()
	first := &domain.Todo{
		ID: "t-1", FacilityID: "f1", AssignedTo: "u1", Type: domain.TodoTypeMaintenance,
		DueDate: due, Status: domain.TodoAssigned, Title: "逾期保养",
		IdempotencyKey: "auto-todo-p1-2026-02-01",
	}
	second := &domain.Todo{
		ID: "t-2", FacilityID: "f1", AssignedTo: "u1", Type: domain.TodoTypeMaintenance,
		DueDate: due, Status: domain.TodoAssigned, Title: "逾期保养",
		IdempotencyKey: "auto-todo-p1-2026-03-01",
	}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first occurrence todo: %v", err)
	}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("create second occurrence todo: %v", err)
	}
	if _, err := repo.GetByIdempotencyKey(ctx, "auto-todo-p1-2026-02-01"); err != nil {
		t.Errorf("first occurrence todo must be retrievable, got: %v", err)
	}
	if _, err := repo.GetByIdempotencyKey(ctx, "auto-todo-p1-2026-03-01"); err != nil {
		t.Errorf("second occurrence todo must be retrievable, got: %v", err)
	}
}
