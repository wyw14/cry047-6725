package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

func TestChangingCyclePreservesHistoricalVersionSnapshot(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	svc := NewPlanService(ports)
	plan, err := svc.CreatePlan(context.Background(), adminActor(), domain.MaintenancePlanInput{FacilityID: facility.ID, TemplateID: template.ID, StartAt: now})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	if _, err := svc.ChangeCycle(context.Background(), adminActor(), plan.ID, 60, "延长维护周期"); err != nil {
		t.Fatalf("change cycle: %v", err)
	}
	versions, err := svc.ListVersions(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected initial and changed snapshots, got %d", len(versions))
	}
	if versions[0].CycleDays != template.CycleDays {
		t.Fatalf("initial version was rewritten from %d to %d", template.CycleDays, versions[0].CycleDays)
	}
	if versions[1].CycleDays != 60 {
		t.Fatalf("changed version has cycle %d, want 60", versions[1].CycleDays)
	}
}
