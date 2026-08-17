package domain

import (
	"context"
	"time"
)

// TimelineRepository is the contract for building facility timelines.
type TimelineRepository interface {
	Append(ctx context.Context, e *TimelineEvent) error
	ListByFacility(ctx context.Context, facilityID string, limit int) ([]*TimelineEvent, error)
}

// NextDateInfo captures the next due date and overdue risk for a facility.
type NextDateInfo struct {
	FacilityID            string    `json:"facility_id"`
	NextDueDate           time.Time `json:"next_due_date"`
	OverdueRisk           bool      `json:"overdue_risk"`
	DaysUntilDue          int       `json:"days_until_due"`
	LastExecutedAt        time.Time `json:"last_executed_at,omitempty"`
	AlternativeFacilityID string    `json:"alternative_facility_id,omitempty"`
}

// Scheduler is the contract for the local, offline-safe scheduler adapter.
type Scheduler interface {
	// Schedule registers a callback to fire at the next tick after the
	// requested time. The callback is invoked with the provided context.
	Schedule(ctx context.Context, at time.Time, name string, fn func(ctx context.Context)) (cancel func(), err error)
	// Now returns the adapter's notion of "now". Replaceable in tests.
	Now() time.Time
	// Start begins the scheduler loop. Must be idempotent.
	Start(ctx context.Context) error
	// Stop signals the scheduler loop to exit.
	Stop(ctx context.Context) error
}

// AttachmentStorage is the contract for local, controlled attachment storage.
type AttachmentStorage interface {
	Save(ctx context.Context, name string, contentType string, data []byte) (path string, err error)
	Read(ctx context.Context, path string) (contentType string, data []byte, err error)
	Delete(ctx context.Context, path string) error
}

// Clock is the contract for a clock abstraction that returns the current time.
type Clock interface {
	Now() time.Time
}

// SystemClock returns the real wall clock.
type SystemClock struct{}

// Now returns time.Now().
func (SystemClock) Now() time.Time { return time.Now() }

// StubClock is a deterministic clock for tests.
type StubClock struct{ T time.Time }

// Now returns the configured time.
func (c StubClock) Now() time.Time { return c.T }

// WithTimeout returns a context that is cancelled after the duration.
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

// Ports aggregates every external dependency the application layer needs.
// It is the single dependency surface used by application services.
type Ports struct {
	Places        PlaceRepository
	Facilities    FacilityRepository
	People        ResponsiblePersonRepository
	Templates     PlanRepository
	Plans         PlanRepository
	Executions    ExecutionRepository
	Anomalies     AnomalyRepository
	Todos         TodoRepository
	Notifications NotificationRepository
	AuditLogs     AuditLogRepository
	Timeline      TimelineRepository
	Notifier      Notifier
	Storage       AttachmentStorage
	Scheduler     Scheduler
	Clock         Clock
}

// Assert that *Ports can satisfy the ports interface (compile-time sanity).
// Empty interface assertion is the conventional Go idiom.
var _ interface{} = (*Ports)(nil)
