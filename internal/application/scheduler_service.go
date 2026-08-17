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
func (s *SchedulerService) Run(ctx context.Context) (ScanResult, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	now := s.now()
	res := ScanResult{RunAt: now}
	// 1. Overdue plans -> mark facility as overdue.
	plans, err := s.ports.Plans.ListOverduePlans(ctx, now)
	if err != nil {
		return res, err
	}
	for _, p := range plans {
		f, err := s.ports.Facilities.Get(ctx, p.FacilityID)
		if err != nil {
			continue
		}
		if f.Status == domain.FacilityNormal || f.Status == domain.FacilityPendingMaintenance {
			if err := s.ports.Facilities.UpdateStatus(ctx, f.ID, domain.FacilityOverdue, f.Version); err == nil {
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
		}
		// 2. Auto-generate an overdue todo for the responsible person.
		rp, _ := s.ports.People.Get(ctx, f.ResponsiblePersonID)
		if rp == nil {
			continue
		}
		key := "auto-todo-" + p.ID + "-" + p.NextDueDate.Format("2006-01-02")
		if _, err := s.ports.Todos.GetByIdempotencyKey(ctx, key); err == nil {
			continue
		}
		t := &domain.Todo{
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
		if err := s.ports.Todos.Create(ctx, t); err == nil {
			res.TodosGenerated++
		}
	}
	// 3. Mark overdue todos.
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

// ScanResult summarizes one scheduler scan.
type ScanResult struct {
	RunAt          time.Time `json:"run_at"`
	OverdueMarked  int       `json:"overdue_marked"`
	TodosGenerated int       `json:"todos_generated"`
	TodosOverdue   int       `json:"todos_overdue"`
}
