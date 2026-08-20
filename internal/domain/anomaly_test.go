package domain

import "testing"

// TestReinspectionStatus verifies the central invariant of the follow-up
// inspection flow: only a passing reinspection advances the anomaly to
// recovered; a failing one reopens it. This keeps the issue open until a
// follow-up inspection actually passes.
func TestReinspectionStatus(t *testing.T) {
	if got := ReinspectionStatus(true); got != AnomalyRecovered {
		t.Fatalf("pass=true: expected %s, got %s", AnomalyRecovered, got)
	}
	if got := ReinspectionStatus(false); got != AnomalyOpen {
		t.Fatalf("pass=false: expected %s (issue stays open), got %s", AnomalyOpen, got)
	}
}

// TestAnomalyStatusTransition_Allowed covers the allowed anomaly transitions.
func TestAnomalyStatusTransition_Allowed(t *testing.T) {
	cases := []struct {
		name string
		from AnomalyStatus
		to   AnomalyStatus
	}{
		{"open->reinspecting (rectify)", AnomalyOpen, AnomalyReinspecting},
		{"reinspecting->recovered (reinspect pass)", AnomalyReinspecting, AnomalyRecovered},
		{"reinspecting->open (reinspect fail)", AnomalyReinspecting, AnomalyOpen},
		{"recovered->recovered (confirm, idempotent)", AnomalyRecovered, AnomalyRecovered},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := AnomalyStatusTransition{From: tc.from, To: tc.to}
			if err := tr.Validate(); err != nil {
				t.Fatalf("expected allowed, got: %v", err)
			}
		})
	}
}

// TestAnomalyStatusTransition_Forbidden covers the forbidden transitions. The
// key invariant enforced here is that an anomaly cannot reach recovered from
// open directly (it must go through a passing reinspection), and a failed
// reinspection can never yield recovered.
func TestAnomalyStatusTransition_Forbidden(t *testing.T) {
	cases := []struct {
		name string
		from AnomalyStatus
		to   AnomalyStatus
	}{
		// Recovery confirmation requires a prior passing reinspection.
		{"open->recovered (no passing reinspection)", AnomalyOpen, AnomalyRecovered},
		// A failed reinspection must reopen to open, never jump to recovered.
		{"reinspecting->reinspecting", AnomalyReinspecting, AnomalyReinspecting},
		// An open anomaly cannot be re-discovered into the same state.
		{"open->open", AnomalyOpen, AnomalyOpen},
		// Reserved/terminal starting points have no outgoing transitions.
		{"rectifying->recovered", AnomalyRectifying, AnomalyRecovered},
		{"closed_no_action->recovered", AnomalyClosedNoAction, AnomalyRecovered},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := AnomalyStatusTransition{From: tc.from, To: tc.to}
			err := tr.Validate()
			if err == nil {
				t.Fatalf("expected forbidden transition, got nil")
			}
			if !IsDomainError(err, CodeStateForbidden) {
				t.Fatalf("expected %s, got: %v", CodeStateForbidden, err)
			}
		})
	}
}
