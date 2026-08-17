package domain

import (
	"context"
	"time"
)

// Pagination is the shared pagination + sorting + filtering contract used by
// every list endpoint. Whitelisting is enforced by the repository layer.
type PageQuery struct {
	Limit   int
	Offset  int
	OrderBy string
	Order   string // asc|desc
	Filters map[string]string
}

func (q *PageQuery) Normalize() {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 200 {
		q.Limit = 200
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Order == "" {
		q.Order = "asc"
	}
	if q.Filters == nil {
		q.Filters = map[string]string{}
	}
}

type PageResult[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
	Limit int   `json:"limit"`
	Page  int   `json:"page"`
}

// ContextKey is the type used for context keys across the codebase.
type ContextKey string

const (
	ContextKeyRequestID ContextKey = "request_id"
	ContextKeyActor     ContextKey = "actor"
	ContextKeyTimeout   ContextKey = "io_timeout"
)

// WithRequestID attaches a request id to a context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, id)
}

// RequestIDFromContext returns the request id stored in ctx, or empty.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return v
	}
	return ""
}

// WithActor attaches the actor identity to a context.
func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, ContextKeyActor, actor)
}

// ActorFromContext returns the actor stored in ctx, or an anonymous actor.
func ActorFromContext(ctx context.Context) Actor {
	if v, ok := ctx.Value(ContextKeyActor).(Actor); ok {
		return v
	}
	return Actor{ID: "anonymous", Name: "匿名用户", Role: RoleViewer}
}

// Actor represents the calling user identity for audit and authorization.
type Actor struct {
	ID    string
	Name  string
	Email string
	Role  Role
}

// Role is the RBAC role attached to an actor.
type Role string

const (
	RoleViewer     Role = "viewer"
	RoleOperator   Role = "operator"
	RoleEngineer   Role = "engineer"
	RoleSupervisor Role = "supervisor"
	RoleAdmin      Role = "admin"
)

// CanManagePlans returns true when the role may edit maintenance plans.
func (r Role) CanManagePlans() bool {
	switch r {
	case RoleEngineer, RoleSupervisor, RoleAdmin:
		return true
	}
	return false
}

// CanExecute returns true when the role may submit maintenance execution records.
func (r Role) CanExecute() bool {
	switch r {
	case RoleOperator, RoleEngineer, RoleSupervisor, RoleAdmin:
		return true
	}
	return false
}

// CanRectify returns true when the role may perform rectification work.
func (r Role) CanRectify() bool {
	switch r {
	case RoleEngineer, RoleSupervisor, RoleAdmin:
		return true
	}
	return false
}

// CanRecover returns true when the role may confirm facility recovery.
func (r Role) CanRecover() bool {
	switch r {
	case RoleSupervisor, RoleAdmin:
		return true
	}
	return false
}

// CanAudit returns true when the role may read audit logs.
func (r Role) CanAudit() bool {
	switch r {
	case RoleSupervisor, RoleAdmin:
		return true
	}
	return false
}

// CanAdmin returns true when the role may manage configuration entities.
func (r Role) CanAdmin() bool {
	return r == RoleAdmin
}

// IsTimeZero reports whether a time is the zero value.
func IsTimeZero(t time.Time) bool { return t.IsZero() }
