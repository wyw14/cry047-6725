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

// Clone returns a deep copy of the inspection value. Because NumericValue is a
// pointer, a plain struct copy would alias the same float64 allocation; Clone
// re-points it at a freshly allocated value so the copy and the original share
// no mutable state.
func (v InspectionValue) Clone() InspectionValue {
	cp := v
	if v.NumericValue != nil {
		n := *v.NumericValue
		cp.NumericValue = &n
	}
	return cp
}

// ConsumableConsumption records how much of a consumable was actually consumed.
type ConsumableConsumption struct {
	Code     string `json:"code"`
	Quantity int    `json:"quantity"`
}

// Clone returns a deep copy of the consumable consumption record. It has no
// pointer or slice fields today, but the method is provided so evidence copy
// helpers can treat all value types uniformly and remain correct if fields are
// added later.
func (c ConsumableConsumption) Clone() ConsumableConsumption { return c }

// PhotoAttachment is a verified photo captured during execution.
type PhotoAttachment struct {
	Path     string `json:"path"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}

// Clone returns a deep copy of the photo attachment. See ConsumableConsumption.
func (p PhotoAttachment) Clone() PhotoAttachment { return p }

// CloneInspectionValues returns a fully independent copy of the slice: a new
// backing array whose elements are each deep-cloned (including their
// NumericValue pointer). The nil-vs-empty distinction is preserved so callers
// that rely on JSON null/[] semantics are unaffected.
func CloneInspectionValues(in []InspectionValue) []InspectionValue {
	if in == nil {
		return nil
	}
	out := make([]InspectionValue, len(in))
	for i := range in {
		out[i] = in[i].Clone()
	}
	return out
}

// ClonePhotoAttachments returns a fully independent copy of the slice.
func ClonePhotoAttachments(in []PhotoAttachment) []PhotoAttachment {
	if in == nil {
		return nil
	}
	out := make([]PhotoAttachment, len(in))
	for i := range in {
		out[i] = in[i].Clone()
	}
	return out
}

// CloneConsumablesConsumed returns a fully independent copy of the slice.
func CloneConsumablesConsumed(in []ConsumableConsumption) []ConsumableConsumption {
	if in == nil {
		return nil
	}
	out := make([]ConsumableConsumption, len(in))
	for i := range in {
		out[i] = in[i].Clone()
	}
	return out
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

// Snapshot returns a deep value copy suitable for passing across service
// boundaries. The returned record shares no backing storage — neither slice
// headers nor pointer fields like InspectionValue.NumericValue — with the
// receiver, so callers may freely mutate the returned record (or any element
// within it) without affecting persisted or in-flight state, and vice versa.
// This is what guarantees that editing the request data or the response object
// after a submission cannot reach back into the saved evidence.
func (e *Execution) Snapshot() *Execution {
	return e.Clone()
}

// Clone returns a deep, independent copy of the execution record. Every slice
// (InspectionValues, Photos, ConsumablesConsumed) is re-allocated with freshly
// cloned elements, and every pointer field reachable from those elements is
// re-pointed at a new allocation. The copy and the original therefore share no
// mutable state.
//
// A nil receiver yields a nil result.
func (e *Execution) Clone() *Execution {
	if e == nil {
		return nil
	}
	cp := *e
	cp.InspectionValues = CloneInspectionValues(e.InspectionValues)
	cp.Photos = ClonePhotoAttachments(e.Photos)
	cp.ConsumablesConsumed = CloneConsumablesConsumed(e.ConsumablesConsumed)
	return &cp
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
