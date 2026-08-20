package domain

import "testing"

// TestMaintenanceCompletionStatus covers the post-maintenance target status.
// The ordinary flow returns Normal; only an overdue CRITICAL facility is routed
// into UnderRepair so its repair and recovery steps are not skipped.
func TestMaintenanceCompletionStatus(t *testing.T) {
	cases := []struct {
		name     string
		facility *Facility
		want     FacilityStatus
	}{
		{"nil_facility_defaults_normal", nil, FacilityNormal},
		{"pending_standard_normal", &Facility{Status: FacilityPendingMaintenance, Criticality: CriticalityStandard}, FacilityNormal},
		{"pending_critical_normal", &Facility{Status: FacilityPendingMaintenance, Criticality: CriticalityCritical}, FacilityNormal},
		{"pending_important_normal", &Facility{Status: FacilityPendingMaintenance, Criticality: CriticalityImportant}, FacilityNormal},
		{"overdue_standard_normal", &Facility{Status: FacilityOverdue, Criticality: CriticalityStandard}, FacilityNormal},
		{"overdue_important_normal", &Facility{Status: FacilityOverdue, Criticality: CriticalityImportant}, FacilityNormal},
		{"overdue_critical_under_repair", &Facility{Status: FacilityOverdue, Criticality: CriticalityCritical}, FacilityUnderRepair},
		{"normal_critical_unchanged_target", &Facility{Status: FacilityNormal, Criticality: CriticalityCritical}, FacilityNormal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaintenanceCompletionStatus(tc.facility); got != tc.want {
				t.Errorf("MaintenanceCompletionStatus = %s, want %s", got, tc.want)
			}
		})
	}
}

// TestMaintenanceCompletionTransition asserts that the transition returned to
// Submit is always a legal state-machine move. In particular the overdue critical
// case must route to UnderRepair (not Normal), and every eligible case must be
// reported as valid so the application layer applies it.
func TestMaintenanceCompletionTransition(t *testing.T) {
	cases := []struct {
		name      string
		facility  *Facility
		wantTo    FacilityStatus
		wantValid bool
	}{
		{
			"pending_standard_to_normal",
			&Facility{Status: FacilityPendingMaintenance, Criticality: CriticalityStandard},
			FacilityNormal, true,
		},
		{
			"pending_critical_to_normal",
			&Facility{Status: FacilityPendingMaintenance, Criticality: CriticalityCritical},
			FacilityNormal, true,
		},
		{
			"overdue_standard_to_normal",
			&Facility{Status: FacilityOverdue, Criticality: CriticalityStandard},
			FacilityNormal, true,
		},
		{
			"overdue_critical_to_under_repair",
			&Facility{Status: FacilityOverdue, Criticality: CriticalityCritical},
			FacilityUnderRepair, true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr, ok := MaintenanceCompletionTransition(tc.facility)
			if ok != tc.wantValid {
				t.Fatalf("valid = %v, want %v", ok, tc.wantValid)
			}
			if tr.To != tc.wantTo {
				t.Errorf("To = %s, want %s", tr.To, tc.wantTo)
			}
			if tr.From != tc.facility.Status {
				t.Errorf("From = %s, want %s", tr.From, tc.facility.Status)
			}
			if tr.Reason == "" {
				t.Errorf("Reason should not be empty")
			}
			// Belt-and-suspenders: the returned transition must independently
			// satisfy ValidateTransition.
			if err := tr.ValidateTransition(tc.facility.Criticality); err != nil {
				t.Errorf("returned transition is not state-machine legal: %v", err)
			}
		})
	}
}

// TestMaintenanceCompletionTransition_NilGuardsAgainstPanic ensures a nil
// facility is handled defensively rather than panicking.
func TestMaintenanceCompletionTransition_NilGuardsAgainstPanic(t *testing.T) {
	tr, ok := MaintenanceCompletionTransition(nil)
	if ok {
		t.Fatalf("expected invalid for nil facility, got valid")
	}
	if tr.To != "" || tr.From != "" {
		t.Errorf("expected zero transition for nil facility, got %+v", tr)
	}
}
