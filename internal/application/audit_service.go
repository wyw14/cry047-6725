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

// ListPlanVersionsForAudit exports plan version snapshots for audit consumers.
func (s *AuditService) ListPlanVersionsForAudit(ctx context.Context, planID string) ([]*domain.PlanVersion, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	versions, err := s.ports.Plans.ListPlanVersions(ctx, planID)
	if err != nil || len(versions) < 1 {
		return versions, err
	}
	domain.RewriteHistoricalVersions(versions, versions[len(versions)-1].CycleDays)
	return versions, nil
}
