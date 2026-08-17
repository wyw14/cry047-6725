package memory

import (
	"context"
	"sort"

	"github.com/cry047/baseline/internal/domain"
)

type AuditLogRepository struct{ store *Store }

func NewAuditLogRepository(s *Store) *AuditLogRepository { return &AuditLogRepository{store: s} }

func (r *AuditLogRepository) Create(ctx context.Context, l *domain.AuditLog) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now()
	}
	r.store.auditLogs = append(r.store.auditLogs, *l)
	return nil
}

func (r *AuditLogRepository) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.AuditLog], error) {
	q.Normalize()
	r.store.mu.RLock()
	items := make([]*domain.AuditLog, 0, len(r.store.auditLogs))
	for i := range r.store.auditLogs {
		cp := r.store.auditLogs[i]
		items = append(items, &cp)
	}
	r.store.mu.RUnlock()
	col := r.store.applySortWhitelist("audit_logs", q.OrderBy)
	sort.Slice(items, func(i, j int) bool {
		switch col {
		case "entity_type":
			return items[i].EntityType < items[j].EntityType
		default:
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
	})
	if v, ok := q.Filters["entity_type"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if it.EntityType == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	if v, ok := q.Filters["entity_id"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if it.EntityID == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	if v, ok := q.Filters["actor_id"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if it.ActorID == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	if v, ok := q.Filters["action"]; ok && v != "" {
		filtered := items[:0]
		for _, it := range items {
			if string(it.Action) == v {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	return paginate(items, q), nil
}

func (r *AuditLogRepository) ListByEntity(ctx context.Context, entityType, entityID string) ([]*domain.AuditLog, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := []*domain.AuditLog{}
	for i := range r.store.auditLogs {
		l := r.store.auditLogs[i]
		if l.EntityType == entityType && l.EntityID == entityID {
			cp := l
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
