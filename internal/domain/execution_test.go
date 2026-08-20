package domain

import (
	"testing"
)

// ptrFloat64 returns a pointer to v. Mirrors the helper in the application
// tests so the domain tests stay self-contained.
func ptrFloat64(v float64) *float64 { return &v }

// TestExecution_Clone_IsDeepCopy verifies that Clone produces a structurally
// equal but fully independent copy: mutating the clone's slices (or the
// original's) never reaches the other, and the NumericValue *float64 pointers
// are re-pointed at distinct allocations. This is the core guarantee that keeps
// persisted evidence immune to later mutation of request data or response
// objects.
func TestExecution_Clone_IsDeepCopy(t *testing.T) {
	pressure := 0.5
	orig := &Execution{
		ID:         "exec-1",
		PlanID:     "plan-1",
		FacilityID: "fac-1",
		Status:     ExecutionSubmitted,
		InspectionValues: []InspectionValue{
			{ItemCode: "filter", Value: "clean", Pass: true},
			{ItemCode: "refrigerant", NumericValue: &pressure, Pass: true},
		},
		Photos:              []PhotoAttachment{{Path: "evidence/a.jpg", Checksum: "sha256:a", Size: 128}},
		ConsumablesConsumed: []ConsumableConsumption{{Code: "filter-pad", Quantity: 1}},
	}

	cp := orig.Clone()

	// Structural equality (deep) for the evidence we care about.
	if len(cp.InspectionValues) != 2 || cp.InspectionValues[0].Value != "clean" {
		t.Fatalf("clone lost inspection values: %#v", cp.InspectionValues)
	}
	if cp.InspectionValues[1].NumericValue == nil || *cp.InspectionValues[1].NumericValue != 0.5 {
		t.Fatalf("clone lost numeric value: %#v", cp.InspectionValues[1].NumericValue)
	}
	if len(cp.Photos) != 1 || cp.Photos[0].Path != "evidence/a.jpg" {
		t.Fatalf("clone lost photos: %#v", cp.Photos)
	}
	if len(cp.ConsumablesConsumed) != 1 || cp.ConsumablesConsumed[0].Quantity != 1 {
		t.Fatalf("clone lost consumables: %#v", cp.ConsumablesConsumed)
	}

	// Mutate the clone; the original must be unaffected.
	cp.InspectionValues[0].Value = "clone-mutated"
	*cp.InspectionValues[1].NumericValue = 0.9
	cp.Photos[0].Checksum = "sha256:clone"
	cp.ConsumablesConsumed[0].Quantity = 99
	if orig.InspectionValues[0].Value != "clean" {
		t.Errorf("mutating clone reached original slice: %#v", orig.InspectionValues[0])
	}
	if orig.InspectionValues[1].NumericValue == nil || *orig.InspectionValues[1].NumericValue != 0.5 {
		t.Errorf("mutating clone's NumericValue reached original pointer: %#v", orig.InspectionValues[1].NumericValue)
	}
	if orig.Photos[0].Checksum != "sha256:a" {
		t.Errorf("mutating clone reached original photo: %#v", orig.Photos[0])
	}
	if orig.ConsumablesConsumed[0].Quantity != 1 {
		t.Errorf("mutating clone reached original consumable: %#v", orig.ConsumablesConsumed[0])
	}

	// Mutate the original; the clone must be unaffected (independence is
	// bidirectional).
	orig.InspectionValues[0].Value = "orig-mutated"
	pressure = 0.1 // the original still holds &pressure
	if cp.InspectionValues[0].Value != "clone-mutated" {
		t.Errorf("mutating original reached clone slice: %#v", cp.InspectionValues[0])
	}
	if cp.InspectionValues[1].NumericValue == nil || *cp.InspectionValues[1].NumericValue != 0.9 {
		t.Errorf("mutating original's NumericValue reached clone pointer: %#v", cp.InspectionValues[1].NumericValue)
	}
}

// TestExecution_Snapshot_Deep verifies Snapshot delegates to Clone and is
// therefore a deep copy, not a shallow struct copy.
func TestExecution_Snapshot_Deep(t *testing.T) {
	pressure := 0.5
	orig := &Execution{
		InspectionValues: []InspectionValue{
			{ItemCode: "filter", Value: "clean", Pass: true},
			{ItemCode: "refrigerant", NumericValue: &pressure, Pass: true},
		},
	}
	snap := orig.Snapshot()
	snap.InspectionValues[0].Value = "snap-mutated"
	*snap.InspectionValues[1].NumericValue = 0.9
	if orig.InspectionValues[0].Value != "clean" {
		t.Errorf("Snapshot is shallow: mutating snapshot reached original slice: %#v", orig.InspectionValues[0])
	}
	if *orig.InspectionValues[1].NumericValue != 0.5 {
		t.Errorf("Snapshot is shallow: mutating snapshot reached original pointer: %v", *orig.InspectionValues[1].NumericValue)
	}
}

// TestExecution_Clone_NilHandling covers the nil-receiver and nil-slice cases
// so the helpers do not regress into panics or lose the nil-vs-empty distinction
// that JSON serialization depends on.
func TestExecution_Clone_NilHandling(t *testing.T) {
	if got := (*Execution)(nil).Clone(); got != nil {
		t.Errorf("nil receiver Clone should be nil, got %#v", got)
	}
	if got := (*Execution)(nil).Snapshot(); got != nil {
		t.Errorf("nil receiver Snapshot should be nil, got %#v", got)
	}
	empty := &Execution{InspectionValues: []InspectionValue{}}
	got := empty.Clone()
	if got.InspectionValues == nil || len(got.InspectionValues) != 0 {
		t.Errorf("non-nil empty slice should stay non-nil empty, got %#v", got.InspectionValues)
	}
	nilSlices := &Execution{}
	gotNil := nilSlices.Clone()
	if gotNil.InspectionValues != nil {
		t.Errorf("nil slice should stay nil, got %#v", gotNil.InspectionValues)
	}
	if gotNil.Photos != nil {
		t.Errorf("nil slice should stay nil, got %#v", gotNil.Photos)
	}
	if gotNil.ConsumablesConsumed != nil {
		t.Errorf("nil slice should stay nil, got %#v", gotNil.ConsumablesConsumed)
	}
}

// TestCloneInspectionValues_DetachesPointer verifies the slice helper re-points
// NumericValue rather than copying the pointer value.
func TestCloneInspectionValues_DetachesPointer(t *testing.T) {
	pressure := 0.5
	in := []InspectionValue{{ItemCode: "x", NumericValue: &pressure}}
	out := CloneInspectionValues(in)
	*out[0].NumericValue = 0.9
	if pressure != 0.5 {
		t.Errorf("CloneInspectionValues aliased NumericValue pointer: original now %v", pressure)
	}
	// nil preserved.
	if got := CloneInspectionValues(nil); got != nil {
		t.Errorf("CloneInspectionValues(nil) should be nil, got %#v", got)
	}
}
