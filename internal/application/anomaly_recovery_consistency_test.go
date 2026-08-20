package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TestReinspectionFailedStaysOpenUntilPass exercises the full "keep the issue
// open until a follow-up inspection passes" contract end to end:
//
//  1. A critical anomaly is discovered -> facility restricted_use.
//  2. Rectification -> anomaly reinspecting, facility under_repair.
//  3. A FAILING reinspection reopens the anomaly (open) and leaves the
//     facility under_repair; recovery confirmation is refused.
//  4. A second rectification cycle and a PASSING reinspection finally advance
//     the anomaly to recovered; the facility is still under_repair (recovery
//     not yet confirmed).
//  5. Recovery confirmation is the only step that marks the facility
//     recovered — consistent across every surface.
func TestReinspectionFailedStaysOpenUntilPass(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityCritical, domain.FacilityNormal)
	svc := NewAnomalyService(ports)
	ctx := context.Background()

	// 1. Discover: critical severity auto-transitions facility to restricted_use.
	anomaly, err := svc.Discover(ctx, operatorActor(), domain.AnomalyInput{
		FacilityID:     facility.ID,
		DiscoveredBy:   operatorActor().ID,
		DiscoveredAt:   now,
		Description:    "配电柜温升异常",
		Severity:       domain.AnomalySeverityCritical,
		IdempotencyKey: "keep-open-until-pass",
	})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityRestrictedUse {
		t.Fatalf("after discover: facility %s, want restricted_use", got)
	}

	// 2. Rectify -> reinspecting, facility under_repair.
	if _, err := svc.Rectify(ctx, supervisorActor(), domain.RectificationInput{
		AnomalyID: anomaly.ID, Measure: "紧固接点并降载", RectifiedBy: supervisorActor().ID,
	}); err != nil {
		t.Fatalf("rectify: %v", err)
	}
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityUnderRepair {
		t.Fatalf("after rectify: facility %s, want under_repair", got)
	}

	// 3. Failed reinspection: anomaly reopens, facility stays under_repair,
	//    recovery confirmation is refused.
	failed, err := svc.Reinspect(ctx, supervisorActor(), domain.ReinspectionInput{
		AnomalyID: anomaly.ID, Result: "温升仍超限", ReinspectedBy: supervisorActor().ID, ReinspectedAt: now.Add(time.Hour),
	}, false)
	if err != nil {
		t.Fatalf("reinspect (fail): %v", err)
	}
	if failed.Status != domain.AnomalyOpen {
		t.Fatalf("failed reinspection entered %s, want open (issue must stay open)", failed.Status)
	}
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityUnderRepair {
		t.Fatalf("after failed reinspection: facility %s, want under_repair", got)
	}
	if _, err := svc.Recover(ctx, supervisorActor(), domain.RecoverInput{
		AnomalyID: anomaly.ID, RecoveredBy: supervisorActor().ID,
	}); err == nil {
		t.Fatalf("recovery confirmation must be refused after a failed reinspection")
	}

	// 4. Second rectification cycle then a passing reinspection.
	if _, err := svc.Rectify(ctx, supervisorActor(), domain.RectificationInput{
		AnomalyID: anomaly.ID, Measure: "更换绕组并复测", RectifiedBy: supervisorActor().ID,
	}); err != nil {
		t.Fatalf("re-rectify: %v", err)
	}
	passed, err := svc.Reinspect(ctx, supervisorActor(), domain.ReinspectionInput{
		AnomalyID: anomaly.ID, Result: "温升恢复正常", ReinspectedBy: supervisorActor().ID, ReinspectedAt: now.Add(2 * time.Hour),
	}, true)
	if err != nil {
		t.Fatalf("reinspect (pass): %v", err)
	}
	if passed.Status != domain.AnomalyRecovered {
		t.Fatalf("passing reinspection entered %s, want recovered", passed.Status)
	}
	// The facility must NOT be marked recovered merely because the reinspection
	// passed — recovery is not yet confirmed.
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityUnderRepair {
		t.Fatalf("after passing reinspection: facility %s, want under_repair (recovery not yet confirmed)", got)
	}

	// 5. Recovery confirmation is the single point that marks the facility
	//    recovered, keeping the visible status consistent everywhere.
	if _, err := svc.Recover(ctx, supervisorActor(), domain.RecoverInput{
		AnomalyID: anomaly.ID, RecoveredBy: supervisorActor().ID,
	}); err != nil {
		t.Fatalf("recover: %v", err)
	}
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityRecovered {
		t.Fatalf("after recover: facility %s, want recovered", got)
	}
}

// TestReinspectPass_DoesNotPrematurelyMarkFacilityRecovered verifies the
// consistency invariant directly: a passing reinspection advances the anomaly
// to recovered but never the facility. The facility only becomes recovered on
// explicit recovery confirmation.
func TestReinspectPass_DoesNotPrematurelyMarkFacilityRecovered(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityCritical, domain.FacilityNormal)
	svc := NewAnomalyService(ports)
	ctx := context.Background()

	anomaly, err := svc.Discover(ctx, operatorActor(), domain.AnomalyInput{
		FacilityID:     facility.ID,
		DiscoveredBy:   operatorActor().ID,
		DiscoveredAt:   now,
		Description:    "机组异响",
		Severity:       domain.AnomalySeverityCritical,
		IdempotencyKey: "no-premature-recovered",
	})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if _, err := svc.Rectify(ctx, supervisorActor(), domain.RectificationInput{
		AnomalyID: anomaly.ID, Measure: "更换轴承", RectifiedBy: supervisorActor().ID,
	}); err != nil {
		t.Fatalf("rectify: %v", err)
	}
	if _, err := svc.Reinspect(ctx, supervisorActor(), domain.ReinspectionInput{
		AnomalyID: anomaly.ID, Result: "复检通过", ReinspectedBy: supervisorActor().ID, ReinspectedAt: now.Add(time.Hour),
	}, true); err != nil {
		t.Fatalf("reinspect: %v", err)
	}
	// Facility stays under_repair after a passing reinspection: the visible
	// "recovered" status must imply a confirmed recovery, not a mere pass.
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityUnderRepair {
		t.Fatalf("after passing reinspection: facility %s, want under_repair (no premature recovered)", got)
	}

	// Confirm recovery -> now the facility is recovered.
	if _, err := svc.Recover(ctx, supervisorActor(), domain.RecoverInput{
		AnomalyID: anomaly.ID, RecoveredBy: supervisorActor().ID,
	}); err != nil {
		t.Fatalf("recover: %v", err)
	}
	if got := facilityStatus(t, ports, facility.ID); got != domain.FacilityRecovered {
		t.Fatalf("after recover: facility %s, want recovered", got)
	}
}

// facilityStatus is a small helper that fetches the current stored facility
// status, asserting that the persisted status is the one shown everywhere.
func facilityStatus(t *testing.T, ports *domain.Ports, id string) domain.FacilityStatus {
	t.Helper()
	f, err := ports.Facilities.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get facility: %v", err)
	}
	return f.Status
}
