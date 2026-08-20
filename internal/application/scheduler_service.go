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
// The scan has two phases:
//  1. Overdue plans: mark each plan's facility overdue (a state transition
//     that fires once per overdue stretch) and guarantee the responsible
//     person has a todo for THIS overdue occurrence — keyed by (plan,
//     NextDueDate) so a plan that falls overdue again on a later cycle gets
//     its own todo, while re-scanning the same occurrence is a no-op.
//  2. Overdue todos: flip open/assigned todos past their due date into the
//     overdue status.
func (s *SchedulerService) Run(ctx context.Context) (ScanResult, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	now := s.now()
	res := ScanResult{RunAt: now}

	// Phase 1: overdue plans.
	plans, err := s.ports.Plans.ListOverduePlans(ctx, now)
	if err != nil {
		return res, err
	}
	for _, p := range plans {
		s.handleOverduePlan(ctx, p, now, &res)
	}

	// Phase 2: overdue todos.
	todos, err := s.ports.Todos.ListOverdue(ctx, now)
	if err != nil {
		return res, err
	}
	for _, t := range todos {
		s.markTodoOverdue(ctx, t, &res)
	}
	return res, nil
}

// handleOverduePlan processes a single overdue maintenance plan.
//
// The facility is transitioned into the overdue state the first time the
// scheduler observes it overdue during a given stretch (Normal/Pending ->
// Overdue); a facility already overdue is not re-transitioned, so the
// timeline "facility_overdue" event fires once per stretch rather than once
// per scan.
//
// Independently of the state transition, the responsible person receives a
// todo for the specific overdue occurrence, identified by the plan's
// NextDueDate (see domain.MaintenanceTodoKey). Because each cycle advances
// NextDueDate to a new day, a plan that falls overdue again after being
// executed yields a new occurrence and therefore a new todo. Re-scanning the
// same occurrence finds the existing idempotency key and skips, which keeps
// repeated scanning safe.
//
// The overdue notification is sent alongside the NEW occurrence todo only, so
// every overdue occurrence surfaces exactly one reminder and re-scanning the
// same occurrence never duplicates it.
func (s *SchedulerService) handleOverduePlan(ctx context.Context, p *domain.MaintenancePlan, now time.Time, res *ScanResult) {
	f, err := s.ports.Facilities.Get(ctx, p.FacilityID)
	if err != nil {
		return
	}

	// State transition: mark the facility overdue (once per overdue stretch).
	if f.Status == domain.FacilityNormal || f.Status == domain.FacilityPendingMaintenance {
		if err := s.ports.Facilities.UpdateStatus(ctx, f.ID, domain.FacilityOverdue, f.Version); err == nil {
			res.OverdueMarked++
			_ = s.timeline(ctx, f.ID, "facility_overdue", "设施逾期", "scheduler", map[string]any{
				"plan_id":       p.ID,
				"next_due_date": p.NextDueDate,
			})
		}
	}

	// Per-occurrence responsible-person todo.
	rp, _ := s.ports.People.Get(ctx, f.ResponsiblePersonID)
	if rp == nil {
		return
	}
	key := domain.MaintenanceTodoKey(p)
	if _, err := s.ports.Todos.GetByIdempotencyKey(ctx, key); err == nil {
		// This occurrence already has a todo; the scan is idempotent.
		return
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
		// A concurrent scan may have inserted this occurrence first. Treat
		// that as success: the occurrence is tracked either way.
		if _, lookupErr := s.ports.Todos.GetByIdempotencyKey(ctx, key); lookupErr == nil {
			res.TodosGenerated++
		}
		return
	}
	res.TodosGenerated++
	// Surface one reminder per overdue occurrence. Bound to new-todo creation
	// above, so it never fires twice for the same occurrence.
	_ = s.notify(ctx, domain.NotificationInput{
		UserID: rp.ID,
		Type:   domain.NotificationMaintenanceOverdue,
		Title:  "保养逾期",
		Body:   fmt.Sprintf("设施 %s 保养已逾期 (计划 %s, 计划到期日 %s)", f.Name, p.ID, p.OccurrenceDate().Format(time.DateOnly)),
	})
}

// markTodoOverdue transitions an open or assigned todo whose due date has
// passed into the overdue status.
func (s *SchedulerService) markTodoOverdue(ctx context.Context, t *domain.Todo, res *ScanResult) {
	updated := *t
	updated.Status = domain.TodoOverdue
	updated.Version = t.Version + 1
	if err := s.ports.Todos.Update(ctx, &updated); err == nil {
		res.TodosOverdue++
	}
}

// ScanResult summarizes one scheduler scan.
type ScanResult struct {
	RunAt          time.Time `json:"run_at"`
	OverdueMarked  int       `json:"overdue_marked"`
	TodosGenerated int       `json:"todos_generated"`
	TodosOverdue   int       `json:"todos_overdue"`
}
