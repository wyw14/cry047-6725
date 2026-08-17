package application

import (
	"context"

	"github.com/cry047/baseline/internal/domain"
)

// PlaceService orchestrates place lifecycle.
type PlaceService struct {
	baseService
}

// NewPlaceService returns a PlaceService with the given ports and options.
func NewPlaceService(ports *domain.Ports, opts ...Option) *PlaceService {
	return &PlaceService{baseService: newBase(ports, opts)}
}

// Create inserts a new place.
func (s *PlaceService) Create(ctx context.Context, actor domain.Actor, in domain.PlaceInput) (*domain.Place, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	p := &domain.Place{
		ID:      s.id(),
		Name:    in.Name,
		Type:    in.Type,
		Address: in.Address,
		Code:    in.Code,
	}
	if err := s.ports.Places.Create(ctx, p); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditCreate, "Place", p.ID, actor, nil, p, "")
	return p, nil
}

// Update modifies an existing place.
func (s *PlaceService) Update(ctx context.Context, actor domain.Actor, id string, in domain.PlaceInput) (*domain.Place, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Places.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *before
	updated.Name = in.Name
	updated.Type = in.Type
	updated.Address = in.Address
	updated.Code = in.Code
	updated.Version = before.Version + 1
	if err := s.ports.Places.Update(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditUpdate, "Place", id, actor, before, &updated, "")
	return &updated, nil
}

// Delete removes a place by id.
func (s *PlaceService) Delete(ctx context.Context, actor domain.Actor, id string) error {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Places.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.ports.Places.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.audit(ctx, domain.AuditDelete, "Place", id, actor, before, nil, "")
	return nil
}

// Get retrieves a place.
func (s *PlaceService) Get(ctx context.Context, id string) (*domain.Place, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Places.Get(ctx, id)
}

// List returns a paginated list of places.
func (s *PlaceService) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Place], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Places.List(ctx, q)
}
