package domain

import (
	"context"
	"encoding/json"
	"time"
)

// AuditAction enumerates the verbs recorded in the audit log.
type AuditAction string

const (
	AuditCreate      AuditAction = "create"
	AuditUpdate      AuditAction = "update"
	AuditDelete      AuditAction = "delete"
	AuditStateChange AuditAction = "state_change"
	AuditImport      AuditAction = "import"
	AuditExport      AuditAction = "export"
	AuditRecover     AuditAction = "recover"
	AuditSkip        AuditAction = "skip"
	AuditRegenerate  AuditAction = "regenerate"
)

// AuditLog captures one auditable mutation in the system.
type AuditLog struct {
	ID          string          `json:"id"`
	Action      AuditAction     `json:"action"`
	EntityType  string          `json:"entity_type"`
	EntityID    string          `json:"entity_id"`
	ActorID     string          `json:"actor_id"`
	ActorName   string          `json:"actor_name"`
	BeforeState json.RawMessage `json:"before_state,omitempty"`
	AfterState  json.RawMessage `json:"after_state,omitempty"`
	RequestID   string          `json:"request_id"`
	Reason      string          `json:"reason,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// AuditLogInput is the validated input for creating an audit log entry.
type AuditLogInput struct {
	Action      AuditAction
	EntityType  string
	EntityID    string
	Actor       Actor
	BeforeState any
	AfterState  any
	Reason      string
	RequestID   string
}

// AuditLogRepository is the persistence contract.
type AuditLogRepository interface {
	Create(ctx context.Context, l *AuditLog) error
	List(ctx context.Context, q PageQuery) (*PageResult[*AuditLog], error)
	ListByEntity(ctx context.Context, entityType, entityID string) ([]*AuditLog, error)
}

// TimelineEvent is a unified timeline entry for a facility.
type TimelineEvent struct {
	ID         string          `json:"id"`
	FacilityID string          `json:"facility_id"`
	OccurredAt time.Time       `json:"occurred_at"`
	EventType  string          `json:"event_type"`
	Title      string          `json:"title"`
	Actor      string          `json:"actor"`
	Detail     json.RawMessage `json:"detail,omitempty"`
}
