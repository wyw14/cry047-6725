package application

import (
	"context"

	"github.com/cry047/baseline/internal/domain"
)

// FacilityService orchestrates facility lifecycle, including the state machine
// that enforces the "overdue critical facilities cannot be marked normal"
// invariant.
type FacilityService struct {
	baseService
}

// NewFacilityService returns a FacilityService.
func NewFacilityService(ports *domain.Ports, opts ...Option) *FacilityService {
	return &FacilityService{baseService: newBase(ports, opts)}
}

// Create inserts a new facility.
func (s *FacilityService) Create(ctx context.Context, actor domain.Actor, in domain.FacilityInput) (*domain.Facility, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	// Cross-aggregate invariant: place must exist.
	if _, err := s.ports.Places.Get(ctx, in.PlaceID); err != nil {
		return nil, err
	}
	// Cross-aggregate invariant: responsible person must exist.
	if _, err := s.ports.People.Get(ctx, in.ResponsiblePersonID); err != nil {
		return nil, err
	}
	// Alternative facility must exist if specified.
	if in.AlternativeFacilityID != "" {
		if _, err := s.ports.Facilities.Get(ctx, in.AlternativeFacilityID); err != nil {
			return nil, err
		}
	}
	f := &domain.Facility{
		ID:                    s.id(),
		PlaceID:               in.PlaceID,
		Name:                  in.Name,
		Code:                  in.Code,
		Category:              in.Category,
		ResponsiblePersonID:   in.ResponsiblePersonID,
		Criticality:           in.Criticality,
		Status:                domain.FacilityNormal,
		Description:           in.Description,
		AlternativeFacilityID: in.AlternativeFacilityID,
	}
	if err := s.ports.Facilities.Create(ctx, f); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditCreate, "Facility", f.ID, actor, nil, f, "")
	_ = s.timeline(ctx, f.ID, "facility_created", "设施登记", actor.Name, f)
	return f, nil
}

// Update modifies a facility. Status is not changed via this method.
func (s *FacilityService) Update(ctx context.Context, actor domain.Actor, id string, in domain.FacilityInput) (*domain.Facility, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Facilities.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.ports.Places.Get(ctx, in.PlaceID); err != nil {
		return nil, err
	}
	if _, err := s.ports.People.Get(ctx, in.ResponsiblePersonID); err != nil {
		return nil, err
	}
	if in.AlternativeFacilityID != "" && in.AlternativeFacilityID != id {
		if _, err := s.ports.Facilities.Get(ctx, in.AlternativeFacilityID); err != nil {
			return nil, err
		}
	}
	updated := *before
	updated.PlaceID = in.PlaceID
	updated.Name = in.Name
	updated.Code = in.Code
	updated.Category = in.Category
	updated.ResponsiblePersonID = in.ResponsiblePersonID
	updated.Criticality = in.Criticality
	updated.Description = in.Description
	updated.AlternativeFacilityID = in.AlternativeFacilityID
	updated.Version = before.Version + 1
	if err := s.ports.Facilities.Update(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditUpdate, "Facility", id, actor, before, &updated, "")
	return &updated, nil
}

// TransitionState changes a facility's status while enforcing the state
// machine invariants.
func (s *FacilityService) TransitionState(ctx context.Context, actor domain.Actor, id string, to domain.FacilityStatus, reason string) (*domain.Facility, error) {
	if !actor.Role.CanExecute() && !actor.Role.CanRecover() {
		return nil, domain.ErrForbidden("当前角色无权限执行状态转换")
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Facilities.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	t := domain.FacilityStatusTransition{From: before.Status, To: to, Reason: reason}
	if err := t.ValidateTransition(before.Criticality); err != nil {
		return nil, err
	}
	updated := *before
	updated.Status = to
	updated.Version = before.Version + 1
	if err := s.ports.Facilities.Update(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditStateChange, "Facility", id, actor, before, &updated, reason)
	_ = s.timeline(ctx, id, "state_change", "状态变更: "+string(before.Status)+" -> "+string(to), actor.Name, map[string]any{
		"from":   before.Status,
		"to":     to,
		"reason": reason,
	})
	// Notify the responsible person when entering restricted/overdue states.
	rp, _ := s.ports.People.Get(ctx, before.ResponsiblePersonID)
	if rp != nil {
		switch to {
		case domain.FacilityOverdue:
			_ = s.notify(ctx, domain.NotificationInput{
				UserID: rp.ID,
				Type:   domain.NotificationMaintenanceOverdue,
				Title:  "设施保养逾期",
				Body:   "设施 " + before.Name + " 已逾期，请尽快安排保养。",
			})
		case domain.FacilityRestrictedUse:
			_ = s.notify(ctx, domain.NotificationInput{
				UserID: rp.ID,
				Type:   domain.NotificationAnomalyDiscovered,
				Title:  "设施限用通知",
				Body:   "设施 " + before.Name + " 已进入限用状态。",
			})
		case domain.FacilityRecovered:
			_ = s.notify(ctx, domain.NotificationInput{
				UserID: rp.ID,
				Type:   domain.NotificationAnomalyRecovered,
				Title:  "设施已恢复",
				Body:   "设施 " + before.Name + " 已完成恢复确认。",
			})
		}
	}
	return &updated, nil
}

// Get retrieves a facility.
func (s *FacilityService) Get(ctx context.Context, id string) (*domain.Facility, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Facilities.Get(ctx, id)
}

// List returns a paginated list of facilities.
func (s *FacilityService) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Facility], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Facilities.List(ctx, q)
}

// NextDateInfo computes the next maintenance date and overdue risk for a facility.
func (s *FacilityService) NextDateInfo(ctx context.Context, id string) (*domain.NextDateInfo, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	f, err := s.ports.Facilities.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	info := &domain.NextDateInfo{
		FacilityID:            id,
		AlternativeFacilityID: f.AlternativeFacilityID,
	}
	plan, err := s.ports.Plans.GetPlanByFacility(ctx, id)
	if err == nil {
		info.NextDueDate = plan.NextDueDate
		info.LastExecutedAt = plan.LastExecutedAt
		if !plan.NextDueDate.IsZero() {
			info.DaysUntilDue = int(plan.NextDueDate.Sub(s.now()).Hours() / 24)
			info.OverdueRisk = plan.NextDueDate.Before(s.now())
		}
	}
	return info, nil
}
