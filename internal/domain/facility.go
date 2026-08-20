package domain

import (
	"context"
	"regexp"
	"time"
)

// FacilityStatus enumerates the lifecycle states a facility may be in.
type FacilityStatus string

const (
	// FacilityNormal means the facility is operating normally.
	FacilityNormal FacilityStatus = "normal"
	// FacilityPendingMaintenance means a maintenance task is currently scheduled.
	FacilityPendingMaintenance FacilityStatus = "pending_maintenance"
	// FacilityOverdue means a maintenance task was not completed before its due date.
	FacilityOverdue FacilityStatus = "overdue"
	// FacilityRestrictedUse means the facility is restricted because of an anomaly.
	FacilityRestrictedUse FacilityStatus = "restricted_use"
	// FacilityUnderRepair means the facility is currently being repaired.
	FacilityUnderRepair FacilityStatus = "under_repair"
	// FacilityRecovered means recovery work has finished and the facility awaits
	// confirmation to return to normal operation.
	FacilityRecovered FacilityStatus = "recovered"
)

// Criticality classifies how business-critical a facility is.
type Criticality string

const (
	CriticalityCritical  Criticality = "critical"
	CriticalityImportant Criticality = "important"
	CriticalityStandard  Criticality = "standard"
)

// Facility is the central aggregate tracked by the platform.
type Facility struct {
	ID                    string         `json:"id"`
	PlaceID               string         `json:"place_id"`
	Name                  string         `json:"name"`
	Code                  string         `json:"code"`
	Category              string         `json:"category"`
	ResponsiblePersonID   string         `json:"responsible_person_id"`
	Criticality           Criticality    `json:"criticality"`
	Status                FacilityStatus `json:"status"`
	Description           string         `json:"description"`
	AlternativeFacilityID string         `json:"alternative_facility_id,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	Version               int            `json:"version"`
}

// FacilityInput is the validated input for creating or updating a facility.
type FacilityInput struct {
	PlaceID               string      `json:"place_id"`
	Name                  string      `json:"name"`
	Code                  string      `json:"code"`
	Category              string      `json:"category"`
	ResponsiblePersonID   string      `json:"responsible_person_id"`
	Criticality           Criticality `json:"criticality"`
	Description           string      `json:"description"`
	AlternativeFacilityID string      `json:"alternative_facility_id,omitempty"`
}

var facilityCodePattern = regexp.MustCompile(`^[A-Z0-9-]{3,32}$`)

// Validate enforces field-level invariants.
func (i FacilityInput) Validate() error {
	if i.PlaceID == "" {
		return ErrInvalid("场所 ID 不能为空", "place_id", nil)
	}
	if len(i.Name) < 2 || len(i.Name) > 128 {
		return ErrInvalid("设施名称长度需在 2-128 之间", "name", nil)
	}
	if !facilityCodePattern.MatchString(i.Code) {
		return ErrInvalid("设施编码需为 3-32 位大写字母数字或短横线", "code", nil)
	}
	if len(i.Category) < 2 || len(i.Category) > 64 {
		return ErrInvalid("设施类别长度需在 2-64 之间", "category", nil)
	}
	if i.ResponsiblePersonID == "" {
		return ErrInvalid("责任人 ID 不能为空", "responsible_person_id", nil)
	}
	switch i.Criticality {
	case CriticalityCritical, CriticalityImportant, CriticalityStandard:
	default:
		return ErrInvalid("关键等级不在白名单内", "criticality", nil)
	}
	if len(i.Description) > 1024 {
		return ErrInvalid("描述长度不能超过 1024", "description", nil)
	}
	return nil
}

// FacilityStatusTransition captures a state transition request.
type FacilityStatusTransition struct {
	From   FacilityStatus
	To     FacilityStatus
	Reason string
}

// ValidateTransition enforces the state machine invariants of a facility.
// The invariant "逾期关键设施不能被标记为正常" is enforced here.
func (t FacilityStatusTransition) ValidateTransition(criticality Criticality) error {
	allowed := map[FacilityStatus]map[FacilityStatus]struct{}{
		FacilityNormal: {
			FacilityPendingMaintenance: {},
			FacilityRestrictedUse:      {},
			FacilityUnderRepair:        {},
		},
		FacilityPendingMaintenance: {
			FacilityNormal:        {},
			FacilityOverdue:       {},
			FacilityRestrictedUse: {},
			FacilityUnderRepair:   {},
		},
		FacilityOverdue: {
			FacilityNormal:        {},
			FacilityUnderRepair:   {},
			FacilityRestrictedUse: {},
			FacilityRecovered:     {},
		},
		FacilityRestrictedUse: {
			FacilityNormal:      {},
			FacilityUnderRepair: {},
			FacilityRecovered:   {},
		},
		FacilityUnderRepair: {
			FacilityRecovered: {},
		},
		FacilityRecovered: {
			FacilityNormal:             {},
			FacilityPendingMaintenance: {},
		},
	}
	if t.From == t.To {
		return ErrStateForbidden("状态未发生变化")
	}
	destinations, ok := allowed[t.From]
	if !ok {
		return ErrStateForbidden("非法的起始状态: " + string(t.From))
	}
	if _, ok := destinations[t.To]; !ok {
		return ErrStateForbidden("状态转换不允许: " + string(t.From) + " -> " + string(t.To))
	}
	// Critical invariant: an overdue critical facility cannot be marked as normal.
	if t.From == FacilityOverdue && t.To == FacilityNormal && criticality == CriticalityCritical {
		return ErrStateForbidden("逾期关键设施不能被直接标记为正常，需经维修和恢复流程")
	}
	// A restricted-use critical facility cannot jump back to normal directly.
	if t.From == FacilityRestrictedUse && t.To == FacilityNormal && criticality == CriticalityCritical {
		return ErrStateForbidden("限用关键设施不能被直接标记为正常，需经维修和恢复流程")
	}
	return nil
}

// MaintenanceCompletionStatus selects the facility status a facility should
// move to after a preventive maintenance execution is submitted.
//
// The ordinary maintenance flow is unaffected: a facility that was merely
// pending (or overdue but not business-critical) returns to Normal.
//
// An overdue CRITICAL facility is the exception. The state machine forbids
// overdue->normal for critical facilities because the repair and recovery
// steps must happen first — letting such a facility jump straight to Normal
// would make it appear healthy while those steps are silently skipped. To keep
// the steps on the critical path instead, the facility is routed into
// UnderRepair, where rectification, reinspection and recovery confirmation are
// tracked before it may return to Normal.
func MaintenanceCompletionStatus(f *Facility) FacilityStatus {
	if f == nil {
		return FacilityNormal
	}
	if f.Status == FacilityOverdue && f.Criticality == CriticalityCritical {
		return FacilityUnderRepair
	}
	return FacilityNormal
}

// MaintenanceCompletionTransition returns the status transition that Submit
// should apply after a maintenance execution is recorded, plus whether that
// transition is legal under the facility state machine. It pairs
// MaintenanceCompletionStatus with ValidateTransition so the application layer
// never bypasses the state machine: when ok is false the computed target would
// be an illegal move and the caller must leave the facility's status untouched.
func MaintenanceCompletionTransition(f *Facility) (FacilityStatusTransition, bool) {
	if f == nil {
		return FacilityStatusTransition{}, false
	}
	target := MaintenanceCompletionStatus(f)
	t := FacilityStatusTransition{
		From:   f.Status,
		To:     target,
		Reason: "maintenance_submitted",
	}
	if err := t.ValidateTransition(f.Criticality); err != nil {
		return t, false
	}
	return t, true
}

// FacilityRepository is the persistence contract.
type FacilityRepository interface {
	Create(ctx context.Context, f *Facility) error
	Update(ctx context.Context, f *Facility) error
	UpdateStatus(ctx context.Context, id string, status FacilityStatus, version int) error
	UpdateStatusChecked(ctx context.Context, id string, status FacilityStatus, version int) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*Facility, error)
	List(ctx context.Context, q PageQuery) (*PageResult[*Facility], error)
	ListByPlace(ctx context.Context, placeID string) ([]*Facility, error)
	ListByResponsiblePerson(ctx context.Context, personID string) ([]*Facility, error)
}
