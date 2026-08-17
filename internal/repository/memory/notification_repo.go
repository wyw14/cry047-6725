package memory

import (
	"context"
	"sort"

	"github.com/cry047/baseline/internal/domain"
)

type NotificationRepository struct{ store *Store }

func NewNotificationRepository(s *Store) *NotificationRepository {
	return &NotificationRepository{store: s}
}

func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = now()
	}
	cp := *n
	r.store.notifications[n.ID] = &cp
	return nil
}

func (r *NotificationRepository) Get(ctx context.Context, id string) (*domain.Notification, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	n, ok := r.store.notifications[id]
	if !ok {
		return nil, domain.ErrNotFound("Notification", id)
	}
	cp := *n
	return &cp, nil
}

func (r *NotificationRepository) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Notification], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.Notification, 0, len(r.store.notifications))
	for _, n := range r.store.notifications {
		cp := *n
		items = append(items, &cp)
	}
	r.store.mu.RUnlock()
	col := r.store.applySortWhitelist("notifications", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		switch col {
		default:
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
	})
	if v, ok := q.Filters["user_id"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if it.UserID == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	if v, ok := q.Filters["type"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if string(it.Type) == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	return paginate(items, q), nil
}

func (r *NotificationRepository) ListByUser(ctx context.Context, userID string, q domain.PageQuery) (*domain.PageResult[*domain.Notification], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.Notification, 0)
	for _, n := range r.store.notifications {
		if n.UserID == userID {
			cp := *n
			items = append(items, &cp)
		}
	}
	r.store.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return paginate(items, q), nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	n, ok := r.store.notifications[id]
	if !ok {
		return domain.ErrNotFound("Notification", id)
	}
	if n.ReadAt.IsZero() {
		n.ReadAt = now()
	}
	return nil
}

func (r *NotificationRepository) ListUnread(ctx context.Context, userID string) ([]*domain.Notification, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := []*domain.Notification{}
	for _, n := range r.store.notifications {
		if n.UserID == userID && n.ReadAt.IsZero() {
			cp := *n
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
