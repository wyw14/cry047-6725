package domain

import (
	"context"
	"time"
)

// AnomalySeverity classifies the impact of a discovered anomaly.
type AnomalySeverity string

const (
	AnomalySeverityLow      AnomalySeverity = "low"
	AnomalySeverityMedium   AnomalySeverity = "medium"
	AnomalySeverityHigh     AnomalySeverity = "high"
	AnomalySeverityCritical AnomalySeverity = "critical"
)

// AnomalyStatus enumerates the lifecycle of an anomaly.
type AnomalyStatus string

const (
	AnomalyOpen           AnomalyStatus = "open"
	AnomalyRectifying     AnomalyStatus = "rectifying"
	AnomalyReinspecting   AnomalyStatus = "reinspecting"
	AnomalyRecovered      AnomalyStatus = "recovered"
	AnomalyClosedNoAction AnomalyStatus = "closed_no_action"
)

// ReinspectionStatus returns the next anomaly status after a reinspection.
//
// A passing reinspection advances the anomaly to recovered (awaiting recovery
// confirmation); a failing reinspection reopens the anomaly so another
// rectification cycle can run. The issue is therefore kept open until a
// follow-up inspection actually passes.
//
// The associated facility's visible status is advanced to recovered only when
// recovery is later confirmed via Recover — never merely because a reinspection
// passed — so the status surfaced across the ledger, detail page and timeline
// stays consistent everywhere.
func ReinspectionStatus(pass bool) AnomalyStatus {
	if pass {
		return AnomalyRecovered
	}
	return AnomalyOpen
}

// AnomalyStatusTransition captures an anomaly state transition request and
// enforces the anomaly lifecycle invariants, mirroring the facility state
// machine. Centralising the rules here keeps the application service honest:
// the recovered status is reachable ONLY through a passing reinspection, and
// a failing reinspection always reopens the anomaly.
type AnomalyStatusTransition struct {
	From AnomalyStatus
	To   AnomalyStatus
}

// allowedAnomalyTransitions is the anomaly state machine.
//
//	 open          -> reinspecting            (rectify)
//	 reinspecting  -> recovered               (reinspect pass)
//	 reinspecting  -> open                    (reinspect fail: reopen for another cycle)
//	 recovered     -> recovered               (recovery confirmation; idempotent self-transition
//	                                          that records the confirmer without advancing status)
//
// "rectifying" and "closed_no_action" are terminal/reserved starting points
// with no outgoing transitions declared here.
var allowedAnomalyTransitions = map[AnomalyStatus]map[AnomalyStatus]struct{}{
	AnomalyOpen: {
		AnomalyReinspecting: {},
	},
	AnomalyReinspecting: {
		AnomalyRecovered: {},
		AnomalyOpen:      {},
	},
	AnomalyRecovered: {
		// Recovery confirmation is an idempotent self-transition, so a same-status
		// move is explicitly permitted here (unlike the facility machine).
		AnomalyRecovered: {},
	},
}

// Validate enforces the anomaly state machine invariants.
func (t AnomalyStatusTransition) Validate() error {
	destinations, ok := allowedAnomalyTransitions[t.From]
	if !ok {
		return ErrStateForbidden("非法的异常起始状态: " + string(t.From))
	}
	if _, ok := destinations[t.To]; !ok {
		return ErrStateForbidden("异常状态转换不允许: " + string(t.From) + " -> " + string(t.To))
	}
	return nil
}

// Anomaly is a discovered abnormal condition of a facility.
type Anomaly struct {
	ID                   string          `json:"id"`
	FacilityID           string          `json:"facility_id"`
	ExecutionID          string          `json:"execution_id,omitempty"`
	DiscoveredBy         string          `json:"discovered_by"`
	DiscoveredAt         time.Time       `json:"discovered_at"`
	Description          string          `json:"description"`
	Severity             AnomalySeverity `json:"severity"`
	Status               AnomalyStatus   `json:"status"`
	RectificationMeasure string          `json:"rectification_measure,omitempty"`
	RectifiedBy          string          `json:"rectified_by,omitempty"`
	RectifiedAt          time.Time       `json:"rectified_at,omitempty"`
	ReinspectionResult   string          `json:"reinspection_result,omitempty"`
	ReinspectedBy        string          `json:"reinspected_by,omitempty"`
	ReinspectedAt        time.Time       `json:"reinspected_at,omitempty"`
	RecoveredBy          string          `json:"recovered_by,omitempty"`
	RecoveredAt          time.Time       `json:"recovered_at,omitempty"`
	IdempotencyKey       string          `json:"idempotency_key,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	Version              int             `json:"version"`
}

// AnomalyInput is the validated input for creating an anomaly.
type AnomalyInput struct {
	FacilityID     string          `json:"facility_id"`
	ExecutionID    string          `json:"execution_id,omitempty"`
	DiscoveredBy   string          `json:"discovered_by"`
	DiscoveredAt   time.Time       `json:"discovered_at"`
	Description    string          `json:"description"`
	Severity       AnomalySeverity `json:"severity"`
	IdempotencyKey string          `json:"idempotency_key"`
}

// Validate enforces anomaly input invariants.
func (i AnomalyInput) Validate() error {
	if i.FacilityID == "" {
		return ErrInvalid("设施 ID 不能为空", "facility_id", nil)
	}
	if i.DiscoveredBy == "" {
		return ErrInvalid("发现人不能为空", "discovered_by", nil)
	}
	if i.DiscoveredAt.IsZero() {
		return ErrInvalid("发现时间不能为空", "discovered_at", nil)
	}
	if len(i.Description) < 2 || len(i.Description) > 2048 {
		return ErrInvalid("异常描述长度需在 2-2048 之间", "description", nil)
	}
	switch i.Severity {
	case AnomalySeverityLow, AnomalySeverityMedium, AnomalySeverityHigh, AnomalySeverityCritical:
	default:
		return ErrInvalid("严重度不在白名单内", "severity", nil)
	}
	if i.IdempotencyKey == "" {
		return ErrInvalid("幂等键不能为空", "idempotency_key", nil)
	}
	return nil
}

// RectificationInput is the validated input for rectification action.
type RectificationInput struct {
	AnomalyID   string `json:"anomaly_id"`
	Measure     string `json:"measure"`
	RectifiedBy string `json:"rectified_by"`
}

// Validate enforces rectification invariants.
func (i RectificationInput) Validate() error {
	if i.AnomalyID == "" {
		return ErrInvalid("异常 ID 不能为空", "anomaly_id", nil)
	}
	if len(i.Measure) < 2 || len(i.Measure) > 2048 {
		return ErrInvalid("整改措施长度需在 2-2048 之间", "measure", nil)
	}
	if i.RectifiedBy == "" {
		return ErrInvalid("整改人不能为空", "rectified_by", nil)
	}
	return nil
}

// ReinspectionInput is the validated input for the re-inspection step.
type ReinspectionInput struct {
	AnomalyID     string    `json:"anomaly_id"`
	Result        string    `json:"result"`
	ReinspectedBy string    `json:"reinspected_by"`
	ReinspectedAt time.Time `json:"reinspected_at"`
}

// Validate enforces reinspection invariants.
func (i ReinspectionInput) Validate() error {
	if i.AnomalyID == "" {
		return ErrInvalid("异常 ID 不能为空", "anomaly_id", nil)
	}
	if len(i.Result) < 2 || len(i.Result) > 2048 {
		return ErrInvalid("复检结果长度需在 2-2048 之间", "result", nil)
	}
	if i.ReinspectedBy == "" {
		return ErrInvalid("复检人不能为空", "reinspected_by", nil)
	}
	if i.ReinspectedAt.IsZero() {
		return ErrInvalid("复检时间不能为空", "reinspected_at", nil)
	}
	return nil
}

// RecoverInput is the validated input for the recovery confirmation step.
type RecoverInput struct {
	AnomalyID   string `json:"anomaly_id"`
	RecoveredBy string `json:"recovered_by"`
}

// Validate enforces recovery confirmation invariants.
func (i RecoverInput) Validate() error {
	if i.AnomalyID == "" {
		return ErrInvalid("异常 ID 不能为空", "anomaly_id", nil)
	}
	if i.RecoveredBy == "" {
		return ErrInvalid("恢复确认人不能为空", "recovered_by", nil)
	}
	return nil
}

// AnomalyRepository is the persistence contract.
type AnomalyRepository interface {
	Create(ctx context.Context, a *Anomaly) error
	Update(ctx context.Context, a *Anomaly) error
	Get(ctx context.Context, id string) (*Anomaly, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*Anomaly, error)
	List(ctx context.Context, q PageQuery) (*PageResult[*Anomaly], error)
	ListByFacility(ctx context.Context, facilityID string, limit int) ([]*Anomaly, error)
	ListOpen(ctx context.Context) ([]*Anomaly, error)
}
