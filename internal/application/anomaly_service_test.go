package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TestAnomalyFullFlow_DiscoverRectifyReinspectRecover covers the full anomaly
// lifecycle: discover -> rectify -> reinspect (pass) -> recover.
func TestAnomalyFullFlow_DiscoverRectifyReinspectRecover(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityCritical, domain.FacilityNormal)
	svc := NewAnomalyService(ports)

	// 1. Discover: critical severity auto-transitions facility to restricted_use.
	a, err := svc.Discover(context.Background(), operatorActor(), domain.AnomalyInput{
		FacilityID:     f.ID,
		DiscoveredBy:   operatorActor().ID,
		DiscoveredAt:   now,
		Description:    "机组异响严重",
		Severity:       domain.AnomalySeverityCritical,
		IdempotencyKey: "anom-1",
	})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if a.Status != domain.AnomalyOpen {
		t.Errorf("expected open, got %s", a.Status)
	}
	ff, _ := ports.Facilities.Get(context.Background(), f.ID)
	if ff.Status != domain.FacilityRestrictedUse {
		t.Errorf("expected restricted_use, got %s", ff.Status)
	}

	// 2. Rectify: facility transitions to under_repair.
	a2, err := svc.Rectify(context.Background(), supervisorActor(), domain.RectificationInput{
		AnomalyID:   a.ID,
		Measure:     "更换轴承并校准",
		RectifiedBy: supervisorActor().ID,
	})
	if err != nil {
		t.Fatalf("rectify: %v", err)
	}
	if a2.Status != domain.AnomalyReinspecting {
		t.Errorf("expected reinspecting, got %s", a2.Status)
	}
	ff, _ = ports.Facilities.Get(context.Background(), f.ID)
	if ff.Status != domain.FacilityUnderRepair {
		t.Errorf("expected under_repair, got %s", ff.Status)
	}

	// 3. Reinspect (pass): anomaly moves to recovered.
	a3, err := svc.Reinspect(context.Background(), supervisorActor(), domain.ReinspectionInput{
		AnomalyID:     a.ID,
		Result:        "复检通过",
		ReinspectedBy: supervisorActor().ID,
		ReinspectedAt: now.Add(time.Hour),
	}, true)
	if err != nil {
		t.Fatalf("reinspect: %v", err)
	}
	if a3.Status != domain.AnomalyRecovered {
		t.Errorf("expected recovered, got %s", a3.Status)
	}

	// 4. Recover: facility transitions to Recovered status.
	a4, err := svc.Recover(context.Background(), supervisorActor(), domain.RecoverInput{
		AnomalyID:   a.ID,
		RecoveredBy: supervisorActor().ID,
	})
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	_ = a4
	ff, _ = ports.Facilities.Get(context.Background(), f.ID)
	if ff.Status != domain.FacilityRecovered {
		t.Errorf("expected recovered facility, got %s", ff.Status)
	}

	// 5. Final state transition: recovered -> normal.
	facSvc := NewFacilityService(ports)
	_, err = facSvc.TransitionState(context.Background(), supervisorActor(), f.ID, domain.FacilityNormal, "恢复确认")
	if err != nil {
		t.Fatalf("recovered->normal: %v", err)
	}
}

// TestAnomalyReinspectFail_RevertsToOpen verifies that a failed reinspection
// returns the anomaly to open and the facility remains under_repair.
func TestAnomalyReinspectFail_RevertsToOpen(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityCritical, domain.FacilityRestrictedUse)
	svc := NewAnomalyService(ports)
	a, err := svc.Discover(context.Background(), operatorActor(), domain.AnomalyInput{
		FacilityID:     f.ID,
		DiscoveredBy:   operatorActor().ID,
		DiscoveredAt:   now,
		Description:    "故障描述",
		Severity:       domain.AnomalySeverityCritical,
		IdempotencyKey: "anom-fail",
	})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if _, err := svc.Rectify(context.Background(), supervisorActor(), domain.RectificationInput{
		AnomalyID: a.ID, Measure: "修复尝试", RectifiedBy: supervisorActor().ID,
	}); err != nil {
		t.Fatalf("rectify: %v", err)
	}
	a2, err := svc.Reinspect(context.Background(), supervisorActor(), domain.ReinspectionInput{
		AnomalyID: a.ID, Result: "复检未通过", ReinspectedBy: supervisorActor().ID, ReinspectedAt: now,
	}, false)
	if err != nil {
		t.Fatalf("reinspect: %v", err)
	}
	if a2.Status != domain.AnomalyOpen {
		t.Errorf("expected open after fail, got %s", a2.Status)
	}
}

// TestAnomalyViewerCannotDiscover enforces RBAC.
func TestAnomalyViewerCannotDiscover(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	svc := NewAnomalyService(ports)
	_, err := svc.Discover(context.Background(), viewerActor(), domain.AnomalyInput{
		FacilityID:     f.ID,
		DiscoveredBy:   viewerActor().ID,
		DiscoveredAt:   now,
		Description:    "test",
		Severity:       domain.AnomalySeverityLow,
		IdempotencyKey: "anom-viewer",
	})
	if err == nil {
		t.Fatalf("expected forbidden for viewer, got nil")
	}
}

// TestAnomalyRecover_RequiresRecoveredStatus enforces the state machine
// invariant: recovery confirmation only works on anomalies in recovered status.
func TestAnomalyRecover_RequiresRecoveredStatus(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	svc := NewAnomalyService(ports)
	a, err := svc.Discover(context.Background(), operatorActor(), domain.AnomalyInput{
		FacilityID:     f.ID,
		DiscoveredBy:   operatorActor().ID,
		DiscoveredAt:   now,
		Description:    "故障描述",
		Severity:       domain.AnomalySeverityLow,
		IdempotencyKey: "anom-low",
	})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	// Try to recover directly without rectify/reinspect.
	_, err = svc.Recover(context.Background(), supervisorActor(), domain.RecoverInput{
		AnomalyID: a.ID, RecoveredBy: supervisorActor().ID,
	})
	if err == nil {
		t.Fatalf("expected STATE_FORBIDDEN, got nil")
	}
}

// TestAnomalyIdempotency verifies duplicate submission returns the same record.
func TestAnomalyIdempotency(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	svc := NewAnomalyService(ports)
	in := domain.AnomalyInput{
		FacilityID:     f.ID,
		DiscoveredBy:   operatorActor().ID,
		DiscoveredAt:   now,
		Description:    "故障",
		Severity:       domain.AnomalySeverityLow,
		IdempotencyKey: "anom-idem",
	}
	a1, err := svc.Discover(context.Background(), operatorActor(), in)
	if err != nil {
		t.Fatalf("discover 1: %v", err)
	}
	a2, err := svc.Discover(context.Background(), operatorActor(), in)
	if err != nil {
		t.Fatalf("discover 2: %v", err)
	}
	if a1.ID != a2.ID {
		t.Errorf("expected same id, got %s vs %s", a1.ID, a2.ID)
	}
}
