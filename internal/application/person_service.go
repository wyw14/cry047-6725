package application

import (
	"context"

	"github.com/cry047/baseline/internal/domain"
)

// PersonService orchestrates responsible person lifecycle.
type PersonService struct {
	baseService
}

// NewPersonService returns a PersonService.
func NewPersonService(ports *domain.Ports, opts ...Option) *PersonService {
	return &PersonService{baseService: newBase(ports, opts)}
}

// Create inserts a new responsible person.
func (s *PersonService) Create(ctx context.Context, actor domain.Actor, in domain.ResponsiblePersonInput) (*domain.ResponsiblePerson, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	p := &domain.ResponsiblePerson{
		ID:         s.id(),
		Name:       in.Name,
		Email:      in.Email,
		Phone:      in.Phone,
		Department: in.Department,
		Active:     true,
	}
	if err := s.ports.People.Create(ctx, p); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditCreate, "ResponsiblePerson", p.ID, actor, nil, p, "")
	return p, nil
}

// Update modifies an existing person.
func (s *PersonService) Update(ctx context.Context, actor domain.Actor, id string, in domain.ResponsiblePersonInput) (*domain.ResponsiblePerson, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.People.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *before
	updated.Name = in.Name
	updated.Email = in.Email
	updated.Phone = in.Phone
	updated.Department = in.Department
	updated.Version = before.Version + 1
	if err := s.ports.People.Update(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditUpdate, "ResponsiblePerson", id, actor, before, &updated, "")
	return &updated, nil
}

// SetActive toggles the active flag of a person.
func (s *PersonService) SetActive(ctx context.Context, actor domain.Actor, id string, active bool) (*domain.ResponsiblePerson, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.People.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *before
	updated.Active = active
	updated.Version = before.Version + 1
	if err := s.ports.People.Update(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditUpdate, "ResponsiblePerson", id, actor, before, &updated, "")
	return &updated, nil
}

// Delete removes a person.
func (s *PersonService) Delete(ctx context.Context, actor domain.Actor, id string) error {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.People.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.ports.People.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.audit(ctx, domain.AuditDelete, "ResponsiblePerson", id, actor, before, nil, "")
	return nil
}

// Get retrieves a person.
func (s *PersonService) Get(ctx context.Context, id string) (*domain.ResponsiblePerson, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.People.Get(ctx, id)
}

// List returns a paginated list of people.
func (s *PersonService) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.ResponsiblePerson], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.People.List(ctx, q)
}
