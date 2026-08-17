package application

import (
	"context"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/domain"
	"github.com/cry047/baseline/internal/repository/memory"
	"github.com/cry047/baseline/internal/service/notifier"
	"github.com/cry047/baseline/internal/service/storage"
	"github.com/google/uuid"
)

// setupTestPorts returns a fully-wired *domain.Ports backed by in-memory
// repositories and offline adapters. A deterministic clock is used so tests
// can control time.
func setupTestPorts(t *testing.T, now time.Time) (*domain.Ports, *domain.StubClock) {
	t.Helper()
	store := memory.NewStore()
	ports := memory.NewPorts(store)
	clock := &domain.StubClock{T: now}
	ports.Clock = clock
	notifierAdapter := notifier.New(128, "")
	ports.Notifier = notifierAdapter
	st, err := storage.New(storage.Config{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("init storage: %v", err)
	}
	ports.Storage = st
	return &ports, clock
}

// setupPlaceAndPerson creates a place and a responsible person, returning
// their ids.
func setupPlaceAndPerson(t *testing.T, ports *domain.Ports) (string, string) {
	t.Helper()
	ctx := context.Background()
	pid := uuid.NewString()
	if err := ports.Places.Create(ctx, &domain.Place{
		ID: pid, Name: "图书馆", Type: domain.PlaceTypeLibrary, Address: "市中心", Code: "LIB-001",
	}); err != nil {
		t.Fatalf("create place: %v", err)
	}
	rpid := uuid.NewString()
	if err := ports.People.Create(ctx, &domain.ResponsiblePerson{
		ID: rpid, Name: "张伟", Email: "z@x.com", Phone: "13800138000", Active: true,
	}); err != nil {
		t.Fatalf("create person: %v", err)
	}
	return pid, rpid
}

// setupFacility creates a facility with the given criticality and status.
func setupFacility(t *testing.T, ports *domain.Ports, placeID, personID string, crit domain.Criticality, status domain.FacilityStatus) *domain.Facility {
	t.Helper()
	ctx := context.Background()
	f := &domain.Facility{
		ID:                  uuid.NewString(),
		PlaceID:             placeID,
		Name:                "空调",
		Code:                "AC-" + uuid.NewString()[:6],
		Category:            "空调",
		ResponsiblePersonID: personID,
		Criticality:         crit,
		Status:              status,
	}
	if err := ports.Facilities.Create(ctx, f); err != nil {
		t.Fatalf("create facility: %v", err)
	}
	return f
}

// setupTemplate creates a plan template.
func setupTemplate(t *testing.T, ports *domain.Ports) *domain.PlanTemplate {
	t.Helper()
	ctx := context.Background()
	minV := 0.4
	maxV := 0.6
	tpl := &domain.PlanTemplate{
		ID:        uuid.NewString(),
		Name:      "空调月度保养",
		CycleDays: 30,
		InspectionItems: []domain.InspectionItem{
			{Code: "filter", Name: "滤网", Required: true},
			{Code: "refrigerant", Name: "制冷剂压力", Required: true, MinValue: &minV, MaxValue: &maxV},
		},
		Consumables: []domain.ConsumableSpec{
			{Code: "filter-pad", Name: "滤网", Quantity: 1, Unit: "件"},
		},
	}
	if err := ports.Templates.CreateTemplate(ctx, tpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	return tpl
}

// setupPlan creates a maintenance plan attached to a facility.
func setupPlan(t *testing.T, ports *domain.Ports, facilityID, templateID string, nextDue time.Time) *domain.MaintenancePlan {
	t.Helper()
	ctx := context.Background()
	tpl, err := ports.Templates.GetTemplate(ctx, templateID)
	if err != nil {
		t.Fatalf("get template: %v", err)
	}
	p := &domain.MaintenancePlan{
		ID:               uuid.NewString(),
		FacilityID:       facilityID,
		TemplateID:       templateID,
		CurrentCycleDays: tpl.CycleDays,
		NextDueDate:      nextDue,
		Active:           true,
	}
	if err := ports.Plans.CreatePlan(ctx, p); err != nil {
		t.Fatalf("create plan: %v", err)
	}
	return p
}

func adminActor() domain.Actor {
	return domain.Actor{ID: "admin-1", Name: "管理员", Role: domain.RoleAdmin}
}

func supervisorActor() domain.Actor {
	return domain.Actor{ID: "sup-1", Name: "主管", Role: domain.RoleSupervisor}
}

func operatorActor() domain.Actor {
	return domain.Actor{ID: "op-1", Name: "操作员", Role: domain.RoleOperator}
}

func viewerActor() domain.Actor {
	return domain.Actor{ID: "viewer-1", Name: "浏览者", Role: domain.RoleViewer}
}
