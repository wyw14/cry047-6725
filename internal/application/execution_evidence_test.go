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

// TestIdempotentResubmitEvidenceStaysDetached verifies the second half of the
// fix: repeated submissions carrying the same idempotency key must still return
// the originally-saved record (idempotency is preserved), AND the records handed
// back on both the first and the second call must be fully detached from
// persisted state — mutating them, or re-submitting with mutated request data,
// cannot alter the saved evidence.
func TestIdempotentResubmitEvidenceStaysDetached(t *testing.T) {
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
		IdempotencyKey: "evidence-idempotent",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Value: "clean", Pass: true},
			{ItemCode: "refrigerant", Value: "normal", NumericValue: &pressure, Pass: true},
		},
		Photos:              []domain.PhotoAttachment{{Path: "evidence/a.jpg", MimeType: "image/jpeg", Size: 128, Checksum: "sha256:a"}},
		ConsumablesConsumed: []domain.ConsumableConsumption{{Code: "filter-pad", Quantity: 1}},
	}
	svc := NewExecutionService(ports)

	first, err := svc.Submit(context.Background(), operatorActor(), in)
	if err != nil {
		t.Fatalf("submit 1: %v", err)
	}

	// Re-submit with the same idempotency key but mutated request data. The
	// idempotent path must return the ORIGINAL evidence, ignoring the mutated
	// payload entirely.
	in.InspectionValues[0].Value = "tampered"
	pressure = 0.1
	in.Photos[0].Path = "evidence/tampered.jpg"
	in.ConsumablesConsumed[0].Quantity = 7
	second, err := svc.Submit(context.Background(), operatorActor(), in)
	if err != nil {
		t.Fatalf("submit 2: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("idempotent resubmit should return same record id, got %s vs %s", second.ID, first.ID)
	}
	list, err := ports.Executions.List(context.Background(), domain.PageQuery{Limit: 100})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("idempotent resubmit should not create a new record, got %d", len(list.Items))
	}

	// Mutating both returned response objects must not affect saved evidence.
	first.InspectionValues[0].Value = "first-mutated"
	first.Photos[0].Checksum = "sha256:first"
	second.InspectionValues[1].Value = "second-mutated"
	*second.InspectionValues[1].NumericValue = 9.9

	stored, err := svc.Get(context.Background(), first.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.InspectionValues[0].Value != "clean" || stored.InspectionValues[1].Value != "normal" {
		t.Fatalf("saved inspection evidence changed after idempotent resubmit: %#v", stored.InspectionValues)
	}
	if stored.InspectionValues[1].NumericValue == nil || *stored.InspectionValues[1].NumericValue != 0.5 {
		t.Fatalf("saved numeric evidence changed after idempotent resubmit: %#v", stored.InspectionValues[1].NumericValue)
	}
	if stored.Photos[0].Path != "evidence/a.jpg" || stored.Photos[0].Checksum != "sha256:a" {
		t.Fatalf("saved photo evidence changed after idempotent resubmit: %#v", stored.Photos[0])
	}
	if stored.ConsumablesConsumed[0].Quantity != 1 {
		t.Fatalf("saved consumable evidence changed after idempotent resubmit: %#v", stored.ConsumablesConsumed[0])
	}
}
