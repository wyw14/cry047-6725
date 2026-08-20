package application

import (
	"context"

	"github.com/cry047/baseline/internal/domain"
)

// AuditService is the read-only access to audit logs.
type AuditService struct {
	baseService
}

// NewAuditService returns an AuditService.
func NewAuditService(ports *domain.Ports, opts ...Option) *AuditService {
	return &AuditService{baseService: newBase(ports, opts)}
}

// List returns paginated audit logs.
func (s *AuditService) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.AuditLog], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.AuditLogs.List(ctx, q)
}

// ListByEntity returns audit logs for a single entity.
func (s *AuditService) ListByEntity(ctx context.Context, entityType, entityID string) ([]*domain.AuditLog, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.AuditLogs.ListByEntity(ctx, entityType, entityID)
}

// ListPlanVersionsForAudit exports the immutable plan version history to audit
// consumers. Snapshots are returned exactly as recorded: an old entry keeps the
// cycle that was in effect at the time it was created, never the live value.
// Use domain.EffectiveCycleAt to reconstruct the cycle that was in effect at a
// given instant.
func (s *AuditService) ListPlanVersionsForAudit(ctx context.Context, planID string) ([]*domain.PlanVersion, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Plans.ListPlanVersions(ctx, planID)
}
