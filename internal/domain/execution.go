package domain

import (
	"context"
	"time"
)

// ExecutionStatus enumerates the lifecycle of a maintenance execution record.
type ExecutionStatus string

const (
	ExecutionDraft     ExecutionStatus = "draft"
	ExecutionSubmitted ExecutionStatus = "submitted"
	ExecutionReviewed  ExecutionStatus = "reviewed"
	ExecutionRejected  ExecutionStatus = "rejected"
)

// InspectionValue is one observation collected during execution.
type InspectionValue struct {
	ItemCode     string   `json:"item_code"`
	Value        string   `json:"value"`
	NumericValue *float64 `json:"numeric_value,omitempty"`
	Pass         bool     `json:"pass"`
	Note         string   `json:"note,omitempty"`
}

// ConsumableConsumption records how much of a consumable was actually consumed.
type ConsumableConsumption struct {
	Code     string `json:"code"`
	Quantity int    `json:"quantity"`
}

// PhotoAttachment is a verified photo captured during execution.
type PhotoAttachment struct {
	Path     string `json:"path"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}

// Execution is a single maintenance execution record.
type Execution struct {
	ID                  string                  `json:"id"`
	PlanID              string                  `json:"plan_id"`
	FacilityID          string                  `json:"facility_id"`
	ExecutedBy          string                  `json:"executed_by"`
	ExecutedAt          time.Time               `json:"executed_at"`
	InspectionValues    []InspectionValue       `json:"inspection_values"`
	Photos              []PhotoAttachment       `json:"photos"`
	ConsumablesConsumed []ConsumableConsumption `json:"consumables_consumed"`
	ReviewComment       string                  `json:"review_comment,omitempty"`
	ReviewedBy          string                  `json:"reviewed_by,omitempty"`
	ReviewedAt          time.Time               `json:"reviewed_at,omitempty"`
	Status              ExecutionStatus         `json:"status"`
	IdempotencyKey      string                  `json:"idempotency_key,omitempty"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
	Version             int                     `json:"version"`
}

// Snapshot returns a detached copy suitable for persistence and responses.
func (e *Execution) Snapshot() *Execution {
	if e == nil {
		return nil
	}
	cp := *e
	cp.InspectionValues = CloneInspectionValues(e.InspectionValues)
	cp.Photos = ClonePhotoAttachments(e.Photos)
	cp.ConsumablesConsumed = CloneConsumableConsumptions(e.ConsumablesConsumed)
	return &cp
}

// CloneInspectionValues copies observations, including optional numeric values.
func CloneInspectionValues(values []InspectionValue) []InspectionValue {
	if values == nil {
		return nil
	}
	out := make([]InspectionValue, len(values))
	copy(out, values)
	for i := range out {
		if values[i].NumericValue != nil {
			numeric := *values[i].NumericValue
			out[i].NumericValue = &numeric
		}
	}
	return out
}

// ClonePhotoAttachments copies the immutable photo metadata slice.
func ClonePhotoAttachments(photos []PhotoAttachment) []PhotoAttachment {
	return append([]PhotoAttachment(nil), photos...)
}

// CloneConsumableConsumptions copies the submitted consumable evidence.
func CloneConsumableConsumptions(items []ConsumableConsumption) []ConsumableConsumption {
	return append([]ConsumableConsumption(nil), items...)
}

// ExecutionInput is the validated input for submitting a maintenance record.
type ExecutionInput struct {
	PlanID              string                  `json:"plan_id"`
	FacilityID          string                  `json:"facility_id"`
	ExecutedBy          string                  `json:"executed_by"`
	ExecutedAt          time.Time               `json:"executed_at"`
	InspectionValues    []InspectionValue       `json:"inspection_values"`
	Photos              []PhotoAttachment       `json:"photos"`
	ConsumablesConsumed []ConsumableConsumption `json:"consumables_consumed"`
	IdempotencyKey      string                  `json:"idempotency_key"`
}

// Validate enforces execution invariants.
func (i ExecutionInput) Validate() error {
	if i.PlanID == "" {
		return ErrInvalid("计划 ID 不能为空", "plan_id", nil)
	}
	if i.FacilityID == "" {
		return ErrInvalid("设施 ID 不能为空", "facility_id", nil)
	}
	if i.ExecutedBy == "" {
		return ErrInvalid("执行人不能为空", "executed_by", nil)
	}
	if i.ExecutedAt.IsZero() {
		return ErrInvalid("执行时间不能为空", "executed_at", nil)
	}
	if i.IdempotencyKey == "" {
		return ErrInvalid("幂等键不能为空", "idempotency_key", nil)
	}
	codes := map[string]bool{}
	for idx, v := range i.InspectionValues {
		if v.ItemCode == "" {
			return ErrInvalid("检查值 item_code 不能为空", "inspection_values["+itoa(idx)+"].item_code", nil)
		}
		if codes[v.ItemCode] {
			return ErrInvalid("检查值 item_code 重复: "+v.ItemCode, "inspection_values["+itoa(idx)+"].item_code", nil)
		}
		codes[v.ItemCode] = true
	}
	for idx, p := range i.Photos {
		if p.Path == "" {
			return ErrInvalid("照片 path 不能为空", "photos["+itoa(idx)+"].path", nil)
		}
		if p.Size <= 0 {
			return ErrInvalid("照片大小必须为正", "photos["+itoa(idx)+"].size", nil)
		}
	}
	return nil
}

// ExecutionRepository is the persistence contract.
type ExecutionRepository interface {
	Create(ctx context.Context, e *Execution) error
	Update(ctx context.Context, e *Execution) error
	Get(ctx context.Context, id string) (*Execution, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*Execution, error)
	List(ctx context.Context, q PageQuery) (*PageResult[*Execution], error)
	ListByFacility(ctx context.Context, facilityID string, limit int) ([]*Execution, error)
	ListByPlan(ctx context.Context, planID string) ([]*Execution, error)
}
