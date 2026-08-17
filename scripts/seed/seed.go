package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/cry047/baseline/internal/domain"
	"github.com/google/uuid"
)

// SeedResult summarizes the seed run.
type SeedResult struct {
	Places     int
	People     int
	Facilities int
	Templates  int
	Plans      int
	Skipped    int
}

// Run inserts demo data into the given Ports if it does not already exist.
// Demo data is idempotent: re-running seed will NOT overwrite existing rows.
func Run(ctx context.Context, ports *domain.Ports, actor domain.Actor) (*SeedResult, error) {
	res := &SeedResult{}
	now := time.Now().UTC()

	// 1. Places.
	places := []domain.Place{
		{ID: "place-lib-001", Name: "市中心图书馆", Type: domain.PlaceTypeLibrary, Address: "市中心广场 1 号", Code: "LIB-001"},
		{ID: "place-mus-001", Name: "城市博物馆", Type: domain.PlaceTypeMuseum, Address: "文化路 88 号", Code: "MUS-001"},
		{ID: "place-exh-001", Name: "国际展览中心", Type: domain.PlaceTypeExhibit, Address: "会展大道 100 号", Code: "EXH-001"},
	}
	for _, p := range places {
		p.CreatedAt = now
		p.UpdatedAt = now
		p.Version = 1
		if existing, err := ports.Places.Get(ctx, p.ID); err == nil && existing != nil {
			res.Skipped++
			continue
		}
		if err := ports.Places.Create(ctx, &p); err != nil {
			return res, fmt.Errorf("seed place %s: %w", p.ID, err)
		}
		res.Places++
	}

	// 2. Responsible persons.
	people := []domain.ResponsiblePerson{
		{ID: "rp-001", Name: "张伟", Email: "zhangwei@example.com", Phone: "13800138001", Department: "运维一部", Active: true},
		{ID: "rp-002", Name: "李娜", Email: "lina@example.com", Phone: "13800138002", Department: "运维二部", Active: true},
		{ID: "rp-003", Name: "王强", Email: "wangqiang@example.com", Phone: "13800138003", Department: "运维一部", Active: true},
	}
	for _, p := range people {
		p.CreatedAt = now
		p.UpdatedAt = now
		p.Version = 1
		if existing, err := ports.People.Get(ctx, p.ID); err == nil && existing != nil {
			res.Skipped++
			continue
		}
		if err := ports.People.Create(ctx, &p); err != nil {
			return res, fmt.Errorf("seed person %s: %w", p.ID, err)
		}
		res.People++
	}

	// 3. Facilities.
	facilities := []domain.Facility{
		{ID: "fac-hvac-001", PlaceID: "place-lib-001", Name: "主空调机组", Code: "HVAC-001", Category: "空调", ResponsiblePersonID: "rp-001", Criticality: domain.CriticalityCritical, Status: domain.FacilityNormal, Description: "图书馆主空调机组"},
		{ID: "fac-elev-001", PlaceID: "place-lib-001", Name: "1 号电梯", Code: "ELEV-001", Category: "电梯", ResponsiblePersonID: "rp-002", Criticality: domain.CriticalityImportant, Status: domain.FacilityNormal, Description: "图书馆 1 号客梯"},
		{ID: "fac-fire-001", PlaceID: "place-mus-001", Name: "消防控制柜", Code: "FIRE-001", Category: "消防", ResponsiblePersonID: "rp-003", Criticality: domain.CriticalityCritical, Status: domain.FacilityNormal, Description: "博物馆消防控制柜"},
		{ID: "fac-lamp-001", PlaceID: "place-exh-001", Name: "展厅照明系统", Code: "LAMP-001", Category: "照明", ResponsiblePersonID: "rp-001", Criticality: domain.CriticalityStandard, Status: domain.FacilityNormal, Description: "国际展览中心主照明"},
	}
	for _, f := range facilities {
		f.CreatedAt = now
		f.UpdatedAt = now
		f.Version = 1
		if existing, err := ports.Facilities.Get(ctx, f.ID); err == nil && existing != nil {
			res.Skipped++
			continue
		}
		if err := ports.Facilities.Create(ctx, &f); err != nil {
			return res, fmt.Errorf("seed facility %s: %w", f.ID, err)
		}
		res.Facilities++
	}

	// 4. Plan templates.
	templates := []domain.PlanTemplate{
		{
			ID: "tpl-hvac-30", Name: "空调月度保养", CycleDays: 30, RequiresShutdown: false,
			InspectionItems: []domain.InspectionItem{
				{Code: "filter", Name: "滤网清洁度", Required: true},
				{Code: "refrigerant", Name: "制冷剂压力", Required: true, MinValue: ptrFloat(0.4), MaxValue: ptrFloat(0.6)},
				{Code: "noise", Name: "运行噪音", Required: false},
			},
			Consumables: []domain.ConsumableSpec{
				{Code: "filter-pad", Name: "滤网", Quantity: 1, Unit: "件"},
			},
		},
		{
			ID: "tpl-elev-90", Name: "电梯季度保养", CycleDays: 90, RequiresShutdown: true,
			InspectionItems: []domain.InspectionItem{
				{Code: "rope", Name: "钢丝绳磨损", Required: true},
				{Code: "brake", Name: "制动器", Required: true},
				{Code: "door", Name: "门系统", Required: false},
			},
			Consumables: []domain.ConsumableSpec{
				{Code: "lubricant", Name: "润滑油", Quantity: 1, Unit: "升"},
			},
		},
		{
			ID: "tpl-fire-7", Name: "消防周检", CycleDays: 7, RequiresShutdown: false,
			InspectionItems: []domain.InspectionItem{
				{Code: "indicator", Name: "指示灯", Required: true},
				{Code: "alarm", Name: "报警测试", Required: true},
			},
			Consumables: []domain.ConsumableSpec{},
		},
		{
			ID: "tpl-lamp-180", Name: "照明半年保养", CycleDays: 180, RequiresShutdown: false,
			InspectionItems: []domain.InspectionItem{
				{Code: "luminance", Name: "亮度", Required: true, MinValue: ptrFloat(300), MaxValue: ptrFloat(800)},
				{Code: "ballast", Name: "镇流器", Required: false},
			},
			Consumables: []domain.ConsumableSpec{
				{Code: "tube", Name: "灯管", Quantity: 4, Unit: "支"},
			},
		},
	}
	for _, t := range templates {
		t.CreatedAt = now
		t.UpdatedAt = now
		t.Version = 1
		if existing, err := ports.Templates.GetTemplate(ctx, t.ID); err == nil && existing != nil {
			res.Skipped++
			continue
		}
		if err := ports.Templates.CreateTemplate(ctx, &t); err != nil {
			return res, fmt.Errorf("seed template %s: %w", t.ID, err)
		}
		res.Templates++
	}

	// 5. Maintenance plans.
	planSpecs := []struct {
		ID, FacilityID, TemplateID string
		Start                      time.Time
	}{
		{"plan-001", "fac-hvac-001", "tpl-hvac-30", now.AddDate(0, 0, -32)},
		{"plan-002", "fac-elev-001", "tpl-elev-90", now.AddDate(0, 0, -10)},
		{"plan-003", "fac-fire-001", "tpl-fire-7", now.AddDate(0, 0, -9)},
		{"plan-004", "fac-lamp-001", "tpl-lamp-180", now.AddDate(0, 0, -30)},
	}
	for _, spec := range planSpecs {
		tpl, err := ports.Templates.GetTemplate(ctx, spec.TemplateID)
		if err != nil {
			return res, err
		}
		p := &domain.MaintenancePlan{
			ID:               spec.ID,
			FacilityID:       spec.FacilityID,
			TemplateID:       spec.TemplateID,
			CurrentCycleDays: tpl.CycleDays,
			NextDueDate:      spec.Start.AddDate(0, 0, tpl.CycleDays),
			Active:           true,
		}
		p.CreatedAt = now
		p.UpdatedAt = now
		p.Version = 1
		if existing, err := ports.Plans.GetPlan(ctx, p.ID); err == nil && existing != nil {
			res.Skipped++
			continue
		}
		if err := ports.Plans.CreatePlan(ctx, p); err != nil {
			return res, fmt.Errorf("seed plan %s: %w", p.ID, err)
		}
		pv := &domain.PlanVersion{
			ID:            uuid.NewString(),
			PlanID:        p.ID,
			VersionNumber: 1,
			CycleDays:     tpl.CycleDays,
			TemplateID:    tpl.ID,
			ChangedBy:     actor.ID,
			ChangedAt:     now,
			Reason:        "seed init",
		}
		_ = ports.Plans.AppendPlanVersion(ctx, pv)
		res.Plans++
	}

	return res, nil
}

// ptrFloat returns a pointer to a float64.
func ptrFloat(v float64) *float64 { return &v }
