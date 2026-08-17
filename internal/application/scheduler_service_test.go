package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TestScheduler_OverdueMarking verifies the scheduler:
//   - Marks facilities with overdue plans as FacilityOverdue
//   - Generates an overdue todo for the responsible person
//   - Notifies the responsible person
func TestScheduler_OverdueMarking(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityCritical, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	// Plan was due 5 days ago.
	setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, -5))
	svc := NewSchedulerService(ports, WithTimeout(5*time.Second))
	res, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.OverdueMarked != 1 {
		t.Errorf("expected 1 overdue_marked, got %d", res.OverdueMarked)
	}
	if res.TodosGenerated != 1 {
		t.Errorf("expected 1 todo_generated, got %d", res.TodosGenerated)
	}
	// Verify the facility is now overdue.
	ff, _ := ports.Facilities.Get(context.Background(), f.ID)
	if ff.Status != domain.FacilityOverdue {
		t.Errorf("expected overdue, got %s", ff.Status)
	}
	// Verify a todo was created.
	todos, _ := ports.Todos.List(context.Background(), domain.PageQuery{Limit: 100, Filters: map[string]string{"assigned_to": rpid}})
	if len(todos.Items) != 1 {
		t.Errorf("expected 1 todo, got %d", len(todos.Items))
	}
	// Verify a notification was created.
	notifs, _ := ports.Notifications.ListByUser(context.Background(), rpid, domain.PageQuery{Limit: 100})
	if len(notifs.Items) != 1 {
		t.Errorf("expected 1 notification, got %d", len(notifs.Items))
	}
	if notifs.Items[0].Type != domain.NotificationMaintenanceOverdue {
		t.Errorf("expected maintenance_overdue, got %s", notifs.Items[0].Type)
	}
}

// TestScheduler_RunsIdempotentlyForOverdue verifies that running the scheduler
// twice does NOT generate duplicate todos (the auto-todo is keyed by
// plan_id + next_due_date).
func TestScheduler_RunsIdempotentlyForOverdue(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, -5))
	svc := NewSchedulerService(ports, WithTimeout(5*time.Second))
	if _, err := svc.Run(context.Background()); err != nil {
		t.Fatalf("run 1: %v", err)
	}
	if _, err := svc.Run(context.Background()); err != nil {
		t.Fatalf("run 2: %v", err)
	}
	todos, _ := ports.Todos.List(context.Background(), domain.PageQuery{Limit: 100, Filters: map[string]string{"assigned_to": rpid}})
	if len(todos.Items) != 1 {
		t.Errorf("expected 1 todo after two runs, got %d", len(todos.Items))
	}
}

// TestScheduler_MarksOverdueTodos verifies the scheduler updates open todos
// past their due date to the overdue status.
func TestScheduler_MarksOverdueTodos(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	// Create an overdue todo manually.
	if err := ports.Todos.Create(context.Background(), &domain.Todo{
		ID:             "todo-overdue",
		FacilityID:     f.ID,
		AssignedTo:     rpid,
		Type:           domain.TodoTypeMaintenance,
		DueDate:        now.AddDate(0, 0, -3),
		Status:         domain.TodoAssigned,
		Title:          "过期待办",
		IdempotencyKey: "todo-overdue-key",
	}); err != nil {
		t.Fatalf("create todo: %v", err)
	}
	svc := NewSchedulerService(ports, WithTimeout(5*time.Second))
	res, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.TodosOverdue != 1 {
		t.Errorf("expected 1 todo_overdue, got %d", res.TodosOverdue)
	}
	stored, _ := ports.Todos.Get(context.Background(), "todo-overdue")
	if stored.Status != domain.TodoOverdue {
		t.Errorf("expected overdue status, got %s", stored.Status)
	}
}
