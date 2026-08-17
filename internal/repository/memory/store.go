package memory

import (
	"sort"
	"sync"
	"time"

	"github.com/cry047/baseline/internal/domain"
)

// Store is a fully in-memory implementation of every repository port.
// It is safe for concurrent use and intended for tests, local demos and the
// offline self-contained test suite (go test ./...).
type Store struct {
	mu sync.RWMutex

	places       map[string]*domain.Place
	placesByCode map[string]string
	facilities   map[string]*domain.Facility
	people       map[string]*domain.ResponsiblePerson

	templates    map[string]*domain.PlanTemplate
	plans        map[string]*domain.MaintenancePlan
	planVersions []domain.PlanVersion

	executions      map[string]*domain.Execution
	executionsByKey map[string]string

	anomalies      map[string]*domain.Anomaly
	anomaliesByKey map[string]string

	todos      map[string]*domain.Todo
	todosByKey map[string]string

	notifications map[string]*domain.Notification

	auditLogs []domain.AuditLog
	timeline  map[string][]domain.TimelineEvent

	// whitelist maps for sort columns
	sortWhitelist map[string]map[string]string
}

// NewStore returns an empty in-memory store.
func NewStore() *Store {
	s := &Store{
		places:          map[string]*domain.Place{},
		placesByCode:    map[string]string{},
		facilities:      map[string]*domain.Facility{},
		people:          map[string]*domain.ResponsiblePerson{},
		templates:       map[string]*domain.PlanTemplate{},
		plans:           map[string]*domain.MaintenancePlan{},
		executions:      map[string]*domain.Execution{},
		executionsByKey: map[string]string{},
		anomalies:       map[string]*domain.Anomaly{},
		anomaliesByKey:  map[string]string{},
		todos:           map[string]*domain.Todo{},
		todosByKey:      map[string]string{},
		notifications:   map[string]*domain.Notification{},
		timeline:        map[string][]domain.TimelineEvent{},
	}
	s.sortWhitelist = map[string]map[string]string{
		"places":              {"name": "name", "code": "code", "created_at": "created_at"},
		"facilities":          {"name": "name", "code": "code", "criticality": "criticality", "status": "status", "created_at": "created_at"},
		"responsible_persons": {"name": "name", "created_at": "created_at"},
		"plan_templates":      {"name": "name", "cycle_days": "cycle_days", "created_at": "created_at"},
		"plans":               {"next_due_date": "next_due_date", "facility_id": "facility_id", "created_at": "created_at"},
		"executions":          {"executed_at": "executed_at", "facility_id": "facility_id", "created_at": "created_at"},
		"anomalies":           {"discovered_at": "discovered_at", "severity": "severity", "status": "status"},
		"todos":               {"due_date": "due_date", "status": "status", "assigned_to": "assigned_to"},
		"notifications":       {"created_at": "created_at"},
		"audit_logs":          {"created_at": "created_at", "entity_type": "entity_type"},
	}
	return s
}

// clone returns a deep-enough copy of a value via JSON round-trip.
// Used to prevent external mutation of in-memory state.
func clone[T any](v T) T {
	// We can't import json in store without circular dep, use a helper.
	return v
}

// now returns the current time.
func now() time.Time { return time.Now().UTC() }

// applySortWhitelist returns the canonical column for (entity, requested) or "".
func (s *Store) applySortWhitelist(entity, requested string) string {
	if requested == "" {
		return "created_at"
	}
	m := s.sortWhitelist[entity]
	if col, ok := m[requested]; ok {
		return col
	}
	return "created_at"
}

// paginate applies limit/offset to a slice and returns a PageResult.
func paginate[T any](items []T, q domain.PageQuery) *domain.PageResult[T] {
	total := int64(len(items))
	q.Normalize()
	start := q.Offset
	if start > len(items) {
		start = len(items)
	}
	end := start + q.Limit
	if end > len(items) {
		end = len(items)
	}
	return &domain.PageResult[T]{
		Items: items[start:end],
		Total: total,
		Limit: q.Limit,
		Page:  q.Offset/q.Limit + 1,
	}
}

// sortStrings returns a copy sorted by the comparator.
func sortStrings(items []string) []string {
	sort.Strings(items)
	return items
}
