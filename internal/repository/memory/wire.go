package memory

import "github.com/cry047/baseline/internal/domain"

// Ports returns a fully wired domain.Ports using in-memory repositories.
// Useful for tests, demos and the offline self-contained test suite.
func NewPorts(s *Store) domain.Ports {
	if s == nil {
		s = NewStore()
	}
	return domain.Ports{
		Places:        NewPlaceRepository(s),
		Facilities:    NewFacilityRepository(s),
		People:        NewResponsiblePersonRepository(s),
		Templates:     NewPlanRepository(s),
		Plans:         NewPlanRepository(s),
		Executions:    NewExecutionRepository(s),
		Anomalies:     NewAnomalyRepository(s),
		Todos:         NewTodoRepository(s),
		Notifications: NewNotificationRepository(s),
		AuditLogs:     NewAuditLogRepository(s),
		Timeline:      NewTimelineRepository(s),
	}
}
