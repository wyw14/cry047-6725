package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TestOverdueCriticalFacilityCannotBeMarkedNormal enforces the prompt invariant:
// "逾期关键设施不能被标记为正常".
func TestOverdueCriticalFacilityCannotBeMarkedNormal(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityCritical, domain.FacilityOverdue)
	svc := NewFacilityService(ports)
	// Attempt to transition overdue -> normal must fail.
	if _, err := svc.TransitionState(context.Background(), supervisorActor(), f.ID, domain.FacilityNormal, "绕过流程"); err == nil {
		t.Fatalf("expected STATE_FORBIDDEN error, got nil")
	}
}

// TestOverdueCriticalFacilityRecoveryFlow verifies the full recovery flow:
// overdue -> under_repair -> recovered -> normal.
func TestOverdueCriticalFacilityRecoveryFlow(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityCritical, domain.FacilityOverdue)
	svc := NewFacilityService(ports)
	// overdue -> under_repair
	f1, err := svc.TransitionState(context.Background(), operatorActor(), f.ID, domain.FacilityUnderRepair, "开始维修")
	if err != nil {
		t.Fatalf("overdue->under_repair: %v", err)
	}
	if f1.Status != domain.FacilityUnderRepair {
		t.Fatalf("expected under_repair, got %s", f1.Status)
	}
	// under_repair -> recovered
	f2, err := svc.TransitionState(context.Background(), supervisorActor(), f.ID, domain.FacilityRecovered, "维修完成")
	if err != nil {
		t.Fatalf("under_repair->recovered: %v", err)
	}
	if f2.Status != domain.FacilityRecovered {
		t.Fatalf("expected recovered, got %s", f2.Status)
	}
	// recovered -> normal
	f3, err := svc.TransitionState(context.Background(), supervisorActor(), f.ID, domain.FacilityNormal, "恢复确认")
	if err != nil {
		t.Fatalf("recovered->normal: %v", err)
	}
	if f3.Status != domain.FacilityNormal {
		t.Fatalf("expected normal, got %s", f3.Status)
	}
}

// TestRestrictedCriticalFacilityCannotBeMarkedNormal enforces the related
// invariant: critical restricted-use facilities cannot jump to normal.
func TestRestrictedCriticalFacilityCannotBeMarkedNormal(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityCritical, domain.FacilityRestrictedUse)
	svc := NewFacilityService(ports)
	if _, err := svc.TransitionState(context.Background(), supervisorActor(), f.ID, domain.FacilityNormal, "绕过"); err == nil {
		t.Fatalf("expected forbidden, got nil")
	}
}

// TestStandardCriticalityOverdueCanRecoverDirect verifies that the invariant
// ONLY applies to critical facilities; a standard criticality overdue facility
// can be marked normal directly (it is not "critical").
func TestStandardCriticalityOverdueCanRecoverDirect(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityOverdue)
	svc := NewFacilityService(ports)
	if _, err := svc.TransitionState(context.Background(), operatorActor(), f.ID, domain.FacilityNormal, "标准设施直接恢复"); err != nil {
		t.Fatalf("expected allowed for standard, got: %v", err)
	}
}

// TestFacilityCreate_RequiresExistingPlaceAndPerson verifies cross-aggregate
// invariants: facility creation requires an existing place and person.
func TestFacilityCreate_RequiresExistingPlaceAndPerson(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	svc := NewFacilityService(ports)
	in := domain.FacilityInput{
		PlaceID:             "missing",
		Name:                "X",
		Code:                "X-001",
		Category:            "X",
		ResponsiblePersonID: "missing",
		Criticality:         domain.CriticalityCritical,
	}
	if _, err := svc.Create(context.Background(), adminActor(), in); err == nil {
		t.Fatalf("expected NOT_FOUND for missing place, got nil")
	}
}

// TestTransitionState_ViewerForbidden verifies the role boundary: viewers
// cannot perform state transitions.
func TestTransitionState_ViewerForbidden(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityCritical, domain.FacilityNormal)
	svc := NewFacilityService(ports)
	_, err := svc.TransitionState(context.Background(), viewerActor(), f.ID, domain.FacilityPendingMaintenance, "")
	if err == nil {
		t.Fatalf("expected forbidden for viewer, got nil")
	}
	if !domain.IsDomainError(err, domain.CodeForbidden) {
		t.Fatalf("expected FORBIDDEN, got: %v", err)
	}
}
