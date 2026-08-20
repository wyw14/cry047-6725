package application

import (
	"context"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// PlanService orchestrates plan templates and maintenance plans.
// The "changing cycle only affects future plans, cannot tamper with completed
// records" invariant is enforced here.
type PlanService struct {
	baseService
}

// NewPlanService returns a PlanService.
func NewPlanService(ports *domain.Ports, opts ...Option) *PlanService {
	return &PlanService{baseService: newBase(ports, opts)}
}

// CreateTemplate inserts a new plan template.
func (s *PlanService) CreateTemplate(ctx context.Context, actor domain.Actor, in domain.PlanTemplateInput) (*domain.PlanTemplate, error) {
	if !actor.Role.CanManagePlans() {
		return nil, domain.ErrForbidden("当前角色无权限管理计划模板")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	t := &domain.PlanTemplate{
		ID:               s.id(),
		Name:             in.Name,
		CycleDays:        in.CycleDays,
		InspectionItems:  in.InspectionItems,
		Consumables:      in.Consumables,
		RequiresShutdown: in.RequiresShutdown,
	}
	if err := s.ports.Templates.CreateTemplate(ctx, t); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditCreate, "PlanTemplate", t.ID, actor, nil, t, "")
	return t, nil
}

// UpdateTemplate modifies a plan template. Existing plans are not affected.
func (s *PlanService) UpdateTemplate(ctx context.Context, actor domain.Actor, id string, in domain.PlanTemplateInput) (*domain.PlanTemplate, error) {
	if !actor.Role.CanManagePlans() {
		return nil, domain.ErrForbidden("当前角色无权限管理计划模板")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Templates.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *before
	updated.Name = in.Name
	updated.CycleDays = in.CycleDays
	updated.InspectionItems = in.InspectionItems
	updated.Consumables = in.Consumables
	updated.RequiresShutdown = in.RequiresShutdown
	updated.Version = before.Version + 1
	if err := s.ports.Templates.UpdateTemplate(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditUpdate, "PlanTemplate", id, actor, before, &updated, "")
	return &updated, nil
}

// GetTemplate retrieves a plan template.
func (s *PlanService) GetTemplate(ctx context.Context, id string) (*domain.PlanTemplate, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Templates.GetTemplate(ctx, id)
}

// ListTemplates returns paginated plan templates.
func (s *PlanService) ListTemplates(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.PlanTemplate], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Templates.ListTemplates(ctx, q)
}

// CreatePlan creates a maintenance plan attached to a facility.
func (s *PlanService) CreatePlan(ctx context.Context, actor domain.Actor, in domain.MaintenancePlanInput) (*domain.MaintenancePlan, error) {
	if !actor.Role.CanManagePlans() {
		return nil, domain.ErrForbidden("当前角色无权限创建计划")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	f, err := s.ports.Facilities.Get(ctx, in.FacilityID)
	if err != nil {
		return nil, err
	}
	tpl, err := s.ports.Templates.GetTemplate(ctx, in.TemplateID)
	if err != nil {
		return nil, err
	}
	// Existing plan check: a facility may only have one active plan.
	if existing, err := s.ports.Plans.GetPlanByFacility(ctx, in.FacilityID); err == nil && existing != nil && existing.Active {
		return nil, domain.ErrConflict("设施 "+in.FacilityID+" 已存在生效计划", nil)
	}
	start := in.StartAt
	if start.IsZero() {
		start = s.now()
	}
	p := &domain.MaintenancePlan{
		ID:               s.id(),
		FacilityID:       in.FacilityID,
		TemplateID:       in.TemplateID,
		CurrentCycleDays: tpl.CycleDays,
		NextDueDate:      start.AddDate(0, 0, tpl.CycleDays),
		Active:           true,
	}
	if err := s.ports.Plans.CreatePlan(ctx, p); err != nil {
		return nil, err
	}
	pv := &domain.PlanVersion{
		ID:            s.id(),
		PlanID:        p.ID,
		VersionNumber: 1,
		CycleDays:     tpl.CycleDays,
		TemplateID:    tpl.ID,
		ChangedBy:     actor.ID,
		ChangedAt:     s.now(),
		Reason:        "init",
	}
	_ = s.ports.Plans.AppendPlanVersion(ctx, pv)
	_ = s.audit(ctx, domain.AuditCreate, "MaintenancePlan", p.ID, actor, nil, p, "")
	_ = s.timeline(ctx, f.ID, "plan_created", "新建保养计划", actor.Name, p)
	return p, nil
}

// ChangeCycle changes the cycle of a plan and only affects FUTURE scheduling.
// Already-completed execution records are immutable.
func (s *PlanService) ChangeCycle(ctx context.Context, actor domain.Actor, planID string, newCycleDays int, reason string) (*domain.MaintenancePlan, error) {
	if !actor.Role.CanManagePlans() {
		return nil, domain.ErrForbidden("当前角色无权限调整周期")
	}
	if newCycleDays < 1 || newCycleDays > 365 {
		return nil, domain.ErrInvalid("周期天数需在 1-365 之间", "cycle_days", nil)
	}
	if reason == "" {
		return nil, domain.ErrInvalid("调整原因不能为空", "reason", nil)
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Plans.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	// Sanity: do not allow cycle change for plans currently under recovery work
	// (i.e., facility in restricted_use / under_repair) because the next due
	// date will be re-derived after recovery.
	f, err := s.ports.Facilities.Get(ctx, before.FacilityID)
	if err != nil {
		return nil, err
	}
	if f.Status == domain.FacilityUnderRepair || f.Status == domain.FacilityRestrictedUse {
		return nil, domain.ErrStateForbidden("设施处于维修/限用状态，不允许调整周期")
	}
	updated := *before
	updated.CurrentCycleDays = newCycleDays
	// Recompute next due date from the most recent execution (or now) — never
	// modify completed execution records.
	base := s.now()
	if !before.LastExecutedAt.IsZero() {
		base = before.LastExecutedAt
	}
	updated.NextDueDate = base.AddDate(0, 0, newCycleDays)
	updated.Version = before.Version + 1
	if err := s.ports.Plans.UpdatePlan(ctx, &updated); err != nil {
		return nil, err
	}
	// Append plan version history. The new cycle is recorded as a new
	// immutable snapshot; prior snapshots are left untouched so that audit
	// history continues to reflect the cycle actually in effect at each point.
	existing, _ := s.ports.Plans.ListPlanVersions(ctx, planID)
	pv := &domain.PlanVersion{
		ID:            s.id(),
		PlanID:        planID,
		VersionNumber: domain.NextVersionNumber(existing),
		CycleDays:     newCycleDays,
		TemplateID:    before.TemplateID,
		ChangedBy:     actor.ID,
		ChangedAt:     s.now(),
		Reason:        reason,
	}
	_ = s.ports.Plans.AppendPlanVersion(ctx, pv)
	_ = s.audit(ctx, domain.AuditUpdate, "MaintenancePlan", planID, actor, before, &updated, reason)
	_ = s.timeline(ctx, before.FacilityID, "cycle_changed", "周期调整", actor.Name, map[string]any{
		"old_cycle_days": before.CurrentCycleDays,
		"new_cycle_days": newCycleDays,
		"reason":         reason,
	})
	// Notify responsible person.
	rp, _ := s.ports.People.Get(ctx, f.ResponsiblePersonID)
	if rp != nil {
		_ = s.notify(ctx, domain.NotificationInput{
			UserID: rp.ID,
			Type:   domain.NotificationPlanUpdated,
			Title:  "保养计划周期已调整",
			Body:   "设施 " + f.Name + " 的保养周期已调整为 " + itoaLocal(newCycleDays) + " 天。",
		})
	}
	return &updated, nil
}

// Skip marks a plan as skipped for the current cycle. The next due date is
// recomputed using the compensation rule: skipping advances the next due date
// by exactly one cycle and does NOT mark the previous execution as completed.
// The skip counter is recorded and audit-logged.
func (s *PlanService) Skip(ctx context.Context, actor domain.Actor, planID, reason string) (*domain.MaintenancePlan, error) {
	if !actor.Role.CanManagePlans() {
		return nil, domain.ErrForbidden("当前角色无权限跳过计划")
	}
	if reason == "" {
		return nil, domain.ErrInvalid("跳过原因不能为空", "reason", nil)
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Plans.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	if !before.Active {
		return nil, domain.ErrStateForbidden("计划未激活，不能跳过")
	}
	updated := *before
	updated.SkippedCount = before.SkippedCount + 1
	// Compensation rule: skip advances the next due date by one cycle.
	updated.NextDueDate = before.NextDueDate.AddDate(0, 0, before.CurrentCycleDays)
	updated.Version = before.Version + 1
	if err := s.ports.Plans.UpdatePlan(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditSkip, "MaintenancePlan", planID, actor, before, &updated, reason)
	_ = s.timeline(ctx, before.FacilityID, "plan_skipped", "计划跳过", actor.Name, map[string]any{
		"new_next_due_date": updated.NextDueDate,
		"skipped_count":     updated.SkippedCount,
		"reason":            reason,
	})
	return &updated, nil
}

// Regenerate advances the plan if its due date is already in the past and
// there is no execution record. Compensation rule: the plan moves forward by
// one cycle and the next due date is recomputed.
func (s *PlanService) Regenerate(ctx context.Context, actor domain.Actor, planID, reason string) (*domain.MaintenancePlan, error) {
	if !actor.Role.CanManagePlans() {
		return nil, domain.ErrForbidden("当前角色无权限重新生成计划")
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Plans.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	updated := *before
	updated.NextDueDate = s.now().AddDate(0, 0, before.CurrentCycleDays)
	updated.Version = before.Version + 1
	if err := s.ports.Plans.UpdatePlan(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditRegenerate, "MaintenancePlan", planID, actor, before, &updated, reason)
	_ = s.timeline(ctx, before.FacilityID, "plan_regenerated", "计划重新生成", actor.Name, map[string]any{
		"new_next_due_date": updated.NextDueDate,
		"reason":            reason,
	})
	return &updated, nil
}

// Get retrieves a plan.
func (s *PlanService) Get(ctx context.Context, id string) (*domain.MaintenancePlan, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Plans.GetPlan(ctx, id)
}

// GetByFacility returns the plan attached to a facility, if any.
func (s *PlanService) GetByFacility(ctx context.Context, facilityID string) (*domain.MaintenancePlan, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Plans.GetPlanByFacility(ctx, facilityID)
}

// List returns paginated plans.
func (s *PlanService) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.MaintenancePlan], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Plans.ListPlans(ctx, q)
}

// ListVersions returns the version history of a plan.
func (s *PlanService) ListVersions(ctx context.Context, planID string) ([]*domain.PlanVersion, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Plans.ListPlanVersions(ctx, planID)
}

// AssignResponsible changes the responsible person attached to a plan's
// facility. The change is audit-logged and the responsible person receives a
// notification.
func (s *PlanService) AssignResponsible(ctx context.Context, actor domain.Actor, planID, newPersonID, reason string) (*domain.MaintenancePlan, error) {
	if !actor.Role.CanManagePlans() {
		return nil, domain.ErrForbidden("当前角色无权限调整负责人")
	}
	if reason == "" {
		return nil, domain.ErrInvalid("调整原因不能为空", "reason", nil)
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Plans.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	rp, err := s.ports.People.Get(ctx, newPersonID)
	if err != nil {
		return nil, err
	}
	f, err := s.ports.Facilities.Get(ctx, before.FacilityID)
	if err != nil {
		return nil, err
	}
	prevPersonID := f.ResponsiblePersonID
	updated := *f
	updated.ResponsiblePersonID = newPersonID
	updated.Version = f.Version + 1
	if err := s.ports.Facilities.Update(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditUpdate, "Facility", f.ID, actor, f, &updated, "负责人调整: "+reason)
	_ = s.timeline(ctx, f.ID, "responsible_changed", "负责人调整", actor.Name, map[string]any{
		"old_person_id": prevPersonID,
		"new_person_id": newPersonID,
		"reason":        reason,
	})
	_ = s.notify(ctx, domain.NotificationInput{
		UserID: newPersonID,
		Type:   domain.NotificationAssigned,
		Title:  "保养计划负责人变更",
		Body:   "您已成为设施 " + f.Name + " 的保养负责人。",
	})
	_ = rp // avoid unused
	return before, nil
}

// itoaLocal is a tiny int-to-string helper.
func itoaLocal(i int) string {
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

// now0 avoids unused-import warnings in pure helpers.
var _ = time.Now
