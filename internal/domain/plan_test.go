package domain

import "testing"

// TestPlanTemplateInput_Validate covers the validation rules for plan templates.
func TestPlanTemplateInput_Validate(t *testing.T) {
	cases := []struct {
		name    string
		in      PlanTemplateInput
		wantErr bool
		field   string
	}{
		{
			name:    "ok",
			in:      PlanTemplateInput{Name: "模板1", CycleDays: 30, InspectionItems: []InspectionItem{{Code: "a", Name: "A"}}, Consumables: []ConsumableSpec{{Code: "c", Name: "C", Quantity: 1}}},
			wantErr: false,
		},
		{
			name:    "name too short",
			in:      PlanTemplateInput{Name: "a"},
			wantErr: true,
			field:   "name",
		},
		{
			name:    "cycle_days out of range",
			in:      PlanTemplateInput{Name: "模板", CycleDays: 400},
			wantErr: true,
			field:   "cycle_days",
		},
		{
			name:    "duplicate inspection code",
			in:      PlanTemplateInput{Name: "模板", CycleDays: 30, InspectionItems: []InspectionItem{{Code: "a", Name: "A"}, {Code: "a", Name: "B"}}},
			wantErr: true,
		},
		{
			name:    "min greater than max",
			in:      PlanTemplateInput{Name: "模板", CycleDays: 30, InspectionItems: []InspectionItem{{Code: "a", Name: "A", MinValue: ptrF(10), MaxValue: ptrF(5)}}},
			wantErr: true,
		},
		{
			name:    "consumable quantity < 1",
			in:      PlanTemplateInput{Name: "模板", CycleDays: 30, Consumables: []ConsumableSpec{{Code: "c", Name: "C", Quantity: 0}}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.in.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tc.wantErr && tc.field != "" {
				de, ok := err.(*Error)
				if !ok {
					t.Fatalf("expected *domain.Error, got %T", err)
				}
				if de.Field != tc.field {
					t.Fatalf("expected field %q, got %q", tc.field, de.Field)
				}
			}
		})
	}
}

func ptrF(v float64) *float64 { return &v }
