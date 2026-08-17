package domain

import (
	"context"
	"time"
)

// NotificationType enumerates the kind of notification.
type NotificationType string

const (
	NotificationMaintenanceDue     NotificationType = "maintenance_due"
	NotificationMaintenanceOverdue NotificationType = "maintenance_overdue"
	NotificationAnomalyDiscovered  NotificationType = "anomaly_discovered"
	NotificationAnomalyRecovered   NotificationType = "anomaly_recovered"
	NotificationPlanUpdated        NotificationType = "plan_updated"
	NotificationAssigned           NotificationType = "assigned"
)

// Notification is a local, durable notification record.
type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Body      string           `json:"body"`
	ReadAt    time.Time        `json:"read_at,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

// NotificationInput is the validated input for creating a notification.
type NotificationInput struct {
	UserID string           `json:"user_id"`
	Type   NotificationType `json:"type"`
	Title  string           `json:"title"`
	Body   string           `json:"body"`
}

// Validate enforces notification invariants.
func (i NotificationInput) Validate() error {
	if i.UserID == "" {
		return ErrInvalid("用户 ID 不能为空", "user_id", nil)
	}
	if len(i.Title) < 2 || len(i.Title) > 256 {
		return ErrInvalid("通知标题长度需在 2-256 之间", "title", nil)
	}
	switch i.Type {
	case NotificationMaintenanceDue, NotificationMaintenanceOverdue, NotificationAnomalyDiscovered, NotificationAnomalyRecovered, NotificationPlanUpdated, NotificationAssigned:
	default:
		return ErrInvalid("通知类型不在白名单内", "type", nil)
	}
	if len(i.Body) > 4096 {
		return ErrInvalid("通知正文长度不能超过 4096", "body", nil)
	}
	return nil
}

// NotificationRepository is the persistence contract.
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	Get(ctx context.Context, id string) (*Notification, error)
	List(ctx context.Context, q PageQuery) (*PageResult[*Notification], error)
	ListByUser(ctx context.Context, userID string, q PageQuery) (*PageResult[*Notification], error)
	MarkRead(ctx context.Context, id string) error
	ListUnread(ctx context.Context, userID string) ([]*Notification, error)
}

// Notifier is the local adapter contract for delivering notifications.
// The implementation is fully offline and must record every notification
// into a durable store and, optionally, to an in-memory ring buffer.
type Notifier interface {
	Notify(ctx context.Context, n *Notification) error
}
