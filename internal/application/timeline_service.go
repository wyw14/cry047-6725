package application

import (
	"context"

	"github.com/cry047/baseline/internal/domain"
)

// TimelineService is the read-only access to facility timelines.
type TimelineService struct {
	baseService
}

// NewTimelineService returns a TimelineService.
func NewTimelineService(ports *domain.Ports, opts ...Option) *TimelineService {
	return &TimelineService{baseService: newBase(ports, opts)}
}

// ListByFacility returns the timeline for a facility.
func (s *TimelineService) ListByFacility(ctx context.Context, facilityID string, limit int) ([]*domain.TimelineEvent, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Timeline.ListByFacility(ctx, facilityID, limit)
}
