package memory

import (
	"context"
	"sort"

	"github.com/cry047/baseline/internal/domain"
)

type AnomalyRepository struct{ store *Store }

func NewAnomalyRepository(s *Store) *AnomalyRepository { return &AnomalyRepository{store: s} }

func (r *AnomalyRepository) Create(ctx context.Context, a *domain.Anomaly) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.anomalies[a.ID]; ok {
		return domain.ErrConflict("异常已存在: "+a.ID, nil)
	}
	if a.IdempotencyKey != "" {
		if _, ok := r.store.anomaliesByKey[a.IdempotencyKey]; ok {
			return domain.ErrConflict("幂等键已存在: "+a.IdempotencyKey, nil)
		}
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now()
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = a.CreatedAt
	}
	if a.Version == 0 {
		a.Version = 1
	}
	cp := *a
	r.store.anomalies[a.ID] = &cp
	if a.IdempotencyKey != "" {
		r.store.anomaliesByKey[a.IdempotencyKey] = a.ID
	}
	return nil
}

func (r *AnomalyRepository) Update(ctx context.Context, a *domain.Anomaly) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.anomalies[a.ID]
	if !ok {
		return domain.ErrNotFound("Anomaly", a.ID)
	}
	if existing.Version != a.Version-1 {
		return domain.ErrConflict("版本冲突: 异常 "+a.ID, nil)
	}
	a.UpdatedAt = now()
	cp := *a
	r.store.anomalies[a.ID] = &cp
	return nil
}

func (r *AnomalyRepository) Get(ctx context.Context, id string) (*domain.Anomaly, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	a, ok := r.store.anomalies[id]
	if !ok {
		return nil, domain.ErrNotFound("Anomaly", id)
	}
	cp := *a
	return &cp, nil
}

func (r *AnomalyRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Anomaly, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	id, ok := r.store.anomaliesByKey[key]
	if !ok {
		return nil, domain.ErrNotFound("Anomaly by idempotency_key", key)
	}
	cp := *r.store.anomalies[id]
	return &cp, nil
}

func (r *AnomalyRepository) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Anomaly], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.Anomaly, 0, len(r.store.anomalies))
	for _, a := range r.store.anomalies {
		cp := *a
		items = append(items, &cp)
	}
	r.store.mu.RUnlock()
	col := r.store.applySortWhitelist("anomalies", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		switch col {
		case "discovered_at":
			return items[i].DiscoveredAt.Before(items[j].DiscoveredAt)
		case "severity":
			return items[i].Severity < items[j].Severity
		case "status":
			return items[i].Status < items[j].Status
		default:
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
	})
	if v, ok := q.Filters["facility_id"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if it.FacilityID == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	if v, ok := q.Filters["status"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if string(it.Status) == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	if v, ok := q.Filters["severity"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if string(it.Severity) == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	return paginate(items, q), nil
}

func (r *AnomalyRepository) ListByFacility(ctx context.Context, facilityID string, limit int) ([]*domain.Anomaly, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	out := []*domain.Anomaly{}
	for _, a := range r.store.anomalies {
		if a.FacilityID == facilityID {
			cp := *a
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DiscoveredAt.After(out[j].DiscoveredAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *AnomalyRepository) ListOpen(ctx context.Context) ([]*domain.Anomaly, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := []*domain.Anomaly{}
	for _, a := range r.store.anomalies {
		if a.Status == domain.AnomalyOpen || a.Status == domain.AnomalyRectifying || a.Status == domain.AnomalyReinspecting {
			cp := *a
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DiscoveredAt.Before(out[j].DiscoveredAt) })
	return out, nil
}
