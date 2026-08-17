package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

func TestCriticalOverdueFacilityStaysNonNormalAfterMaintenance(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityCritical, domain.FacilityOverdue)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, -2))
	svc := NewExecutionService(ports)
	_, err := svc.Submit(context.Background(), operatorActor(), domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     facility.ID,
		ExecutedBy:     operatorActor().ID,
		ExecutedAt:     now,
		IdempotencyKey: "critical-overdue-maintenance",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Value: "clean", Pass: true},
			{ItemCode: "refrigerant", NumericValue: ptrFloat64(0.5), Pass: true},
		},
	})
	if err != nil {
		t.Fatalf("submit maintenance: %v", err)
	}
	stored, err := ports.Facilities.Get(context.Background(), facility.ID)
	if err != nil {
		t.Fatalf("get facility: %v", err)
	}
	if stored.Status == domain.FacilityNormal {
		t.Fatalf("critical overdue facility became normal after ordinary maintenance")
	}
}
