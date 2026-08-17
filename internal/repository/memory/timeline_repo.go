package memory

import (
	"context"
	"sort"

	"github.com/cry047/baseline/internal/domain"
)

type TimelineRepository struct{ store *Store }

func NewTimelineRepository(s *Store) *TimelineRepository { return &TimelineRepository{store: s} }

func (r *TimelineRepository) Append(ctx context.Context, e *domain.TimelineEvent) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if e.OccurredAt.IsZero() {
		e.OccurredAt = now()
	}
	cp := *e
	r.store.timeline[e.FacilityID] = append(r.store.timeline[e.FacilityID], cp)
	return nil
}

func (r *TimelineRepository) ListByFacility(ctx context.Context, facilityID string, limit int) ([]*domain.TimelineEvent, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	src := r.store.timeline[facilityID]
	out := make([]*domain.TimelineEvent, 0, len(src))
	for i := range src {
		cp := src[i]
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt.After(out[j].OccurredAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
