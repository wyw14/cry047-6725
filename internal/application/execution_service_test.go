package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TestExecutionService_Idempotency verifies that submitting the same
// idempotency_key twice returns the same record without creating a new one.
func TestExecutionService_Idempotency(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, 30))
	svc := NewExecutionService(ports)
	in := domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     f.ID,
		ExecutedBy:     operatorActor().ID,
		ExecutedAt:     now,
		IdempotencyKey: "idem-key-1",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Value: "ok", Pass: true},
			{ItemCode: "refrigerant", NumericValue: ptrFloat64(0.5), Pass: true},
		},
	}
	e1, err := svc.Submit(context.Background(), operatorActor(), in)
	if err != nil {
		t.Fatalf("submit 1: %v", err)
	}
	// Submit again with same key.
	e2, err := svc.Submit(context.Background(), operatorActor(), in)
	if err != nil {
		t.Fatalf("submit 2: %v", err)
	}
	if e1.ID != e2.ID {
		t.Errorf("expected same id, got %s vs %s", e1.ID, e2.ID)
	}
	list, err := ports.Executions.List(context.Background(), domain.PageQuery{Limit: 100})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 1 {
		t.Errorf("expected 1 execution, got %d", len(list.Items))
	}
}

// TestExecutionService_ForbidsIfFacilityUnderRepair verifies the invariant
// that maintenance executions cannot be submitted while the facility is under
// repair (must use the anomaly flow instead).
func TestExecutionService_ForbidsIfFacilityUnderRepair(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityUnderRepair)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, 30))
	svc := NewExecutionService(ports)
	_, err := svc.Submit(context.Background(), operatorActor(), domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     f.ID,
		ExecutedBy:     operatorActor().ID,
		ExecutedAt:     now,
		IdempotencyKey: "idem-key-2",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Pass: true},
			{ItemCode: "refrigerant", NumericValue: ptrFloat64(0.5), Pass: true},
		},
	})
	if err == nil {
		t.Fatalf("expected STATE_FORBIDDEN, got nil")
	}
	if !domain.IsDomainError(err, domain.CodeStateForbidden) {
		t.Errorf("expected STATE_FORBIDDEN, got: %v", err)
	}
}

// TestExecutionService_InvalidInspectionValues verifies template-driven
// validation: required items must be present and numeric ranges enforced.
func TestExecutionService_InvalidInspectionValues(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, 30))
	svc := NewExecutionService(ports)
	// Missing required item 'refrigerant'.
	_, err := svc.Submit(context.Background(), operatorActor(), domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     f.ID,
		ExecutedBy:     operatorActor().ID,
		ExecutedAt:     now,
		IdempotencyKey: "idem-key-3",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Pass: true},
		},
	})
	if err == nil {
		t.Fatalf("expected INVALID for missing required, got nil")
	}
	// Out-of-range value: refrigerant = 5 (max is 0.6).
	_, err = svc.Submit(context.Background(), operatorActor(), domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     f.ID,
		ExecutedBy:     operatorActor().ID,
		ExecutedAt:     now,
		IdempotencyKey: "idem-key-4",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Pass: true},
			{ItemCode: "refrigerant", NumericValue: ptrFloat64(5), Pass: false},
		},
	})
	if err == nil {
		t.Fatalf("expected INVALID for out-of-range, got nil")
	}
}

// TestExecutionService_ViewerCannotSubmit enforces the RBAC boundary.
func TestExecutionService_ViewerCannotSubmit(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, 30))
	svc := NewExecutionService(ports)
	_, err := svc.Submit(context.Background(), viewerActor(), domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     f.ID,
		ExecutedBy:     viewerActor().ID,
		ExecutedAt:     now,
		IdempotencyKey: "idem-viewer",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Pass: true},
			{ItemCode: "refrigerant", NumericValue: ptrFloat64(0.5), Pass: true},
		},
	})
	if err == nil {
		t.Fatalf("expected FORBIDDEN for viewer, got nil")
	}
}

// TestExecutionService_PlanFacilityMismatch verifies cross-aggregate invariant.
func TestExecutionService_PlanFacilityMismatch(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f1 := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	f2 := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f1.ID, tpl.ID, now.AddDate(0, 0, 30))
	svc := NewExecutionService(ports)
	_, err := svc.Submit(context.Background(), operatorActor(), domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     f2.ID, // mismatch
		ExecutedBy:     operatorActor().ID,
		ExecutedAt:     now,
		IdempotencyKey: "idem-mismatch",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Pass: true},
			{ItemCode: "refrigerant", NumericValue: ptrFloat64(0.5), Pass: true},
		},
	})
	if err == nil {
		t.Fatalf("expected INVALID for mismatch, got nil")
	}
}

// TestExecutionService_SuccessResetsStatus verifies that a successful
// execution resets facility status to Normal.
func TestExecutionService_SuccessResetsStatus(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityPendingMaintenance)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, 30))
	svc := NewExecutionService(ports)
	if _, err := svc.Submit(context.Background(), operatorActor(), domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     f.ID,
		ExecutedBy:     operatorActor().ID,
		ExecutedAt:     now,
		IdempotencyKey: "idem-success",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Pass: true},
			{ItemCode: "refrigerant", NumericValue: ptrFloat64(0.5), Pass: true},
		},
	}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	updated, _ := ports.Facilities.Get(context.Background(), f.ID)
	if updated.Status != domain.FacilityNormal {
		t.Errorf("expected normal after execution, got %s", updated.Status)
	}
}
