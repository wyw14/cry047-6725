package application

import (
	"context"

	"github.com/cry047/baseline/internal/domain"
)

// AnomalyService orchestrates anomaly lifecycle: discovery, rectification,
// reinspection and recovery.
type AnomalyService struct {
	baseService
}

// NewAnomalyService returns an AnomalyService.
func NewAnomalyService(ports *domain.Ports, opts ...Option) *AnomalyService {
	return &AnomalyService{baseService: newBase(ports, opts)}
}

// Discover creates a new anomaly. If the anomaly is severe (high/critical),
// the facility is transitioned into restricted_use automatically. For low/
// medium severities, the facility remains in its current state until
// rectification begins.
func (s *AnomalyService) Discover(ctx context.Context, actor domain.Actor, in domain.AnomalyInput) (*domain.Anomaly, error) {
	if !actor.Role.CanExecute() {
		return nil, domain.ErrForbidden("当前角色无权限登记异常")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	// Idempotency.
	if existing, err := s.ports.Anomalies.GetByIdempotencyKey(ctx, in.IdempotencyKey); err == nil {
		return existing, nil
	}
	f, err := s.ports.Facilities.Get(ctx, in.FacilityID)
	if err != nil {
		return nil, err
	}
	if in.ExecutionID != "" {
		if _, err := s.ports.Executions.Get(ctx, in.ExecutionID); err != nil {
			return nil, err
		}
	}
	a := &domain.Anomaly{
		ID:             s.id(),
		FacilityID:     in.FacilityID,
		ExecutionID:    in.ExecutionID,
		DiscoveredBy:   in.DiscoveredBy,
		DiscoveredAt:   in.DiscoveredAt,
		Description:    in.Description,
		Severity:       in.Severity,
		Status:         domain.AnomalyOpen,
		IdempotencyKey: in.IdempotencyKey,
	}
	if err := s.ports.Anomalies.Create(ctx, a); err != nil {
		return nil, err
	}
	// Auto-transition the facility to restricted_use for high/critical.
	if in.Severity == domain.AnomalySeverityHigh || in.Severity == domain.AnomalySeverityCritical {
		if f.Status == domain.FacilityNormal || f.Status == domain.FacilityPendingMaintenance || f.Status == domain.FacilityOverdue {
			_ = s.ports.Facilities.UpdateStatus(ctx, f.ID, domain.FacilityRestrictedUse, f.Version)
		}
	}
	_ = s.audit(ctx, domain.AuditCreate, "Anomaly", a.ID, actor, nil, a, "")
	_ = s.timeline(ctx, in.FacilityID, "anomaly_discovered", "异常发现", actor.Name, a)
	rp, _ := s.ports.People.Get(ctx, f.ResponsiblePersonID)
	if rp != nil {
		_ = s.notify(ctx, domain.NotificationInput{
			UserID: rp.ID,
			Type:   domain.NotificationAnomalyDiscovered,
			Title:  "新异常登记",
			Body:   "设施 " + f.Name + " 出现 " + string(in.Severity) + " 级异常：" + in.Description,
		})
	}
	return a, nil
}

// Rectify records the rectification measure and transitions the anomaly into
// reinspecting status. The associated facility transitions into under_repair
// (if it was restricted_use).
func (s *AnomalyService) Rectify(ctx context.Context, actor domain.Actor, in domain.RectificationInput) (*domain.Anomaly, error) {
	if !actor.Role.CanRectify() {
		return nil, domain.ErrForbidden("当前角色无权限执行整改")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Anomalies.Get(ctx, in.AnomalyID)
	if err != nil {
		return nil, err
	}
	if before.Status != domain.AnomalyOpen {
		return nil, domain.ErrStateForbidden("仅 open 状态可被整改")
	}
	f, err := s.ports.Facilities.Get(ctx, before.FacilityID)
	if err != nil {
		return nil, err
	}
	updated := *before
	updated.RectificationMeasure = in.Measure
	updated.RectifiedBy = in.RectifiedBy
	updated.RectifiedAt = s.now()
	updated.Status = domain.AnomalyReinspecting
	updated.Version = before.Version + 1
	if err := s.ports.Anomalies.Update(ctx, &updated); err != nil {
		return nil, err
	}
	if f.Status == domain.FacilityRestrictedUse {
		_ = s.ports.Facilities.UpdateStatus(ctx, f.ID, domain.FacilityUnderRepair, f.Version)
	}
	_ = s.audit(ctx, domain.AuditUpdate, "Anomaly", in.AnomalyID, actor, before, &updated, "rectification")
	_ = s.timeline(ctx, before.FacilityID, "anomaly_rectified", "异常整改完成", actor.Name, map[string]any{
		"measure": in.Measure,
	})
	return &updated, nil
}

// Reinspect records the reinspection result. If pass=true the anomaly moves
// to recovered status and awaits recovery confirmation; otherwise it returns
// to open and the facility remains under_repair.
func (s *AnomalyService) Reinspect(ctx context.Context, actor domain.Actor, in domain.ReinspectionInput, pass bool) (*domain.Anomaly, error) {
	if !actor.Role.CanRectify() {
		return nil, domain.ErrForbidden("当前角色无权限执行复检")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Anomalies.Get(ctx, in.AnomalyID)
	if err != nil {
		return nil, err
	}
	if before.Status != domain.AnomalyReinspecting {
		return nil, domain.ErrStateForbidden("仅 reinspecting 状态可被复检")
	}
	updated := *before
	updated.ReinspectionResult = in.Result
	updated.ReinspectedBy = in.ReinspectedBy
	updated.ReinspectedAt = in.ReinspectedAt
	updated.Status = domain.ReinspectionStatus(pass)
	updated.Version = before.Version + 1
	if err := s.ports.Anomalies.Update(ctx, &updated); err != nil {
		return nil, err
	}
	if facility, err := s.ports.Facilities.Get(ctx, before.FacilityID); err == nil && updated.Status == domain.AnomalyRecovered {
		_ = s.ports.Facilities.UpdateStatus(ctx, facility.ID, domain.FacilityRecovered, facility.Version)
	}
	_ = s.audit(ctx, domain.AuditUpdate, "Anomaly", in.AnomalyID, actor, before, &updated, "reinspection: "+in.Result)
	_ = s.timeline(ctx, before.FacilityID, "anomaly_reinspected", "异常复检", actor.Name, map[string]any{
		"pass":   pass,
		"result": in.Result,
	})
	return &updated, nil
}

// Recover confirms that the facility has been recovered. Only an anomaly in
// recovered status may be confirmed. The associated facility transitions to
// Recovered, awaiting a final transition back to Normal.
func (s *AnomalyService) Recover(ctx context.Context, actor domain.Actor, in domain.RecoverInput) (*domain.Anomaly, error) {
	if !actor.Role.CanRecover() {
		return nil, domain.ErrForbidden("当前角色无权限确认恢复")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Anomalies.Get(ctx, in.AnomalyID)
	if err != nil {
		return nil, err
	}
	if before.Status != domain.AnomalyRecovered {
		return nil, domain.ErrStateForbidden("仅 recovered 状态可被确认恢复")
	}
	f, err := s.ports.Facilities.Get(ctx, before.FacilityID)
	if err != nil {
		return nil, err
	}
	updated := *before
	updated.RecoveredBy = in.RecoveredBy
	updated.RecoveredAt = s.now()
	updated.Status = domain.AnomalyRecovered // keep as recovered for audit; same status
	updated.Version = before.Version + 1
	if err := s.ports.Anomalies.Update(ctx, &updated); err != nil {
		return nil, err
	}
	if f.Status == domain.FacilityUnderRepair || f.Status == domain.FacilityRestrictedUse {
		_ = s.ports.Facilities.UpdateStatus(ctx, f.ID, domain.FacilityRecovered, f.Version)
	}
	_ = s.audit(ctx, domain.AuditRecover, "Anomaly", in.AnomalyID, actor, before, &updated, "recovery confirmed")
	_ = s.timeline(ctx, before.FacilityID, "anomaly_recovered", "异常恢复确认", actor.Name, map[string]any{
		"recovered_by": in.RecoveredBy,
	})
	rp, _ := s.ports.People.Get(ctx, f.ResponsiblePersonID)
	if rp != nil {
		_ = s.notify(ctx, domain.NotificationInput{
			UserID: rp.ID,
			Type:   domain.NotificationAnomalyRecovered,
			Title:  "异常已恢复",
			Body:   "设施 " + f.Name + " 的异常已恢复，请确认设施状态。",
		})
	}
	return &updated, nil
}

// Get retrieves an anomaly.
func (s *AnomalyService) Get(ctx context.Context, id string) (*domain.Anomaly, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Anomalies.Get(ctx, id)
}

// List returns paginated anomalies.
func (s *AnomalyService) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Anomaly], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Anomalies.List(ctx, q)
}
