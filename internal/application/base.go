package application

import (
	"context"
	"time"

	"github.com/cry047/baseline/internal/domain"
	"github.com/google/uuid"
)

// baseService is the shared configuration and helpers for every application
// service. It is embedded by value into each concrete service.
type baseService struct {
	ports        *domain.Ports
	idgen        func() string
	timeout      time.Duration
	tickInterval time.Duration
}

func newBase(ports *domain.Ports, opts []Option) baseService {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return baseService{ports: ports, idgen: o.idgen, timeout: o.timeout, tickInterval: o.tickInterval}
}

func (b *baseService) id() string {
	if b.idgen == nil {
		return uuid.NewString()
	}
	return b.idgen()
}

func (b *baseService) ctx(parent context.Context) (context.Context, context.CancelFunc) {
	return domain.WithTimeout(parent, b.timeout)
}

func (b *baseService) now() time.Time {
	if b.ports.Clock != nil {
		return b.ports.Clock.Now()
	}
	return time.Now().UTC()
}

// audit is a small helper that records an audit log entry.
func (b *baseService) audit(ctx context.Context, action domain.AuditAction, entity, id string, actor domain.Actor, before, after any, reason string) error {
	return recordAudit(ctx, b.ports, domain.AuditLogInput{
		Action:      action,
		EntityType:  entity,
		EntityID:    id,
		Actor:       actor,
		BeforeState: before,
		AfterState:  after,
		Reason:      reason,
		RequestID:   domain.RequestIDFromContext(ctx),
	})
}

// timeline appends a timeline event for the given facility.
func (b *baseService) timeline(ctx context.Context, facilityID, eventType, title, actor string, detail any) error {
	var raw []byte
	if detail != nil {
		raw, _ = jsonMarshal(detail)
	}
	return b.ports.Timeline.Append(ctx, &domain.TimelineEvent{
		ID:         b.id(),
		FacilityID: facilityID,
		OccurredAt: b.now(),
		EventType:  eventType,
		Title:      title,
		Actor:      actor,
		Detail:     raw,
	})
}

// notify records a local notification and dispatches it via the notifier.
func (b *baseService) notify(ctx context.Context, in domain.NotificationInput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	n := &domain.Notification{
		ID:        b.id(),
		UserID:    in.UserID,
		Type:      in.Type,
		Title:     in.Title,
		Body:      in.Body,
		CreatedAt: b.now(),
	}
	if err := b.ports.Notifications.Create(ctx, n); err != nil {
		return err
	}
	if b.ports.Notifier != nil {
		_ = b.ports.Notifier.Notify(ctx, n)
	}
	return nil
}
