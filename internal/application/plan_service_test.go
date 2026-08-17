package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TestChangeCycleOnlyAffectsFuturePlan enforces the prompt invariant:
// "更换周期只影响后续计划，不能篡改已完成记录".
func TestChangeCycleOnlyAffectsFuturePlan(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityCritical, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, 30))
	svc := NewPlanService(ports)

	// Submit an execution with the original cycle.
	execSvc := NewExecutionService(ports)
	executedAt := now.AddDate(0, 0, -1)
	exec1, err := execSvc.Submit(context.Background(), operatorActor(), domain.ExecutionInput{
		PlanID:         plan.ID,
		FacilityID:     f.ID,
		ExecutedBy:     operatorActor().ID,
		ExecutedAt:     executedAt,
		IdempotencyKey: "exec-1",
		InspectionValues: []domain.InspectionValue{
			{ItemCode: "filter", Value: "ok", Pass: true},
			{ItemCode: "refrigerant", NumericValue: ptrFloat64(0.5), Pass: true},
		},
	})
	if err != nil {
		t.Fatalf("submit execution: %v", err)
	}

	// Capture the execution record (it's immutable).
	originalExecTime := exec1.ExecutedAt
	originalExecStatus := exec1.Status

	// Change the cycle from 30 to 60.
	if _, err := svc.ChangeCycle(context.Background(), adminActor(), plan.ID, 60, "扩展周期"); err != nil {
		t.Fatalf("change cycle: %v", err)
	}

	// Verify the existing execution record was NOT modified.
	stored, err := ports.Executions.Get(context.Background(), exec1.ID)
	if err != nil {
		t.Fatalf("get execution: %v", err)
	}
	if !stored.ExecutedAt.Equal(originalExecTime) {
		t.Errorf("execution record ExecutedAt was mutated: was %v, now %v", originalExecTime, stored.ExecutedAt)
	}
	if stored.Status != originalExecStatus {
		t.Errorf("execution record Status was mutated: was %s, now %s", originalExecStatus, stored.Status)
	}

	// Verify the plan's next_due_date was recomputed based on the new cycle
	// (60 days) and the previous execution time.
	updated, err := ports.Plans.GetPlan(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	if updated.CurrentCycleDays != 60 {
		t.Errorf("expected cycle 60, got %d", updated.CurrentCycleDays)
	}
	wantNext := executedAt.AddDate(0, 0, 60)
	if !updated.NextDueDate.Equal(wantNext) {
		t.Errorf("expected next due %v, got %v", wantNext, updated.NextDueDate)
	}
}

// TestPlanService_Skip_Plan verifies that skipping advances next due date by
// exactly one cycle and records skipped_count.
func TestPlanService_Skip_Plan(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, 30))
	svc := NewPlanService(ports)
	originalNext := plan.NextDueDate
	updated, err := svc.Skip(context.Background(), adminActor(), plan.ID, "应急跳过")
	if err != nil {
		t.Fatalf("skip: %v", err)
	}
	if updated.SkippedCount != 1 {
		t.Errorf("expected skipped_count 1, got %d", updated.SkippedCount)
	}
	wantNext := originalNext.AddDate(0, 0, plan.CurrentCycleDays)
	if !updated.NextDueDate.Equal(wantNext) {
		t.Errorf("expected next %v, got %v", wantNext, updated.NextDueDate)
	}
}

// TestPlanService_ViewerCannotChangeCycle enforces the RBAC boundary.
func TestPlanService_ViewerCannotChangeCycle(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, 30))
	svc := NewPlanService(ports)
	_, err := svc.ChangeCycle(context.Background(), viewerActor(), plan.ID, 60, "")
	if err == nil {
		t.Fatalf("expected forbidden for viewer, got nil")
	}
}

// TestPlanService_AssignResponsible verifies the assignment flow.
func TestPlanService_AssignResponsible(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, 30))
	svc := NewPlanService(ports)
	// Create a second person.
	rp2ID := "rp-2"
	if err := ports.People.Create(context.Background(), &domain.ResponsiblePerson{
		ID: rp2ID, Name: "李娜", Email: "l@x.com", Phone: "13900000002", Active: true,
	}); err != nil {
		t.Fatalf("create person: %v", err)
	}
	if _, err := svc.AssignResponsible(context.Background(), adminActor(), plan.ID, rp2ID, "工作调整"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	f2, err := ports.Facilities.Get(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("get facility: %v", err)
	}
	if f2.ResponsiblePersonID != rp2ID {
		t.Errorf("expected %s, got %s", rp2ID, f2.ResponsiblePersonID)
	}
}

// TestPlanService_Regenerate verifies regeneration advances next due date.
func TestPlanService_Regenerate(t *testing.T) {
	now := time.Now().UTC()
	ports, _ := setupTestPorts(t, now)
	pid, rpid := setupPlaceAndPerson(t, ports)
	f := setupFacility(t, ports, pid, rpid, domain.CriticalityStandard, domain.FacilityNormal)
	tpl := setupTemplate(t, ports)
	plan := setupPlan(t, ports, f.ID, tpl.ID, now.AddDate(0, 0, -10)) // already overdue
	svc := NewPlanService(ports)
	updated, err := svc.Regenerate(context.Background(), adminActor(), plan.ID, "重新生成")
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if !updated.NextDueDate.After(now) {
		t.Errorf("expected next due after now, got %v", updated.NextDueDate)
	}
}

func ptrFloat64(v float64) *float64 { return &v }
