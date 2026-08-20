package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

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
}

// TestScheduler_EachOverdueOccurrenceGetsItsOwnTodo drives the full multi-cycle
// lifecycle: three distinct overdue occurrences of one plan must each produce
// their own todo, and a repeated scan of the final occurrence (no due-date
// advance) must not create a duplicate. This pins both requirements at once:
// per-occurrence todo generation and idempotent repeated scans.
func TestScheduler_EachOverdueOccurrenceGetsItsOwnTodo(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, -10))
	scheduler := NewSchedulerService(ports)

	// Three distinct overdue occurrences: due dates advance across cycles.
	dueDates := []time.Time{
		now.AddDate(0, 0, -10), // occurrence #1 (initial, set by setupPlan)
		now.AddDate(0, 0, -6),  // occurrence #2
		now.AddDate(0, 0, -1),  // occurrence #3
	}
	var keys []string
	for i, due := range dueDates {
		if i > 0 {
			current, err := ports.Plans.GetPlan(context.Background(), plan.ID)
			if err != nil {
				t.Fatalf("get plan at occurrence %d: %v", i, err)
			}
			current.NextDueDate = due
			current.Version++
			if err := ports.Plans.UpdatePlan(context.Background(), current); err != nil {
				t.Fatalf("advance to occurrence %d: %v", i, err)
			}
		}
		res, err := scheduler.Run(context.Background())
		if err != nil {
			t.Fatalf("scan %d: %v", i, err)
		}
		if res.TodosGenerated != 1 {
			t.Fatalf("occurrence %d: expected 1 todo generated, got %d", i, res.TodosGenerated)
		}
		cur, _ := ports.Plans.GetPlan(context.Background(), plan.ID)
		keys = append(keys, domain.MaintenanceTodoKey(cur))
	}

	// A repeated scan of the same (final) occurrence must be idempotent: no new
	// todo, no error.
	res, err := scheduler.Run(context.Background())
	if err != nil {
		t.Fatalf("repeated scan: %v", err)
	}
	if res.TodosGenerated != 0 {
		t.Fatalf("repeated scan of the same occurrence must not generate a todo, got %d", res.TodosGenerated)
	}

	todos, err := ports.Todos.ListByUser(context.Background(), personID, domain.PageQuery{Limit: 20})
	if err != nil {
		t.Fatalf("list todos: %v", err)
	}
	if len(todos.Items) != 3 {
		t.Fatalf("expected one todo per occurrence (3), got %d", len(todos.Items))
	}
	// Every todo must carry a distinct per-occurrence idempotency key.
	seen := map[string]bool{}
	for _, todo := range todos.Items {
		if seen[todo.IdempotencyKey] {
			t.Fatalf("duplicate idempotency key across todos: %s", todo.IdempotencyKey)
		}
		seen[todo.IdempotencyKey] = true
	}
	for i, k := range keys {
		if !seen[k] {
			t.Fatalf("occurrence %d key %s not found among generated todos", i, k)
		}
	}
}

// TestScheduler_IdempotentUnderConcurrentScans verifies that two scheduler runs
// racing on the same overdue occurrence create exactly one todo: the loser's
// Create surfaces an idempotency-key conflict that the scheduler treats as
// "already exists" rather than as an error. This is the race-safety guarantee
// that makes repeated scans idempotent even under concurrent/overlapping ticks.
func TestScheduler_IdempotentUnderConcurrentScans(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, -5))
	scheduler := NewSchedulerService(ports, WithTimeout(10*time.Second))

	ctx := context.Background()
	results := make([]ScanResult, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = scheduler.Run(ctx)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
	// Exactly one run wins the facility-status transition.
	if marked := results[0].OverdueMarked + results[1].OverdueMarked; marked != 1 {
		t.Errorf("expected exactly 1 facility marked overdue across concurrent runs, got %d", marked)
	}
	// Exactly one run creates the todo for the occurrence.
	if generated := results[0].TodosGenerated + results[1].TodosGenerated; generated != 1 {
		t.Errorf("expected exactly 1 todo generated across concurrent runs, got %d", generated)
	}
	// The idempotent conflict path must not be counted as an error.
	if errs2 := results[0].Errors + results[1].Errors; errs2 != 0 {
		t.Errorf("expected 0 errors across concurrent runs, got %d", errs2)
	}
	todos, err := ports.Todos.ListByUser(context.Background(), personID, domain.PageQuery{Limit: 20})
	if err != nil {
		t.Fatalf("list todos: %v", err)
	}
	if len(todos.Items) != 1 {
		t.Fatalf("expected exactly 1 todo after concurrent scans, got %d", len(todos.Items))
	}
}

// TestScheduler_CountsUnprocessablePlansAsErrors verifies that an overdue plan
// referencing a facility that cannot be loaded is counted in ScanResult.Errors
// and — critically — does not abort the scan, so other facilities still get
// their reminders. This exercises the new error-accounting contract.
func TestScheduler_CountsUnprocessablePlansAsErrors(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	goodFacility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	// A plan whose facility does not exist (dangling reference).
	badPlan := &domain.MaintenancePlan{
		ID:               "plan-dangling",
		FacilityID:       "facility-does-not-exist",
		TemplateID:       template.ID,
		CurrentCycleDays: template.CycleDays,
		NextDueDate:      now.AddDate(0, 0, -5),
		Active:           true,
	}
	if err := ports.Plans.CreatePlan(context.Background(), badPlan); err != nil {
		t.Fatalf("create bad plan: %v", err)
	}
	// A healthy overdue plan alongside it, to prove the scan continues past the
	// bad row and still produces its todo.
	setupPlan(t, ports, goodFacility.ID, template.ID, now.AddDate(0, 0, -5))

	scheduler := NewSchedulerService(ports)
	res, err := scheduler.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Errors != 1 {
		t.Errorf("expected 1 error for the dangling plan, got %d", res.Errors)
	}
	if res.TodosGenerated != 1 {
		t.Errorf("expected the good plan to still get its todo, got %d generated", res.TodosGenerated)
	}
	if res.OverdueMarked != 1 {
		t.Errorf("expected the good facility to be marked overdue, got %d", res.OverdueMarked)
	}
}
