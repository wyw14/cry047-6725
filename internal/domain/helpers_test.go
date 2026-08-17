package domain

import (
	"testing"
	"time"
)

// mustParseTime parses a RFC3339 time string and fails the test if invalid.
func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// TestRole_Permissions verifies the RBAC helpers used by application services.
func TestRole_Permissions(t *testing.T) {
	if !RoleAdmin.CanManagePlans() {
		t.Errorf("admin should manage plans")
	}
	if !RoleEngineer.CanManagePlans() {
		t.Errorf("engineer should manage plans")
	}
	if RoleOperator.CanManagePlans() {
		t.Errorf("operator should not manage plans")
	}
	if RoleViewer.CanManagePlans() {
		t.Errorf("viewer should not manage plans")
	}
	if !RoleOperator.CanExecute() {
		t.Errorf("operator should execute")
	}
	if RoleViewer.CanExecute() {
		t.Errorf("viewer should not execute")
	}
	if !RoleEngineer.CanRectify() {
		t.Errorf("engineer should rectify")
	}
	if !RoleSupervisor.CanRecover() {
		t.Errorf("supervisor should recover")
	}
	if !RoleAdmin.CanRecover() {
		t.Errorf("admin should recover")
	}
	if RoleOperator.CanRecover() {
		t.Errorf("operator should not recover")
	}
	if RoleViewer.CanRecover() {
		t.Errorf("viewer should not recover")
	}
	if !RoleAdmin.CanAudit() {
		t.Errorf("admin should audit")
	}
	if RoleOperator.CanAudit() {
		t.Errorf("operator should not audit")
	}
}

func TestPageQuery_Normalize(t *testing.T) {
	q := PageQuery{Limit: 0, Offset: -5, Order: "", Filters: nil}
	q.Normalize()
	if q.Limit != 20 {
		t.Errorf("expected limit 20, got %d", q.Limit)
	}
	if q.Offset != 0 {
		t.Errorf("expected offset 0, got %d", q.Offset)
	}
	if q.Order != "asc" {
		t.Errorf("expected order asc, got %s", q.Order)
	}
	if q.Filters == nil {
		t.Errorf("expected filters map non-nil")
	}
	q2 := PageQuery{Limit: 500}
	q2.Normalize()
	if q2.Limit != 200 {
		t.Errorf("expected limit clamped to 200, got %d", q2.Limit)
	}
}
