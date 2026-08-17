package memory

import (
	"context"
	"sort"

	"github.com/cry047/baseline/internal/domain"
)

type FacilityRepository struct{ store *Store }

func NewFacilityRepository(s *Store) *FacilityRepository { return &FacilityRepository{store: s} }

func (r *FacilityRepository) Create(ctx context.Context, f *domain.Facility) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.facilities[f.ID]; ok {
		return domain.ErrConflict("设施已存在: "+f.ID, nil)
	}
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now()
	}
	if f.UpdatedAt.IsZero() {
		f.UpdatedAt = f.CreatedAt
	}
	if f.Version == 0 {
		f.Version = 1
	}
	cp := *f
	r.store.facilities[f.ID] = &cp
	return nil
}

func (r *FacilityRepository) Update(ctx context.Context, f *domain.Facility) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.facilities[f.ID]
	if !ok {
		return domain.ErrNotFound("Facility", f.ID)
	}
	if existing.Version != f.Version-1 {
		return domain.ErrConflict("版本冲突: 设施 "+f.ID, nil)
	}
	f.UpdatedAt = now()
	cp := *f
	r.store.facilities[f.ID] = &cp
	return nil
}

func (r *FacilityRepository) UpdateStatus(ctx context.Context, id string, status domain.FacilityStatus, version int) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.facilities[id]
	if !ok {
		return domain.ErrNotFound("Facility", id)
	}
	if existing.Version != version {
		return domain.ErrConflict("版本冲突: 设施 "+id, nil)
	}
	existing.Status = status
	existing.Version = version + 1
	existing.UpdatedAt = now()
	return nil
}

func (r *FacilityRepository) UpdateStatusChecked(ctx context.Context, id string, status domain.FacilityStatus, version int) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.facilities[id]
	if !ok {
		return domain.ErrNotFound("Facility", id)
	}
	if existing.Version != version {
		return domain.ErrConflict("版本冲突: 设施 "+id, nil)
	}
	transition := domain.FacilityStatusTransition{From: existing.Status, To: status, Reason: "checked repository update"}
	if err := transition.ValidateTransition(existing.Criticality); err != nil {
		return err
	}
	existing.Status = status
	existing.Version = version + 1
	existing.UpdatedAt = now()
	return nil
}

func (r *FacilityRepository) Delete(ctx context.Context, id string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.facilities[id]; !ok {
		return domain.ErrNotFound("Facility", id)
	}
	delete(r.store.facilities, id)
	return nil
}

func (r *FacilityRepository) Get(ctx context.Context, id string) (*domain.Facility, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	f, ok := r.store.facilities[id]
	if !ok {
		return nil, domain.ErrNotFound("Facility", id)
	}
	cp := *f
	return &cp, nil
}

func (r *FacilityRepository) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Facility], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.Facility, 0, len(r.store.facilities))
	for _, f := range r.store.facilities {
		cp := *f
		items = append(items, &cp)
	}
	r.store.mu.RUnlock()
	col := r.store.applySortWhitelist("facilities", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		return compareFacility(items[i], items[j], col, q.Order == "desc")
	})
	items = applyFacilityFilters(items, q.Filters)
	return paginate(items, q), nil
}

func (r *FacilityRepository) ListByPlace(ctx context.Context, placeID string) ([]*domain.Facility, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := make([]*domain.Facility, 0)
	for _, f := range r.store.facilities {
		if f.PlaceID == placeID {
			cp := *f
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func (r *FacilityRepository) ListByResponsiblePerson(ctx context.Context, personID string) ([]*domain.Facility, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := make([]*domain.Facility, 0)
	for _, f := range r.store.facilities {
		if f.ResponsiblePersonID == personID {
			cp := *f
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func compareFacility(a, b *domain.Facility, col string, desc bool) bool {
	var less bool
	switch col {
	case "name":
		less = a.Name < b.Name
	case "code":
		less = a.Code < b.Code
	case "criticality":
		less = a.Criticality < b.Criticality
	case "status":
		less = a.Status < b.Status
	default:
		less = a.CreatedAt.Before(b.CreatedAt)
	}
	if desc {
		return !less && a.ID != b.ID
	}
	return less
}

func applyFacilityFilters(items []*domain.Facility, filters map[string]string) []*domain.Facility {
	if len(filters) == 0 {
		return items
	}
	out := items[:0]
	for _, it := range items {
		match := true
		if v, ok := filters["place_id"]; ok && v != "" && it.PlaceID != v {
			match = false
		}
		if match {
			if v, ok := filters["criticality"]; ok && v != "" && string(it.Criticality) != v {
				match = false
			}
		}
		if match {
			if v, ok := filters["status"]; ok && v != "" && string(it.Status) != v {
				match = false
			}
		}
		if match {
			if v, ok := filters["responsible_person_id"]; ok && v != "" && it.ResponsiblePersonID != v {
				match = false
			}
		}
		if match {
			if v, ok := filters["category"]; ok && v != "" && it.Category != v {
				match = false
			}
		}
		if match {
			if v, ok := filters["name"]; ok && v != "" && !containsString(it.Name, v) {
				match = false
			}
		}
		if match {
			out = append(out, it)
		}
	}
	return out
}
