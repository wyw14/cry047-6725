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

// TestChangingCyclePreservesEveryIntermediateCycle confirms that a sequence
// of cycle changes leaves the full history intact: each snapshot keeps the
// cycle that was configured at that point, and the live plan reflects only the
// most recent change. This is the core "older audit records must keep showing
// the cycle actually used at the time" invariant.
func TestChangingCyclePreservesEveryIntermediateCycle(t *testing.T) {
	now := time.Now().UTC()
	ports, clock := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports) // cycle 30
	svc := NewPlanService(ports)

	plan, err := svc.CreatePlan(context.Background(), adminActor(), domain.MaintenancePlanInput{
		FacilityID: facility.ID, TemplateID: template.ID, StartAt: now,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	// Stamp each cycle change at a distinct, increasing instant so we can later
	// reconstruct the cycle in effect at any point in time.
	t1 := now.Add(48 * time.Hour)
	clock.T = t1
	if _, err := svc.ChangeCycle(context.Background(), adminActor(), plan.ID, 60, "延长到60天"); err != nil {
		t.Fatalf("change cycle #1: %v", err)
	}
	t2 := now.Add(96 * time.Hour)
	clock.T = t2
	if _, err := svc.ChangeCycle(context.Background(), adminActor(), plan.ID, 90, "延长到90天"); err != nil {
		t.Fatalf("change cycle #2: %v", err)
	}

	versions, err := svc.ListVersions(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(versions))
	}
	wantCycles := []int{template.CycleDays, 60, 90}
	for i, want := range wantCycles {
		if versions[i].CycleDays != want {
			t.Errorf("version[%d] cycle = %d, want %d (history was rewritten)", i, versions[i].CycleDays, want)
		}
		if versions[i].VersionNumber != i+1 {
			t.Errorf("version[%d] number = %d, want %d", i, versions[i].VersionNumber, i+1)
		}
	}

	// The live plan carries only the most recent setting.
	live, err := ports.Plans.GetPlan(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	if live.CurrentCycleDays != 90 {
		t.Errorf("live plan cycle = %d, want 90", live.CurrentCycleDays)
	}

	// EffectiveCycleAt must reconstruct the cycle actually in effect at each
	// instant, proving historical records can still report the right value.
	cases := []struct {
		name string
		at   time.Time
		want int
	}{
		{"before-first-change", now, template.CycleDays},
		{"after-first-before-second", t1.Add(time.Hour), 60},
		{"after-second", t2.Add(time.Hour), 90},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := domain.EffectiveCycleAt(versions, tc.at)
			if !ok {
				t.Fatalf("EffectiveCycleAt returned false")
			}
			if got != tc.want {
				t.Errorf("effective cycle at %v = %d, want %d", tc.at, got, tc.want)
			}
		})
	}
}

// TestAuditService_PlanVersionsAreImmutableAfterChange drives the exact symptom
// from the bug report through the audit accessor: after a cycle change, older
// audit snapshots retrieved via ListPlanVersionsForAudit must still show the
// original cycle, not the freshly configured one.
func TestAuditService_PlanVersionsAreImmutableAfterChange(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	planSvc := NewPlanService(ports)
	auditSvc := NewAuditService(ports)

	plan, err := planSvc.CreatePlan(context.Background(), adminActor(), domain.MaintenancePlanInput{
		FacilityID: facility.ID, TemplateID: template.ID, StartAt: now,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	before, err := auditSvc.ListPlanVersionsForAudit(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("list for audit (before): %v", err)
	}
	if len(before) != 1 || before[0].CycleDays != template.CycleDays {
		t.Fatalf("pre-change audit snapshot: %+v", before)
	}

	if _, err := planSvc.ChangeCycle(context.Background(), adminActor(), plan.ID, 120, "延长到120天"); err != nil {
		t.Fatalf("change cycle: %v", err)
	}

	after, err := auditSvc.ListPlanVersionsForAudit(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("list for audit (after): %v", err)
	}
	if len(after) != 2 {
		t.Fatalf("expected 2 audit snapshots after change, got %d", len(after))
	}
	// The original snapshot must be untouched.
	if after[0].CycleDays != template.CycleDays {
		t.Errorf("audit snapshot[0] cycle = %d, want %d (history was rewritten)", after[0].CycleDays, template.CycleDays)
	}
	// The new setting is recorded as a separate, correct snapshot.
	if after[1].CycleDays != 120 {
		t.Errorf("audit snapshot[1] cycle = %d, want 120", after[1].CycleDays)
	}
}

// TestChangeCycle_KeepsFutureSchedulingCorrect verifies that recording the new
// setting correctly does not break future scheduling: the next due date is
// derived from the new cycle and the most recent execution.
func TestChangeCycle_KeepsFutureSchedulingCorrect(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	placeID, personID := setupPlaceAndPerson(t, ports)
	facility := setupFacility(t, ports, placeID, personID, domain.CriticalityStandard, domain.FacilityNormal)
	template := setupTemplate(t, ports)
	planSvc := NewPlanService(ports)
	execSvc := NewExecutionService(ports)

	plan, err := planSvc.CreatePlan(context.Background(), adminActor(), domain.MaintenancePlanInput{
		FacilityID: facility.ID, TemplateID: template.ID, StartAt: now,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	// Execute once under the original 30-day cycle.
	executedAt := now.AddDate(0, 0, -1)
	if _, err := execSvc.Submit(context.Background(), operatorActor(), domain.ExecutionInput{
		PlanID: plan.ID, FacilityID: facility.ID, ExecutedBy: operatorActor().ID,
		ExecutedAt: executedAt, IdempotencyKey: "sched-1",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Value: "ok", Pass: true},
			{ItemCode: "refrigerant", NumericValue: ptrFloat64(0.5), Pass: true},
		},
	}); err != nil {
		t.Fatalf("submit execution: %v", err)
	}
	// Change the cycle; next due must use the new cycle from the last execution.
	if _, err := planSvc.ChangeCycle(context.Background(), adminActor(), plan.ID, 45, "调整为45天"); err != nil {
		t.Fatalf("change cycle: %v", err)
	}
	updated, err := ports.Plans.GetPlan(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	if updated.CurrentCycleDays != 45 {
		t.Fatalf("live cycle = %d, want 45", updated.CurrentCycleDays)
	}
	wantNext := executedAt.AddDate(0, 0, 45)
	if !updated.NextDueDate.Equal(wantNext) {
		t.Errorf("next due = %v, want %v", updated.NextDueDate, wantNext)
	}
}
