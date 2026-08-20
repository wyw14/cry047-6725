package application

import (
	"context"
	"fmt"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// SchedulerService periodically scans for overdue plans and due todos,
// transitions facility state accordingly and generates maintenance todos.
// It uses the local scheduler adapter, so the platform remains offline.
type SchedulerService struct {
	baseService
}

// NewSchedulerService returns a SchedulerService.
func NewSchedulerService(ports *domain.Ports, opts ...Option) *SchedulerService {
	return &SchedulerService{baseService: newBase(ports, opts)}
}

// Run executes one full scan synchronously and returns a summary.
//
// The scan has two independent phases:
//  1. For every overdue plan, transition its facility into the overdue state
//     (once per overdue episode) and ensure exactly one maintenance todo exists
//     for the overdue occurrence. Todo generation is idempotent per occurrence
//     — the occurrence's idempotency key is (plan id, due date) — so a repeated
//     scan of the same overdue occurrence is a no-op, while a later overdue
//     occurrence (the plan's due date having advanced) gets its own todo.
//  2. Transition every open/assigned todo past its due date into the overdue
//     status.
//
// Neither phase aborts the whole scan on a single plan's failure: the failing
// item is counted in ScanResult.Errors and the scan continues, so one bad row
// cannot starve every other facility of reminders.
func (s *SchedulerService) Run(ctx context.Context) (ScanResult, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	now := s.now()
	res := ScanResult{RunAt: now}

	// 1. Overdue plans -> mark facility overdue and ensure one todo per overdue
	//    occurrence exists for the responsible person.
	plans, err := s.ports.Plans.ListOverduePlans(ctx, now)
	if err != nil {
		return res, err
	}
	for _, p := range plans {
		f, err := s.ports.Facilities.Get(ctx, p.FacilityID)
		if err != nil {
			// The plan references a facility that cannot be loaded (deleted or a
			// transient read failure). Nothing can be done for this plan; record
			// it and keep scanning the rest.
			res.Errors++
			continue
		}
		s.markFacilityOverdue(ctx, p, f, &res)
		generated, err := s.ensureOverdueTodo(ctx, p, f, now)
		if generated {
			res.TodosGenerated++
		}
		if err != nil {
			res.Errors++
		}
	}

	// 2. Mark overdue todos.
	todos, err := s.ports.Todos.ListOverdue(ctx, now)
	if err != nil {
		return res, err
	}
	for _, t := range todos {
		updated := *t
		updated.Status = domain.TodoOverdue
		updated.Version = t.Version + 1
		if err := s.ports.Todos.Update(ctx, &updated); err == nil {
			res.TodosOverdue++
		}
	}
	return res, nil
}

// markFacilityOverdue transitions the facility into the overdue state the first
// time the scan observes the overdue plan, and records the matching timeline
// event and notification. Only Normal / PendingMaintenance facilities are
// transitioned, so the notification fires once per overdue episode rather than
// on every scan while the plan stays overdue. A version conflict (another scan
// or a manual state change moved first) is not fatal: the facility is simply
// re-evaluated on the next scan.
func (s *SchedulerService) markFacilityOverdue(ctx context.Context, p *domain.MaintenancePlan, f *domain.Facility, res *ScanResult) {
	if f.Status != domain.FacilityNormal && f.Status != domain.FacilityPendingMaintenance {
		return
	}
	if err := s.ports.Facilities.UpdateStatus(ctx, f.ID, domain.FacilityOverdue, f.Version); err != nil {
		return
	}
	res.OverdueMarked++
	_ = s.timeline(ctx, f.ID, "facility_overdue", "设施逾期", "scheduler", map[string]any{
		"plan_id":       p.ID,
		"next_due_date": p.NextDueDate,
	})
	rp, _ := s.ports.People.Get(ctx, f.ResponsiblePersonID)
	if rp != nil {
		_ = s.notify(ctx, domain.NotificationInput{
			UserID: rp.ID,
			Type:   domain.NotificationMaintenanceOverdue,
			Title:  "保养逾期",
			Body:   fmt.Sprintf("设施 %s 保养已逾期 (计划 %s)", f.Name, p.ID),
		})
	}
}

// ensureOverdueTodo guarantees that exactly one maintenance todo exists for the
// overdue occurrence of p, creating it when missing. It reports whether a new
// todo was created by this call.
//
// Idempotency is per occurrence: the todo's idempotency key is derived from
// (plan id, occurrence due date) by domain.MaintenanceTodoKey, so repeated
// scans of the same overdue occurrence (same due date) find the existing todo
// and do nothing, while a later overdue occurrence (advanced due date) resolves
// to a different key and gets its own todo. The check-then-create window is
// closed by treating an idempotency-key conflict from Create as "another run
// already created it" rather than as an error — that is the correct idempotent
// outcome when two scans race on the same occurrence.
func (s *SchedulerService) ensureOverdueTodo(ctx context.Context, p *domain.MaintenancePlan, f *domain.Facility, now time.Time) (bool, error) {
	rp, err := s.ports.People.Get(ctx, f.ResponsiblePersonID)
	if err != nil || rp == nil {
		// No responsible person to assign the todo to; there is nothing to
		// generate for this occurrence. This is not a creation error.
		return false, nil
	}
	key := domain.MaintenanceTodoKey(p)
	// Fast path: a todo for this occurrence already exists. Repeated scans of
	// the same overdue occurrence stop here.
	if _, err := s.ports.Todos.GetByIdempotencyKey(ctx, key); err == nil {
		return false, nil
	}
	todo := &domain.Todo{
		ID:             s.id(),
		FacilityID:     f.ID,
		PlanID:         p.ID,
		AssignedTo:     rp.ID,
		Type:           domain.TodoTypeMaintenance,
		DueDate:        now.AddDate(0, 0, 3),
		Status:         domain.TodoAssigned,
		Title:          "逾期保养: " + f.Name,
		IdempotencyKey: key,
	}
	if err := s.ports.Todos.Create(ctx, todo); err != nil {
		// A conflict means a concurrent scan already created the todo for this
		// occurrence: the idempotent outcome, not a failure.
		if domain.IsDomainError(err, domain.CodeConflict) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ScanResult summarizes one scheduler scan.
type ScanResult struct {
	RunAt          time.Time `json:"run_at"`
	OverdueMarked  int       `json:"overdue_marked"`
	TodosGenerated int       `json:"todos_generated"`
	TodosOverdue   int       `json:"todos_overdue"`
	// Errors counts items the scan could not process (e.g. a plan referencing a
	// facility that cannot be loaded, or an unexpected todo-creation failure).
	// Benign concurrency races (an idempotency-key conflict from a parallel
	// scan, a version conflict on a todo status update) are intentionally not
	// counted here.
	Errors int `json:"errors"`
}
