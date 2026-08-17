package domain

import "testing"

// TestFacilityInput_Validate covers field-level validation for facilities.
func TestFacilityInput_Validate(t *testing.T) {
	valid := FacilityInput{
		PlaceID:             "p1",
		Name:                "空调",
		Code:                "AC-001",
		Category:            "空调",
		ResponsiblePersonID: "rp-1",
		Criticality:         CriticalityCritical,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}
	cases := []struct {
		name string
		mut  func(FacilityInput) FacilityInput
	}{
		{"empty place_id", func(i FacilityInput) FacilityInput { i.PlaceID = ""; return i }},
		{"name too short", func(i FacilityInput) FacilityInput { i.Name = "a"; return i }},
		{"code invalid pattern", func(i FacilityInput) FacilityInput { i.Code = "abc"; return i }},
		{"category too short", func(i FacilityInput) FacilityInput { i.Category = "a"; return i }},
		{"empty responsible_person_id", func(i FacilityInput) FacilityInput { i.ResponsiblePersonID = ""; return i }},
		{"invalid criticality", func(i FacilityInput) FacilityInput { i.Criticality = "bad"; return i }},
		{"description too long", func(i FacilityInput) FacilityInput { i.Description = string(make([]byte, 2048)); return i }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.mut(valid)
			if err := in.Validate(); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

// TestPlaceInput_Validate covers place input validation.
func TestPlaceInput_Validate(t *testing.T) {
	valid := PlaceInput{Name: "图书馆", Type: PlaceTypeLibrary, Address: "北京路 1 号", Code: "LIB-001"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}
	cases := []struct {
		name string
		mut  func(PlaceInput) PlaceInput
	}{
		{"name too short", func(i PlaceInput) PlaceInput { i.Name = "a"; return i }},
		{"invalid type", func(i PlaceInput) PlaceInput { i.Type = "bad"; return i }},
		{"empty address", func(i PlaceInput) PlaceInput { i.Address = ""; return i }},
		{"code lower case", func(i PlaceInput) PlaceInput { i.Code = "abc"; return i }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.mut(valid)
			if err := in.Validate(); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

// TestResponsiblePersonInput_Validate covers person input validation.
func TestResponsiblePersonInput_Validate(t *testing.T) {
	valid := ResponsiblePersonInput{Name: "张三", Email: "zhangsan@example.com", Phone: "13800138000"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}
	cases := []struct {
		name string
		mut  func(ResponsiblePersonInput) ResponsiblePersonInput
	}{
		{"name too short", func(i ResponsiblePersonInput) ResponsiblePersonInput { i.Name = "a"; return i }},
		{"invalid email", func(i ResponsiblePersonInput) ResponsiblePersonInput { i.Email = "bad"; return i }},
		{"invalid phone", func(i ResponsiblePersonInput) ResponsiblePersonInput { i.Phone = "123"; return i }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.mut(valid)
			if err := in.Validate(); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

// TestExecutionInput_Validate covers execution input validation, including
// idempotency_key requirement.
func TestExecutionInput_Validate(t *testing.T) {
	valid := ExecutionInput{
		PlanID:         "plan-1",
		FacilityID:     "fac-1",
		ExecutedBy:     "rp-1",
		ExecutedAt:     mustParseTime("2026-01-01T00:00:00Z"),
		IdempotencyKey: "idem-1",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}
	if err := (ExecutionInput{PlanID: "p", FacilityID: "f", ExecutedBy: "x", ExecutedAt: mustParseTime("2026-01-01T00:00:00Z")}).Validate(); err == nil {
		t.Fatalf("expected idempotency_key required error")
	}
}
