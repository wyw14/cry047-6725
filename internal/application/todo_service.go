package application

import (
	"context"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// TodoService orchestrates todo generation, assignment and completion.
type TodoService struct {
	baseService
}

// NewTodoService returns a TodoService.
func NewTodoService(ports *domain.Ports, opts ...Option) *TodoService {
	return &TodoService{baseService: newBase(ports, opts)}
}

// Create inserts a new todo. Idempotency is enforced via IdempotencyKey.
func (s *TodoService) Create(ctx context.Context, actor domain.Actor, in domain.TodoInput) (*domain.Todo, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	if existing, err := s.ports.Todos.GetByIdempotencyKey(ctx, in.IdempotencyKey); err == nil {
		return existing, nil
	}
	t := &domain.Todo{
		ID:             s.id(),
		FacilityID:     in.FacilityID,
		PlanID:         in.PlanID,
		AnomalyID:      in.AnomalyID,
		AssignedTo:     in.AssignedTo,
		Type:           in.Type,
		DueDate:        in.DueDate,
		Status:         domain.TodoAssigned,
		Title:          in.Title,
		IdempotencyKey: in.IdempotencyKey,
	}
	if err := s.ports.Todos.Create(ctx, t); err != nil {
		return nil, err
	}
	_ = s.notify(ctx, domain.NotificationInput{
		UserID: in.AssignedTo,
		Type:   domain.NotificationAssigned,
		Title:  "新待办: " + in.Title,
		Body:   "截止日期: " + in.DueDate.Format(time.DateOnly),
	})
	return t, nil
}

// Reassign moves a todo to another user.
func (s *TodoService) Reassign(ctx context.Context, actor domain.Actor, id, newUserID, reason string) (*domain.Todo, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Todos.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.ports.People.Get(ctx, newUserID); err != nil {
		return nil, err
	}
	updated := *before
	updated.AssignedTo = newUserID
	updated.Version = before.Version + 1
	if err := s.ports.Todos.Update(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditUpdate, "Todo", id, actor, before, &updated, "reassign: "+reason)
	_ = s.notify(ctx, domain.NotificationInput{
		UserID: newUserID,
		Type:   domain.NotificationAssigned,
		Title:  "待办转派: " + before.Title,
		Body:   "原因: " + reason,
	})
	return &updated, nil
}

// Complete marks a todo as completed.
func (s *TodoService) Complete(ctx context.Context, actor domain.Actor, id string) (*domain.Todo, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Todos.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if before.Status == domain.TodoCompleted {
		return before, nil
	}
	updated := *before
	updated.Status = domain.TodoCompleted
	updated.CompletedAt = s.now()
	updated.Version = before.Version + 1
	if err := s.ports.Todos.Update(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditUpdate, "Todo", id, actor, before, &updated, "completed")
	return &updated, nil
}

// Skip marks a todo as skipped.
func (s *TodoService) Skip(ctx context.Context, actor domain.Actor, id, reason string) (*domain.Todo, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	before, err := s.ports.Todos.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *before
	updated.Status = domain.TodoSkipped
	updated.Version = before.Version + 1
	if err := s.ports.Todos.Update(ctx, &updated); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, domain.AuditUpdate, "Todo", id, actor, before, &updated, "skip: "+reason)
	return &updated, nil
}

// Get retrieves a todo.
func (s *TodoService) Get(ctx context.Context, id string) (*domain.Todo, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Todos.Get(ctx, id)
}

// List returns paginated todos.
func (s *TodoService) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Todo], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Todos.List(ctx, q)
}

// ListByUser returns paginated todos for a user.
func (s *TodoService) ListByUser(ctx context.Context, userID string, q domain.PageQuery) (*domain.PageResult[*domain.Todo], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Todos.ListByUser(ctx, userID, q)
}
