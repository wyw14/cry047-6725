package memory

import (
	"context"
	"sort"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

type TodoRepository struct{ store *Store }

func NewTodoRepository(s *Store) *TodoRepository { return &TodoRepository{store: s} }

func (r *TodoRepository) Create(ctx context.Context, t *domain.Todo) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.todos[t.ID]; ok {
		return domain.ErrConflict("待办已存在: "+t.ID, nil)
	}
	if t.IdempotencyKey != "" {
		if _, ok := r.store.todosByKey[t.IdempotencyKey]; ok {
			return domain.ErrConflict("幂等键已存在: "+t.IdempotencyKey, nil)
		}
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now()
	}
	if t.Version == 0 {
		t.Version = 1
	}
	cp := *t
	r.store.todos[t.ID] = &cp
	if t.IdempotencyKey != "" {
		r.store.todosByKey[t.IdempotencyKey] = t.ID
	}
	return nil
}

func (r *TodoRepository) Update(ctx context.Context, t *domain.Todo) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.todos[t.ID]
	if !ok {
		return domain.ErrNotFound("Todo", t.ID)
	}
	if existing.Version != t.Version-1 {
		return domain.ErrConflict("版本冲突: 待办 "+t.ID, nil)
	}
	cp := *t
	r.store.todos[t.ID] = &cp
	return nil
}

func (r *TodoRepository) Get(ctx context.Context, id string) (*domain.Todo, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	t, ok := r.store.todos[id]
	if !ok {
		return nil, domain.ErrNotFound("Todo", id)
	}
	cp := *t
	return &cp, nil
}

func (r *TodoRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Todo, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	id, ok := r.store.todosByKey[key]
	if !ok {
		return nil, domain.ErrNotFound("Todo by idempotency_key", key)
	}
	cp := *r.store.todos[id]
	return &cp, nil
}

func (r *TodoRepository) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Todo], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.Todo, 0, len(r.store.todos))
	for _, t := range r.store.todos {
		cp := *t
		items = append(items, &cp)
	}
	r.store.mu.RUnlock()
	col := r.store.applySortWhitelist("todos", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		switch col {
		case "due_date":
			return items[i].DueDate.Before(items[j].DueDate)
		case "status":
			return items[i].Status < items[j].Status
		case "assigned_to":
			return items[i].AssignedTo < items[j].AssignedTo
		default:
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
	})
	if v, ok := q.Filters["status"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if string(it.Status) == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	if v, ok := q.Filters["assigned_to"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if it.AssignedTo == v {
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
	return paginate(items, q), nil
}

func (r *TodoRepository) ListByUser(ctx context.Context, userID string, q domain.PageQuery) (*domain.PageResult[*domain.Todo], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.Todo, 0)
	for _, t := range r.store.todos {
		if t.AssignedTo == userID {
			cp := *t
			items = append(items, &cp)
		}
	}
	r.store.mu.RUnlock()
	col := r.store.applySortWhitelist("todos", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		switch col {
		case "due_date":
			return items[i].DueDate.Before(items[j].DueDate)
		case "status":
			return items[i].Status < items[j].Status
		default:
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
	})
	return paginate(items, q), nil
}

func (r *TodoRepository) ListOverdue(ctx context.Context, before time.Time) ([]*domain.Todo, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := []*domain.Todo{}
	for _, t := range r.store.todos {
		if (t.Status == domain.TodoOpen || t.Status == domain.TodoAssigned) && t.DueDate.Before(before) {
			cp := *t
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DueDate.Before(out[j].DueDate) })
	return out, nil
}
