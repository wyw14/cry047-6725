package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

func TestFailedReinspectionCannotEnterRecoveryConfirmation(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityCritical, domain.FacilityNormal)
	svc := NewAnomalyService(ports)
	anomaly, err := svc.Discover(context.Background(), operatorActor(), domain.AnomalyInput{
		FacilityID:     facility.ID,
		DiscoveredBy:   operatorActor().ID,
		DiscoveredAt:   now,
		Description:    "配电柜温升异常",
		Severity:       domain.AnomalySeverityCritical,
		IdempotencyKey: "failed-reinspection-chain",
	})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if _, err := svc.Rectify(context.Background(), supervisorActor(), domain.RectificationInput{
		AnomalyID: anomaly.ID, Measure: "紧固接点并降载", RectifiedBy: supervisorActor().ID,
	}); err != nil {
		t.Fatalf("rectify: %v", err)
	}
	result, err := svc.Reinspect(context.Background(), supervisorActor(), domain.ReinspectionInput{
		AnomalyID: anomaly.ID, Result: "温升仍超限", ReinspectedBy: supervisorActor().ID, ReinspectedAt: now.Add(time.Hour),
	}, false)
	if err != nil {
		t.Fatalf("reinspect: %v", err)
	}
	if result.Status != domain.AnomalyOpen {
		t.Fatalf("failed reinspection entered %s, want open", result.Status)
	}
	if _, err := svc.Recover(context.Background(), supervisorActor(), domain.RecoverInput{
		AnomalyID: anomaly.ID, RecoveredBy: supervisorActor().ID,
	}); err == nil {
		t.Fatalf("failed reinspection unexpectedly allowed recovery confirmation")
	}
	storedFacility, err := ports.Facilities.Get(context.Background(), facility.ID)
	if err != nil {
		t.Fatalf("get facility: %v", err)
	}
	if storedFacility.Status != domain.FacilityUnderRepair {
		t.Fatalf("facility left under_repair after failed reinspection: %s", storedFacility.Status)
	}
}
