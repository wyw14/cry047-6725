package application

import (
	"context"

	"github.com/cry047/baseline/internal/domain"
)

// NotificationService orchestrates local notifications.
type NotificationService struct {
	baseService
}

// NewNotificationService returns a NotificationService.
func NewNotificationService(ports *domain.Ports, opts ...Option) *NotificationService {
	return &NotificationService{baseService: newBase(ports, opts)}
}

// Get retrieves a notification.
func (s *NotificationService) Get(ctx context.Context, id string) (*domain.Notification, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Notifications.Get(ctx, id)
}

// List returns paginated notifications.
func (s *NotificationService) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Notification], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Notifications.List(ctx, q)
}

// ListByUser returns paginated notifications for a user.
func (s *NotificationService) ListByUser(ctx context.Context, userID string, q domain.PageQuery) (*domain.PageResult[*domain.Notification], error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Notifications.ListByUser(ctx, userID, q)
}

// MarkRead marks a notification as read.
func (s *NotificationService) MarkRead(ctx context.Context, actor domain.Actor, id string) error {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	if err := s.ports.Notifications.MarkRead(ctx, id); err != nil {
		return err
	}
	return nil
}

// ListUnread returns unread notifications for a user.
func (s *NotificationService) ListUnread(ctx context.Context, userID string) ([]*domain.Notification, error) {
	ctx, cancel := s.ctx(ctx)
	defer cancel()
	return s.ports.Notifications.ListUnread(ctx, userID)
}
