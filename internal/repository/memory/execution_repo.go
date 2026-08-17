package memory

import (
	"context"
	"sort"

	"github.com/cry047/baseline/internal/domain"
)

type ExecutionRepository struct{ store *Store }

func NewExecutionRepository(s *Store) *ExecutionRepository { return &ExecutionRepository{store: s} }

func (r *ExecutionRepository) Create(ctx context.Context, e *domain.Execution) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.executions[e.ID]; ok {
		return domain.ErrConflict("执行记录已存在: "+e.ID, nil)
	}
	if e.IdempotencyKey != "" {
		if _, ok := r.store.executionsByKey[e.IdempotencyKey]; ok {
			return domain.ErrConflict("幂等键已存在: "+e.IdempotencyKey, nil)
		}
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now()
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = e.CreatedAt
	}
	if e.Version == 0 {
		e.Version = 1
	}
	r.store.executions[e.ID] = e.Snapshot()
	if e.IdempotencyKey != "" {
		r.store.executionsByKey[e.IdempotencyKey] = e.ID
	}
	return nil
}

func (r *ExecutionRepository) Update(ctx context.Context, e *domain.Execution) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.executions[e.ID]
	if !ok {
		return domain.ErrNotFound("Execution", e.ID)
	}
	if existing.Version != e.Version-1 {
		return domain.ErrConflict("版本冲突: 执行记录 "+e.ID, nil)
	}
	e.UpdatedAt = now()
	r.store.executions[e.ID] = e.Snapshot()
	return nil
}

func (r *ExecutionRepository) Get(ctx context.Context, id string) (*domain.Execution, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	e, ok := r.store.executions[id]
	if !ok {
		return nil, domain.ErrNotFound("Execution", id)
	}
	return e.Snapshot(), nil
}

func (r *ExecutionRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Execution, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	id, ok := r.store.executionsByKey[key]
	if !ok {
		return nil, domain.ErrNotFound("Execution by idempotency_key", key)
	}
	return r.store.executions[id].Snapshot(), nil
}

func (r *ExecutionRepository) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Execution], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.Execution, 0, len(r.store.executions))
	for _, e := range r.store.executions {
		items = append(items, e.Snapshot())
	}
	r.store.mu.RUnlock()
	col := r.store.applySortWhitelist("executions", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		switch col {
		case "executed_at":
			return items[i].ExecutedAt.Before(items[j].ExecutedAt)
		case "facility_id":
			return items[i].FacilityID < items[j].FacilityID
		default:
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
	})
	if v, ok := q.Filters["plan_id"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if it.PlanID == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
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
	return paginate(items, q), nil
}

func (r *ExecutionRepository) ListByFacility(ctx context.Context, facilityID string, limit int) ([]*domain.Execution, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	out := []*domain.Execution{}
	for _, e := range r.store.executions {
		if e.FacilityID == facilityID {
			out = append(out, e.Snapshot())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExecutedAt.After(out[j].ExecutedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *ExecutionRepository) ListByPlan(ctx context.Context, planID string) ([]*domain.Execution, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := []*domain.Execution{}
	for _, e := range r.store.executions {
		if e.PlanID == planID {
			out = append(out, e.Snapshot())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExecutedAt.Before(out[j].ExecutedAt) })
	return out, nil
}
