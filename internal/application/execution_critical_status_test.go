package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// submitMaintenance is a test helper that submits a valid preventive
// maintenance execution for the given facility/plan and returns the resulting
// execution record (or fails the test).
func submitMaintenance(t *testing.T, ports *domain.Ports, plan *domain.MaintenancePlan, facility *domain.Facility, now time.Time, key string) *domain.Execution {
	t.Helper()
	svc := NewExecutionService(ports)
	e, err := svc.Submit(context.Background(), operatorActor(), domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     facility.ID,
		ExecutedBy:     operatorActor().ID,
		ExecutedAt:     now,
		IdempotencyKey: key,
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Value: "clean", Pass: true},
			{ItemCode: "refrigerant", NumericValue: ptrFloat64(0.5), Pass: true},
		},
	})
	if err != nil {
		t.Fatalf("submit maintenance: %v", err)
	}
	return e
}

// facilityStatus reloads a facility and returns its current status.
func facilityStatus(t *testing.T, ports *domain.Ports, id string) domain.FacilityStatus {
	t.Helper()
	stored, err := ports.Facilities.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get facility: %v", err)
	}
	return stored.Status
}

// TestCriticalOverdueFacilityStaysNonNormalAfterMaintenance verifies the core
// fix: an overdue critical facility submitted through the ordinary maintenance
// flow must NOT appear healthy. Its repair and recovery steps are kept on the
// critical path by routing it into UnderRepair instead of Normal.
func TestCriticalOverdueFacilityStaysNonNormalAfterMaintenance(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityCritical, domain.FacilityOverdue)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, -2))
	e := submitMaintenance(t, ports, plan, facility, now, "critical-overdue-maintenance")

	// The execution record itself is still created.
	if e.Status != domain.ExecutionSubmitted {
		t.Fatalf("execution not submitted, status=%s", e.Status)
	}
	if facilityStatus(t, ports, facility.ID) == domain.FacilityNormal {
		t.Fatalf("critical overdue facility became normal after ordinary maintenance")
	}
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityUnderRepair {
		t.Fatalf("expected under_repair, got %s", got)
	}
}

// TestCriticalOverdueFacility_RoutesToRepairAndNotifies verifies that routing
// an overdue critical facility into UnderRepair also records a timeline event
// and notifies the responsible person that repair work is required.
func TestCriticalOverdueFacility_RoutesToRepairAndNotifies(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityCritical, domain.FacilityOverdue)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, -2))
	submitMaintenance(t, ports, plan, facility, now, "critical-overdue-notify")

	events, err := ports.Timeline.ListByFacility(context.Background(), facility.ID, 50)
	if err != nil {
		t.Fatalf("list timeline: %v", err)
	}
	var sawCompleted, sawRepairRequired bool
	for _, ev := range events {
		if ev.EventType == "maintenance_completed" {
			sawCompleted = true
		}
		if ev.EventType == "repair_required" {
			sawRepairRequired = true
		}
	}
	if !sawCompleted {
		t.Errorf("expected maintenance_completed timeline event")
	}
	if !sawRepairRequired {
		t.Errorf("expected repair_required timeline event for overdue critical facility")
	}

	notes, err := ports.Notifications.ListByUser(context.Background(), personID, domain.PageQuery{Limit: 50})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notes.Items) == 0 {
		t.Fatalf("expected responsible person to be notified of required repair")
	}
}

// TestOverdueNonCriticalFacility_ReturnsToNormal verifies the ordinary flow is
// preserved for an overdue facility that is NOT high priority: it returns to
// Normal after maintenance, exactly as before the fix.
func TestOverdueNonCriticalFacility_ReturnsToNormal(t *testing.T) {
	for _, crit := range []domain.Criticality{domain.CriticalityStandard, domain.CriticalityImportant} {
		now := time.Now().UTC()
		ports, _ := setupTestPorts(t, now)
		placeID, personID := setupPlaceAndPerson(t, ports)
		facility := setupFacility(t, ports, placeID, personID, crit, domain.FacilityOverdue)
		template := setupTemplate(t, ports)
		plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, -2))
		submitMaintenance(t, ports, plan, facility, now, "overdue-"+string(crit))
		if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityNormal {
			t.Errorf("criticality=%s: expected normal (ordinary flow), got %s", crit, got)
		}
	}
}

// TestPendingCriticalFacility_ReturnsToNormal verifies that a critical facility
// that was merely pending maintenance (not overdue) still returns to Normal —
// the routing exception applies only to OVERDUE critical facilities.
func TestPendingCriticalFacility_ReturnsToNormal(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityCritical, domain.FacilityPendingMaintenance)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, 30))
	submitMaintenance(t, ports, plan, facility, now, "pending-critical")
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityNormal {
		t.Fatalf("expected normal for pending critical, got %s", got)
	}
}

// TestPendingStandardFacility_ReturnsToNormal is the regression guard for the
// pre-existing ordinary flow on non-critical pending facilities.
func TestPendingStandardFacility_ReturnsToNormal(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityPendingMaintenance)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, 30))
	submitMaintenance(t, ports, plan, facility, now, "pending-standard")
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityNormal {
		t.Fatalf("expected normal for pending standard, got %s", got)
	}
}

// TestNormalFacility_StatusUnchangedAfterMaintenance verifies that a facility
// already in Normal is left untouched by the maintenance flow (the routing step
// only applies to pending/overdue facilities).
func TestNormalFacility_StatusUnchangedAfterMaintenance(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityCritical, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, 30))
	submitMaintenance(t, ports, plan, facility, now, "normal-critical")
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityNormal {
		t.Fatalf("normal facility should remain normal, got %s", got)
	}
}

// TestOverdueCriticalFacility_CanCompleteRepairFlow verifies the end-to-end
// recovery path: after being routed to UnderRepair, the facility can proceed
// through recovery confirmation back to Normal, proving the repair/recovery
// steps are reachable (not permanently stuck) once actually performed.
func TestOverdueCriticalFacility_CanCompleteRepairFlow(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityCritical, domain.FacilityOverdue)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, -2))
	submitMaintenance(t, ports, plan, facility, now, "overdue-critical-flow")
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityUnderRepair {
		t.Fatalf("expected under_repair after maintenance, got %s", got)
	}

	// Repair done -> recover -> normal, all legal under the state machine.
	facSvc := NewFacilityService(ports)
	if _, err := facSvc.TransitionState(context.Background(), supervisorActor(), facility.ID, domain.FacilityRecovered, "维修完成"); err != nil {
		t.Fatalf("under_repair->recovered: %v", err)
	}
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityRecovered {
		t.Fatalf("expected recovered, got %s", got)
	}
	if _, err := facSvc.TransitionState(context.Background(), supervisorActor(), facility.ID, domain.FacilityNormal, "恢复确认"); err != nil {
		t.Fatalf("recovered->normal: %v", err)
	}
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityNormal {
		t.Fatalf("expected normal after full recovery, got %s", got)
	}
}
