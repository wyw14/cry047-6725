package domain

import (
	"context"
	"fmt"
	"time"
)

// MaintenanceTodoKey derives the idempotency key for the maintenance todo that
// the scheduler auto-generates for one overdue occurrence of a plan.
//
// The key is (plan id, occurrence due date): the due date is what distinguishes
// one occurrence from the next, so each overdue occurrence gets its own todo,
// while repeated scans of the same occurrence (same due date) resolve to the
// same key and are idempotent. The date is rendered in UTC at day granularity so
// the key is stable regardless of the process timezone — two scans that observe
// the same occurrence must always agree on its key.
func MaintenanceTodoKey(plan *MaintenancePlan) string {
	return fmt.Sprintf("auto-todo-%s-%s", plan.ID, plan.OccurrenceDate().UTC().Format(time.DateOnly))
}

// TodoType enumerates the kind of work tracked by a todo.
type TodoType string

const (
	TodoTypeMaintenance   TodoType = "maintenance"
	TodoTypeRectification TodoType = "rectification"
	TodoTypeReinspection  TodoType = "reinspection"
	TodoTypeRecovery      TodoType = "recovery"
	TodoTypeReview        TodoType = "review"
)

// TodoStatus enumerates the lifecycle of a todo.
type TodoStatus string

const (
	TodoOpen      TodoStatus = "open"
	TodoAssigned  TodoStatus = "assigned"
	TodoCompleted TodoStatus = "completed"
	TodoSkipped   TodoStatus = "skipped"
	TodoOverdue   TodoStatus = "overdue"
)

// Todo is a work item assigned to a user, generated from plans and anomalies.
type Todo struct {
	ID             string     `json:"id"`
	FacilityID     string     `json:"facility_id"`
	PlanID         string     `json:"plan_id,omitempty"`
	AnomalyID      string     `json:"anomaly_id,omitempty"`
	AssignedTo     string     `json:"assigned_to"`
	Type           TodoType   `json:"type"`
	DueDate        time.Time  `json:"due_date"`
	Status         TodoStatus `json:"status"`
	Title          string     `json:"title"`
	IdempotencyKey string     `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    time.Time  `json:"completed_at,omitempty"`
	Version        int        `json:"version"`
}

// TodoInput is the validated input for creating a todo.
type TodoInput struct {
	FacilityID     string    `json:"facility_id"`
	PlanID         string    `json:"plan_id,omitempty"`
	AnomalyID      string    `json:"anomaly_id,omitempty"`
	AssignedTo     string    `json:"assigned_to"`
	Type           TodoType  `json:"type"`
	DueDate        time.Time `json:"due_date"`
	Title          string    `json:"title"`
	IdempotencyKey string    `json:"idempotency_key"`
}

// Validate enforces todo input invariants.
func (i TodoInput) Validate() error {
	if i.FacilityID == "" {
		return ErrInvalid("设施 ID 不能为空", "facility_id", nil)
	}
	if i.AssignedTo == "" {
		return ErrInvalid("指派人不能为空", "assigned_to", nil)
	}
	if i.DueDate.IsZero() {
		return ErrInvalid("截止日期不能为空", "due_date", nil)
	}
	switch i.Type {
	case TodoTypeMaintenance, TodoTypeRectification, TodoTypeReinspection, TodoTypeRecovery, TodoTypeReview:
	default:
		return ErrInvalid("待办类型不在白名单内", "type", nil)
	}
	if len(i.Title) < 2 || len(i.Title) > 256 {
		return ErrInvalid("待办标题长度需在 2-256 之间", "title", nil)
	}
	if i.IdempotencyKey == "" {
		return ErrInvalid("幂等键不能为空", "idempotency_key", nil)
	}
	return nil
}

// TodoRepository is the persistence contract.
type TodoRepository interface {
	Create(ctx context.Context, t *Todo) error
	Update(ctx context.Context, t *Todo) error
	Get(ctx context.Context, id string) (*Todo, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*Todo, error)
	List(ctx context.Context, q PageQuery) (*PageResult[*Todo], error)
	ListByUser(ctx context.Context, userID string, q PageQuery) (*PageResult[*Todo], error)
	ListOverdue(ctx context.Context, before time.Time) ([]*Todo, error)
}
