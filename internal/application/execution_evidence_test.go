package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

func TestSubmittedExecutionEvidenceIsDetached(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	plan := setupPlan(t, ports, facility.ID, template.ID, now.AddDate(0, 0, 30))
	pressure := 0.5
	in := domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     facility.ID,
		ExecutedBy:     operatorActor().ID,
		ExecutedAt:     now,
		IdempotencyKey: "evidence-detached",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Value: "clean", Pass: true},
			{ItemCode: "refrigerant", Value: "normal", NumericValue: &pressure, Pass: true},
		},
		Photos:              []domain.PhotoAttachment{{Path: "evidence/a.jpg", MimeType: "image/jpeg", Size: 128, Checksum: "sha256:a"}},
		ConsumablesConsumed: []domain.ConsumableConsumption{{Code: "filter-pad", Quantity: 1}},
	}
	svc := NewExecutionService(ports)
	created, err := svc.Submit(context.Background(), operatorActor(), in)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	in.InspectionValues[0].Value = "caller-overwrite"
	pressure = 0.4
	in.Photos[0].Path = "evidence/replaced.jpg"
	in.ConsumablesConsumed[0].Quantity = 9
	created.InspectionValues[1].Value = "response-overwrite"
	created.Photos[0].Checksum = "sha256:changed"
	stored, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.InspectionValues[0].Value != "clean" || stored.InspectionValues[1].Value != "normal" {
		t.Fatalf("inspection evidence changed after submit: %#v", stored.InspectionValues)
	}
	if stored.InspectionValues[1].NumericValue == nil || *stored.InspectionValues[1].NumericValue != 0.5 {
		t.Fatalf("numeric evidence changed after submit: %#v", stored.InspectionValues[1].NumericValue)
	}
	if stored.Photos[0].Path != "evidence/a.jpg" || stored.Photos[0].Checksum != "sha256:a" {
		t.Fatalf("photo evidence changed after submit: %#v", stored.Photos[0])
	}
	if stored.ConsumablesConsumed[0].Quantity != 1 {
		t.Fatalf("consumable evidence changed after submit: %#v", stored.ConsumablesConsumed[0])
	}
}
