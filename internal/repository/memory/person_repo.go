package memory

import (
	"context"
	"sort"

	"github.com/cry047/baseline/internal/domain"
)

type ResponsiblePersonRepository struct{ store *Store }

func NewResponsiblePersonRepository(s *Store) *ResponsiblePersonRepository {
	return &ResponsiblePersonRepository{store: s}
}

func (r *ResponsiblePersonRepository) Create(ctx context.Context, p *domain.ResponsiblePerson) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.people[p.ID]; ok {
		return domain.ErrConflict("责任人已存在: "+p.ID, nil)
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now()
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = p.CreatedAt
	}
	if p.Version == 0 {
		p.Version = 1
	}
	cp := *p
	r.store.people[p.ID] = &cp
	return nil
}

func (r *ResponsiblePersonRepository) Update(ctx context.Context, p *domain.ResponsiblePerson) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.people[p.ID]
	if !ok {
		return domain.ErrNotFound("ResponsiblePerson", p.ID)
	}
	if existing.Version != p.Version-1 {
		return domain.ErrConflict("版本冲突", nil)
	}
	p.UpdatedAt = now()
	cp := *p
	r.store.people[p.ID] = &cp
	return nil
}

func (r *ResponsiblePersonRepository) Delete(ctx context.Context, id string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.people[id]; !ok {
		return domain.ErrNotFound("ResponsiblePerson", id)
	}
	delete(r.store.people, id)
	return nil
}

func (r *ResponsiblePersonRepository) Get(ctx context.Context, id string) (*domain.ResponsiblePerson, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	p, ok := r.store.people[id]
	if !ok {
		return nil, domain.ErrNotFound("ResponsiblePerson", id)
	}
	cp := *p
	return &cp, nil
}

func (r *ResponsiblePersonRepository) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.ResponsiblePerson], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.ResponsiblePerson, 0, len(r.store.people))
	for _, p := range r.store.people {
		cp := *p
		items = append(items, &cp)
	}
	r.store.mu.RUnlock()
	col := r.store.applySortWhitelist("responsible_persons", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		switch col {
		case "name":
			return items[i].Name < items[j].Name
		default:
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
	})
	if f, ok := q.Filters["active"]; ok && f != "" {
		filtered := items[:0]
		for _, it := range items {
			if (it.Active && f == "true") || (!it.Active && f == "false") {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	if f, ok := q.Filters["department"]; ok && f != "" {
		filtered := items[:0]
		for _, it := range items {
			if it.Department == f {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	return paginate(items, q), nil
}
