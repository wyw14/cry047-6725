package domain

import (
	"testing"
)

// TestFacilityStatusTransition_Allowed covers the allowed transitions.
func TestFacilityStatusTransition_Allowed(t *testing.T) {
	cases := []struct {
		name        string
		from        FacilityStatus
		to          FacilityStatus
		criticality Criticality
		wantErr     bool
	}{
		{"normal->pending_critical", FacilityNormal, FacilityPendingMaintenance, CriticalityCritical, false},
		{"normal->restricted_critical", FacilityNormal, FacilityRestrictedUse, CriticalityCritical, false},
		{"pending->normal", FacilityPendingMaintenance, FacilityNormal, CriticalityStandard, false},
		{"pending->overdue", FacilityPendingMaintenance, FacilityOverdue, CriticalityStandard, false},
		{"overdue->under_repair", FacilityOverdue, FacilityUnderRepair, CriticalityCritical, false},
		{"overdue->recovered_critical", FacilityOverdue, FacilityRecovered, CriticalityCritical, false},
		{"restricted->under_repair", FacilityRestrictedUse, FacilityUnderRepair, CriticalityCritical, false},
		{"restricted->recovered", FacilityRestrictedUse, FacilityRecovered, CriticalityCritical, false},
		{"under_repair->recovered", FacilityUnderRepair, FacilityRecovered, CriticalityCritical, false},
		{"recovered->normal", FacilityRecovered, FacilityNormal, CriticalityStandard, false},
		{"recovered->pending", FacilityRecovered, FacilityPendingMaintenance, CriticalityCritical, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := FacilityStatusTransition{From: tc.from, To: tc.to}
			err := tr.ValidateTransition(tc.criticality)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

// TestFacilityStatusTransition_Forbidden covers the forbidden transitions,
// with particular attention to the invariant:
//   - 逾期关键设施不能被标记为正常 (overdue critical cannot be marked normal)
//   - 限用关键设施不能直接恢复为正常
func TestFacilityStatusTransition_Forbidden(t *testing.T) {
	cases := []struct {
		name        string
		from        FacilityStatus
		to          FacilityStatus
		criticality Criticality
	}{
		// Overdue critical cannot be marked normal directly.
		{"overdue_critical->normal", FacilityOverdue, FacilityNormal, CriticalityCritical},
		// Restricted-use critical cannot be marked normal directly.
		{"restricted_critical->normal", FacilityRestrictedUse, FacilityNormal, CriticalityCritical},
		// Under_repair cannot jump to normal directly.
		{"under_repair->normal", FacilityUnderRepair, FacilityNormal, CriticalityCritical},
		// Cannot transition to the same state.
		{"same_state", FacilityNormal, FacilityNormal, CriticalityStandard},
		// Normal cannot go to recovered directly.
		{"normal->recovered", FacilityNormal, FacilityRecovered, CriticalityCritical},
		// Pending cannot go to recovered directly.
		{"pending->recovered", FacilityPendingMaintenance, FacilityRecovered, CriticalityCritical},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := FacilityStatusTransition{From: tc.from, To: tc.to}
			err := tr.ValidateTransition(tc.criticality)
			if err == nil {
				t.Fatalf("expected forbidden transition error, got nil")
			}
			if !IsDomainError(err, CodeStateForbidden) {
				t.Fatalf("expected CodeStateForbidden, got: %v", err)
			}
		})
	}
}

// TestFacilityStatusTransition_StandardCriticalityAllowedForOverdue verifies
// that an overdue NON-critical facility can be marked normal (the invariant
// only applies to CRITICAL facilities).
func TestFacilityStatusTransition_StandardCriticalityAllowedForOverdue(t *testing.T) {
	tr := FacilityStatusTransition{From: FacilityOverdue, To: FacilityNormal}
	if err := tr.ValidateTransition(CriticalityStandard); err != nil {
		t.Fatalf("expected allowed for standard criticality, got: %v", err)
	}
	tr = FacilityStatusTransition{From: FacilityRestrictedUse, To: FacilityNormal}
	if err := tr.ValidateTransition(CriticalityStandard); err != nil {
		t.Fatalf("expected allowed for standard criticality, got: %v", err)
	}
}
