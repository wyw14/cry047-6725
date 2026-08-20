package domain

import (
	"context"
	"time"
)

// PlanTemplate describes a reusable maintenance checklist.
type PlanTemplate struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	CycleDays        int              `json:"cycle_days"`
	InspectionItems  []InspectionItem `json:"inspection_items"`
	Consumables      []ConsumableSpec `json:"consumables"`
	RequiresShutdown bool             `json:"requires_shutdown"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	Version          int              `json:"version"`
}

// InspectionItem is a single checklist item inside a template.
type InspectionItem struct {
	Code     string   `json:"code"`
	Name     string   `json:"name"`
	Required bool     `json:"required"`
	Unit     string   `json:"unit,omitempty"`
	MinValue *float64 `json:"min_value,omitempty"`
	MaxValue *float64 `json:"max_value,omitempty"`
}

// ConsumableSpec declares a consumable that should be consumed during execution.
type ConsumableSpec struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Unit     string `json:"unit,omitempty"`
}

// PlanTemplateInput is the validated input for creating or updating a template.
type PlanTemplateInput struct {
	Name             string           `json:"name"`
	CycleDays        int              `json:"cycle_days"`
	InspectionItems  []InspectionItem `json:"inspection_items"`
	Consumables      []ConsumableSpec `json:"consumables"`
	RequiresShutdown bool             `json:"requires_shutdown"`
}

// Validate enforces template field-level invariants.
func (i PlanTemplateInput) Validate() error {
	if len(i.Name) < 2 || len(i.Name) > 128 {
		return ErrInvalid("模板名称长度需在 2-128 之间", "name", nil)
	}
	if i.CycleDays < 1 || i.CycleDays > 365 {
		return ErrInvalid("周期天数需在 1-365 之间", "cycle_days", nil)
	}
	codes := map[string]bool{}
	for idx, item := range i.InspectionItems {
		if item.Code == "" {
			return ErrInvalid("检查项 code 不能为空", "inspection_items["+itoa(idx)+"].code", nil)
		}
		if codes[item.Code] {
			return ErrInvalid("检查项 code 重复: "+item.Code, "inspection_items["+itoa(idx)+"].code", nil)
		}
		codes[item.Code] = true
		if item.Name == "" {
			return ErrInvalid("检查项 name 不能为空", "inspection_items["+itoa(idx)+"].name", nil)
		}
		if item.MinValue != nil && item.MaxValue != nil && *item.MinValue > *item.MaxValue {
			return ErrInvalid("检查项最小值不能大于最大值", "inspection_items["+itoa(idx)+"].min_value", nil)
		}
	}
	for idx, c := range i.Consumables {
		if c.Code == "" {
			return ErrInvalid("耗材 code 不能为空", "consumables["+itoa(idx)+"].code", nil)
		}
		if c.Name == "" {
			return ErrInvalid("耗材 name 不能为空", "consumables["+itoa(idx)+"].name", nil)
		}
		if c.Quantity < 1 {
			return ErrInvalid("耗材数量需大于 0", "consumables["+itoa(idx)+"].quantity", nil)
		}
	}
	return nil
}

// MaintenancePlan is the live maintenance plan attached to a facility.
type MaintenancePlan struct {
	ID               string    `json:"id"`
	FacilityID       string    `json:"facility_id"`
	TemplateID       string    `json:"template_id"`
	CurrentCycleDays int       `json:"current_cycle_days"`
	NextDueDate      time.Time `json:"next_due_date"`
	LastExecutedAt   time.Time `json:"last_executed_at,omitempty"`
	LastExecutionID  string    `json:"last_execution_id,omitempty"`
	Active           bool      `json:"active"`
	SkippedCount     int       `json:"skipped_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Version          int       `json:"version"`
}

// OccurrenceDate identifies the single maintenance occurrence a plan currently
// represents: the due date of its current cycle, expressed as a calendar day.
//
// Each cycle advances NextDueDate to a new day (on execution, skip or
// regeneration), so a plan that falls overdue again on a later cycle yields a
// different OccurrenceDate — and therefore a distinct auto-generated todo
// (see MaintenanceTodoKey). Two scheduler scans that observe the same
// NextDueDate observe the same occurrence, which is what keeps repeated
// scanning idempotent.
//
// This previously returned CreatedAt as a "stable occurrence anchor", which
// collapsed every overdue occurrence of a plan onto a single todo — only the
// first overdue ever produced a reminder. Keying by NextDueDate instead gives
// each overdue occurrence its own todo while still making re-scans safe.
func (p MaintenancePlan) OccurrenceDate() time.Time {
	if !p.NextDueDate.IsZero() {
		return p.NextDueDate
	}
	// Defensive: a plan with no due date yet keys off its creation day so the
	// derived idempotency key is still stable.
	return p.CreatedAt
}

// PlanVersion captures the historical cycle and template changes of a plan.
type PlanVersion struct {
	ID            string    `json:"id"`
	PlanID        string    `json:"plan_id"`
	VersionNumber int       `json:"version_number"`
	CycleDays     int       `json:"cycle_days"`
	TemplateID    string    `json:"template_id,omitempty"`
	ChangedBy     string    `json:"changed_by"`
	ChangedAt     time.Time `json:"changed_at"`
	Reason        string    `json:"reason"`
}

// MaintenancePlanInput is the validated input for creating a plan.
type MaintenancePlanInput struct {
	FacilityID string    `json:"facility_id"`
	TemplateID string    `json:"template_id"`
	StartAt    time.Time `json:"start_at"`
}

// Validate enforces plan input invariants.
func (i MaintenancePlanInput) Validate() error {
	if i.FacilityID == "" {
		return ErrInvalid("设施 ID 不能为空", "facility_id", nil)
	}
	if i.TemplateID == "" {
		return ErrInvalid("模板 ID 不能为空", "template_id", nil)
	}
	return nil
}

// PlanRepository is the persistence contract for templates and plans.
type PlanRepository interface {
	CreateTemplate(ctx context.Context, t *PlanTemplate) error
	UpdateTemplate(ctx context.Context, t *PlanTemplate) error
	GetTemplate(ctx context.Context, id string) (*PlanTemplate, error)
	ListTemplates(ctx context.Context, q PageQuery) (*PageResult[*PlanTemplate], error)

	CreatePlan(ctx context.Context, p *MaintenancePlan) error
	UpdatePlan(ctx context.Context, p *MaintenancePlan) error
	GetPlan(ctx context.Context, id string) (*MaintenancePlan, error)
	GetPlanByFacility(ctx context.Context, facilityID string) (*MaintenancePlan, error)
	ListPlans(ctx context.Context, q PageQuery) (*PageResult[*MaintenancePlan], error)
	ListOverduePlans(ctx context.Context, before time.Time) ([]*MaintenancePlan, error)
	ListDuePlans(ctx context.Context, before time.Time) ([]*MaintenancePlan, error)

	AppendPlanVersion(ctx context.Context, v *PlanVersion) error
	ListPlanVersions(ctx context.Context, planID string) ([]*PlanVersion, error)
}

// itoa is a tiny zero-alloc int-to-string helper to avoid importing strconv
// in the hot path.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
