package application

import (
	"context"
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
