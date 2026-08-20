package application

import (
	"context"

	"github.com/cry047/baseline/internal/domain"
)

// ExecutionService orchestrates maintenance execution records.
type ExecutionService struct {
	baseService
}

// NewExecutionService returns an ExecutionService.
func NewExecutionService(ports *domain.Ports, opts ...Option) *ExecutionService {
	return &ExecutionService{baseService: newBase(ports, opts)}
}

// Submit creates a maintenance execution record. Idempotency is enforced via
// IdempotencyKey — the same key returns the previously-created record.
func (s *ExecutionService) Submit(ctx context.Context, actor domain.Actor, in domain.ExecutionInput) (*domain.Execution, error) {
	if !actor.Role.CanExecute() {
		return nil, domain.ErrForbidden("当前角色无权限提交保养执行记录")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	// Idempotency: return the existing record if the key is already used.
	if existing, err := s.ports.Executions.GetByIdempotencyKey(ctx, in.IdempotencyKey); err == nil {
		return existing, nil
	}
	plan, err := s.ports.Plans.GetPlan(ctx, in.PlanID)
	if err != nil {
		return nil, err
	}
	if plan.FacilityID != in.FacilityID {
		return nil, domain.ErrInvalid("计划与设施不匹配", "facility_id", nil)
	}
	f, err := s.ports.Facilities.Get(ctx, in.FacilityID)
	if err != nil {
		return nil, err
	}
	// Facility must not be in under_repair / restricted_use to record a
	// preventive maintenance execution (use anomaly flow for those).
	if f.Status == domain.FacilityUnderRepair || f.Status == domain.FacilityRestrictedUse {
		return nil, domain.ErrStateForbidden("设施处于维修/限用状态，不能提交预防性保养记录")
	}
	tpl, err := s.ports.Templates.GetTemplate(ctx, plan.TemplateID)
	if err != nil {
		return nil, err
	}
	if err := validateInspectionValues(tpl.InspectionItems, in.InspectionValues); err != nil {
		return nil, err
	}
	if err := validateConsumables(tpl.Consumables, in.ConsumablesConsumed); err != nil {
		return nil, err
	}
	// Build the execution record from the caller's input, copying the
	// evidence slices into freshly allocated backing arrays. The request data
	// (in) is caller-owned and may be reused or mutated after Submit returns;
	// without this deep copy the persisted record and the returned object would
	// alias the same slice memory, so later edits to in would silently mutate
	// the saved evidence. Cloning each element (rather than slicing in[:len])
	// also detaches the NumericValue *float64 pointers.
	e := &domain.Execution{
		ID:                  s.id(),
		PlanID:              in.PlanID,
		FacilityID:          in.FacilityID,
		ExecutedBy:          in.ExecutedBy,
		ExecutedAt:          in.ExecutedAt,
		InspectionValues:    domain.CloneInspectionValues(in.InspectionValues),
		Photos:              domain.ClonePhotoAttachments(in.Photos),
		ConsumablesConsumed: domain.CloneConsumablesConsumed(in.ConsumablesConsumed),
		Status:              domain.ExecutionSubmitted,
		IdempotencyKey:      in.IdempotencyKey,
	}
	if err := s.ports.Executions.Create(ctx, e); err != nil {
		// Race: another concurrent request with the same idempotency_key won.
		// Return the existing record instead of an error.
		if existing, lookupErr := s.ports.Executions.GetByIdempotencyKey(ctx, in.IdempotencyKey); lookupErr == nil {
			return existing, nil
		}
		return nil, err
	}
	// Update plan: last execution, next due date derived from cycle.
	updated := *plan
	updated.LastExecutedAt = in.ExecutedAt
	updated.LastExecutionID = e.ID
	updated.NextDueDate = in.ExecutedAt.AddDate(0, 0, plan.CurrentCycleDays)
	updated.Version = plan.Version + 1
	if err := s.ports.Plans.UpdatePlan(ctx, &updated); err != nil {
		// If the plan update fails due to a race (another concurrent submit
		// incremented version first), reload and retry.
		latest, reloadErr := s.ports.Plans.GetPlan(ctx, in.PlanID)
		if reloadErr != nil {
			return nil, err
		}
		latest.LastExecutedAt = in.ExecutedAt
		latest.LastExecutionID = e.ID
		latest.NextDueDate = in.ExecutedAt.AddDate(0, 0, latest.CurrentCycleDays)
		latest.Version = latest.Version + 1
		_ = s.ports.Plans.UpdatePlan(ctx, latest)
	}
	// Mark facility as Normal after successful maintenance, unless an anomaly
	// is currently open (which would keep it in restricted_use / under_repair).
	if f.Status == domain.FacilityPendingMaintenance || f.Status == domain.FacilityOverdue {
		_ = s.ports.Facilities.UpdateStatus(ctx, f.ID, domain.FacilityNormal, f.Version)
	}
	_ = s.audit(ctx, domain.AuditCreate, "Execution", e.ID, actor, nil, e, "")
	_ = s.timeline(ctx, in.FacilityID, "execution_submitted", "提交保养执行记录", actor.Name, e)
	return e.Snapshot(), nil
}

// Review approves or rejects an execution.
func (s *ExecutionService) Review(ctx context.Context, actor domain.Actor, executionID, comment string, approved bool) (*domain.Execution, error) {
	if !actor.Role.CanRecover() {
		return nil, domain.ErrForbidden("当前角色无权限复核执行记录")
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Executions.Get(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if before.Status != domain.ExecutionSubmitted {
		return nil, domain.ErrStateForbidden("仅 submitted 状态可被复核")
	}
	updated := *before
	updated.ReviewComment = comment
	updated.ReviewedBy = actor.ID
	updated.ReviewedAt = s.now()
	if approved {
		updated.Status = domain.ExecutionReviewed
	} else {
		updated.Status = domain.ExecutionRejected
	}
	updated.Version = before.Version + 1
	if err := s.ports.Executions.Update(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditUpdate, "Execution", executionID, actor, before, &updated, "review: "+comment)
	_ = s.timeline(ctx, before.FacilityID, "execution_reviewed", "保养记录复核", actor.Name, map[string]any{
		"approved": approved,
		"comment":  comment,
	})
	// Return a deep snapshot so the caller receives an independent copy that
	// cannot mutate, or be mutated by, the persisted record.
	return (&updated).Snapshot(), nil
}

// Get retrieves an execution record.
func (s *ExecutionService) Get(ctx context.Context, id string) (*domain.Execution, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Executions.Get(ctx, id)
}

// List returns paginated executions.
func (s *ExecutionService) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Execution], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Executions.List(ctx, q)
}

// ListByFacility returns the latest executions for a facility.
func (s *ExecutionService) ListByFacility(ctx context.Context, facilityID string, limit int) ([]*domain.Execution, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Executions.ListByFacility(ctx, facilityID, limit)
}

// validateInspectionValues enforces required-item presence and numeric ranges
// declared in the template.
func validateInspectionValues(templateItems []domain.InspectionItem, values []domain.InspectionValue) error {
	byCode := map[string]domain.InspectionItem{}
	for _, it := range templateItems {
		byCode[it.Code] = it
	}
	for _, v := range values {
		it, ok := byCode[v.ItemCode]
		if !ok {
			return domain.ErrInvalid("检查值 item_code 不在模板内: "+v.ItemCode, "inspection_values", nil)
		}
		if it.MinValue != nil && v.NumericValue != nil && *v.NumericValue < *it.MinValue {
			return domain.ErrInvalid("检查值小于最小值: "+v.ItemCode, "inspection_values", nil)
		}
		if it.MaxValue != nil && v.NumericValue != nil && *v.NumericValue > *it.MaxValue {
			return domain.ErrInvalid("检查值大于最大值: "+v.ItemCode, "inspection_values", nil)
		}
	}
	for _, it := range templateItems {
		if it.Required {
			found := false
			for _, v := range values {
				if v.ItemCode == it.Code {
					found = true
					break
				}
			}
			if !found {
				return domain.ErrInvalid("必填检查项缺失: "+it.Code, "inspection_values", nil)
			}
		}
	}
	return nil
}

// validateConsumables enforces that declared template consumables cannot be
// consumed above their declared quantity (a soft check) and unknown consumables
// are rejected.
func validateConsumables(templateSpecs []domain.ConsumableSpec, consumed []domain.ConsumableConsumption) error {
	byCode := map[string]domain.ConsumableSpec{}
	for _, c := range templateSpecs {
		byCode[c.Code] = c
	}
	for _, c := range consumed {
		spec, ok := byCode[c.Code]
		if !ok {
			return domain.ErrInvalid("耗材 code 不在模板内: "+c.Code, "consumables_consumed", nil)
		}
		if c.Quantity < 0 {
			return domain.ErrInvalid("耗材数量不能为负: "+c.Code, "consumables_consumed", nil)
		}
		if c.Quantity > spec.Quantity*10 {
			return domain.ErrInvalid("耗材数量超过模板上限的 10 倍: "+c.Code, "consumables_consumed", nil)
		}
	}
	return nil
}
