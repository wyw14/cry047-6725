package application

import (
	"context"
	"time"

	"github.com/cry047/baseline/internal/domain"
	"github.com/google/uuid"
)

// recordAudit builds and persists an AuditLog entry.
func recordAudit(ctx context.Context, ports *domain.Ports, in domain.AuditLogInput) error {
	var before, after []byte
	if in.BeforeState != nil {
		before, _ = jsonMarshal(in.BeforeState)
	}
	if in.AfterState != nil {
		after, _ = jsonMarshal(in.AfterState)
	}
	l := &domain.AuditLog{
		ID:          uuid.NewString(),
		Action:      in.Action,
		EntityType:  in.EntityType,
		EntityID:    in.EntityID,
		ActorID:     in.Actor.ID,
		ActorName:   in.Actor.Name,
		BeforeState: before,
		AfterState:  after,
		RequestID:   in.RequestID,
		Reason:      in.Reason,
		CreatedAt:   time.Now().UTC(),
	}
	return ports.AuditLogs.Create(ctx, l)
}
