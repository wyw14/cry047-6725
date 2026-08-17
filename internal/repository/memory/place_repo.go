package memory

import (
	"context"
	"sort"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// PlaceRepository is the in-memory implementation.
type PlaceRepository struct{ store *Store }

func NewPlaceRepository(s *Store) *PlaceRepository { return &PlaceRepository{store: s} }

func (r *PlaceRepository) Create(ctx context.Context, p *domain.Place) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.placesByCode[p.Code]; ok {
		return domain.ErrConflict("场所编码已存在: "+p.Code, nil)
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
	r.store.places[p.ID] = &cp
	r.store.placesByCode[p.Code] = p.ID
	return nil
}

func (r *PlaceRepository) Update(ctx context.Context, p *domain.Place) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.places[p.ID]
	if !ok {
		return domain.ErrNotFound("Place", p.ID)
	}
	if existing.Version != p.Version-1 {
		return domain.ErrConflict("版本冲突: 期望 "+itoa(existing.Version), nil)
	}
	// If the code changed, free the old code and reserve the new.
	if existing.Code != p.Code {
		if _, occupied := r.store.placesByCode[p.Code]; occupied {
			return domain.ErrConflict("场所编码已被占用: "+p.Code, nil)
		}
		delete(r.store.placesByCode, existing.Code)
		r.store.placesByCode[p.Code] = p.ID
	}
	p.UpdatedAt = now()
	cp := *p
	r.store.places[p.ID] = &cp
	return nil
}

func (r *PlaceRepository) Delete(ctx context.Context, id string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.places[id]
	if !ok {
		return domain.ErrNotFound("Place", id)
	}
	delete(r.store.places, id)
	delete(r.store.placesByCode, existing.Code)
	return nil
}

func (r *PlaceRepository) Get(ctx context.Context, id string) (*domain.Place, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	p, ok := r.store.places[id]
	if !ok {
		return nil, domain.ErrNotFound("Place", id)
	}
	cp := *p
	return &cp, nil
}

func (r *PlaceRepository) GetByCode(ctx context.Context, code string) (*domain.Place, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	id, ok := r.store.placesByCode[code]
	if !ok {
		return nil, domain.ErrNotFound("Place", code)
	}
	cp := *r.store.places[id]
	return &cp, nil
}

func (r *PlaceRepository) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Place], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.Place, 0, len(r.store.places))
	for _, p := range r.store.places {
		cp := *p
		items = append(items, &cp)
	}
	r.store.mu.RUnlock()

	col := r.store.applySortWhitelist("places", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		return comparePlace(items[i], items[j], col, q.Order == "desc")
	})

	// Apply filter by type
	if f, ok := q.Filters["type"]; ok && f != "" {
		filtered := items[:0]
		for _, it := range items {
			if string(it.Type) == f {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	if f, ok := q.Filters["name"]; ok && f != "" {
		filtered := items[:0]
		for _, it := range items {
			if containsString(it.Name, f) {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	return paginate(items, q), nil
}

func comparePlace(a, b *domain.Place, col string, desc bool) bool {
	var less bool
	switch col {
	case "name":
		less = a.Name < b.Name
	case "code":
		less = a.Code < b.Code
	case "created_at":
		less = a.CreatedAt.Before(b.CreatedAt)
	default:
		less = a.CreatedAt.Before(b.CreatedAt)
	}
	if desc {
		return !less && a.ID != b.ID
	}
	return less
}

func containsString(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// itoa local int-to-string.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// time.Now alias to keep tests deterministic if needed.
var _ = time.Now
