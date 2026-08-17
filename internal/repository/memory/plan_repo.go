package memory

import (
	"context"
	"sort"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

type PlanRepository struct{ store *Store }

func NewPlanRepository(s *Store) *PlanRepository { return &PlanRepository{store: s} }

// --- Templates ---

func (r *PlanRepository) CreateTemplate(ctx context.Context, t *domain.PlanTemplate) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.templates[t.ID]; ok {
		return domain.ErrConflict("模板已存在: "+t.ID, nil)
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now()
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = t.CreatedAt
	}
	if t.Version == 0 {
		t.Version = 1
	}
	cp := *t
	r.store.templates[t.ID] = &cp
	return nil
}

func (r *PlanRepository) UpdateTemplate(ctx context.Context, t *domain.PlanTemplate) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.templates[t.ID]
	if !ok {
		return domain.ErrNotFound("PlanTemplate", t.ID)
	}
	if existing.Version != t.Version-1 {
		return domain.ErrConflict("版本冲突: 模板 "+t.ID, nil)
	}
	t.UpdatedAt = now()
	cp := *t
	r.store.templates[t.ID] = &cp
	return nil
}

func (r *PlanRepository) GetTemplate(ctx context.Context, id string) (*domain.PlanTemplate, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	t, ok := r.store.templates[id]
	if !ok {
		return nil, domain.ErrNotFound("PlanTemplate", id)
	}
	cp := *t
	return &cp, nil
}

func (r *PlanRepository) ListTemplates(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.PlanTemplate], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.PlanTemplate, 0, len(r.store.templates))
	for _, t := range r.store.templates {
		cp := *t
		items = append(items, &cp)
	}
	r.store.mu.RUnlock()
	col := r.store.applySortWhitelist("plan_templates", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		switch col {
		case "name":
			return items[i].Name < items[j].Name
		case "cycle_days":
			return items[i].CycleDays < items[j].CycleDays
		default:
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
	})
	if v, ok := q.Filters["name"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if containsString(it.Name, v) {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	return paginate(items, q), nil
}

// --- Plans ---

func (r *PlanRepository) CreatePlan(ctx context.Context, p *domain.MaintenancePlan) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.plans[p.ID]; ok {
		return domain.ErrConflict("计划已存在: "+p.ID, nil)
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
	r.store.plans[p.ID] = &cp
	return nil
}

func (r *PlanRepository) UpdatePlan(ctx context.Context, p *domain.MaintenancePlan) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	existing, ok := r.store.plans[p.ID]
	if !ok {
		return domain.ErrNotFound("MaintenancePlan", p.ID)
	}
	if existing.Version != p.Version-1 {
		return domain.ErrConflict("版本冲突: 计划 "+p.ID, nil)
	}
	p.UpdatedAt = now()
	cp := *p
	r.store.plans[p.ID] = &cp
	return nil
}

func (r *PlanRepository) GetPlan(ctx context.Context, id string) (*domain.MaintenancePlan, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	p, ok := r.store.plans[id]
	if !ok {
		return nil, domain.ErrNotFound("MaintenancePlan", id)
	}
	cp := *p
	return &cp, nil
}

func (r *PlanRepository) GetPlanByFacility(ctx context.Context, facilityID string) (*domain.MaintenancePlan, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	for _, p := range r.store.plans {
		if p.FacilityID == facilityID {
			cp := *p
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound("MaintenancePlan by facility", facilityID)
}

func (r *PlanRepository) ListPlans(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.MaintenancePlan], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.MaintenancePlan, 0, len(r.store.plans))
	for _, p := range r.store.plans {
		cp := *p
		items = append(items, &cp)
	}
	r.store.mu.RUnlock()
	col := r.store.applySortWhitelist("plans", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		switch col {
		case "next_due_date":
			return items[i].NextDueDate.Before(items[j].NextDueDate)
		case "facility_id":
			return items[i].FacilityID < items[j].FacilityID
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
	if v, ok := q.Filters["active"]; ok && v != "" {
		filtered := items[:0]
		active := v == "true"
		for _, it := range items {
			if it.Active == active {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	return paginate(items, q), nil
}

func (r *PlanRepository) ListOverduePlans(ctx context.Context, before time.Time) ([]*domain.MaintenancePlan, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := []*domain.MaintenancePlan{}
	for _, p := range r.store.plans {
		if p.Active && !p.NextDueDate.IsZero() && p.NextDueDate.Before(before) {
			cp := *p
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NextDueDate.Before(out[j].NextDueDate) })
	return out, nil
}

func (r *PlanRepository) ListDuePlans(ctx context.Context, before time.Time) ([]*domain.MaintenancePlan, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := []*domain.MaintenancePlan{}
	for _, p := range r.store.plans {
		if p.Active && !p.NextDueDate.IsZero() && (p.NextDueDate.Before(before) || p.NextDueDate.Equal(before)) {
			cp := *p
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NextDueDate.Before(out[j].NextDueDate) })
	return out, nil
}

// --- Plan Versions ---

func (r *PlanRepository) AppendPlanVersion(ctx context.Context, v *domain.PlanVersion) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if v.ChangedAt.IsZero() {
		v.ChangedAt = now()
	}
	r.store.planVersions = append(r.store.planVersions, *v)
	return nil
}

func (r *PlanRepository) ListPlanVersions(ctx context.Context, planID string) ([]*domain.PlanVersion, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := []*domain.PlanVersion{}
	for i := range r.store.planVersions {
		if r.store.planVersions[i].PlanID == planID {
			cp := r.store.planVersions[i]
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].VersionNumber < out[j].VersionNumber })
	return out, nil
}
