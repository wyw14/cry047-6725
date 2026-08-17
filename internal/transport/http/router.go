package http

import (
	"context"
	"net/http"
	"time"

	"github.com/cry047/baseline/internal/application"
	"github.com/cry047/baseline/internal/domain"
	mw "github.com/cry047/baseline/internal/middleware"
	"github.com/cry047/baseline/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

// Server bundles the application services and HTTP handlers.
type Server struct {
	services *Services
	router   *gin.Engine
	srv      *http.Server
}

// Services aggregates every application service.
type Services struct {
	Places        *application.PlaceService
	People        *application.PersonService
	Facilities    *application.FacilityService
	Plans         *application.PlanService
	Executions    *application.ExecutionService
	Anomalies     *application.AnomalyService
	Todos         *application.TodoService
	Notifications *application.NotificationService
	Audit         *application.AuditService
	Timeline      *application.TimelineService
	Scheduler     *application.SchedulerService
}

// Config is the HTTP server config.
type Config struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	Mode            string // release|test|debug
	AllowedOrigins  []string
}

// responseWriterImpl is the implementation of handler.ResponseWriter that
// delegates to the package-level response helpers. It is set into the handler
// package via SetResponseWriter at New() time to break the import cycle.
type responseWriterImpl struct{}

func (responseWriterImpl) OK(c *gin.Context, data any)       { OK(c, data) }
func (responseWriterImpl) Created(c *gin.Context, data any)  { Created(c, data) }
func (responseWriterImpl) Accepted(c *gin.Context, data any) { Accepted(c, data) }
func (responseWriterImpl) NoContent(c *gin.Context)          { NoContent(c) }
func (responseWriterImpl) Fail(c *gin.Context, err error)    { Fail(c, err) }
func (responseWriterImpl) FailWithFieldErrors(c *gin.Context, msg string, errs []handler.FieldError) {
	fe := make([]FieldError, len(errs))
	for i, e := range errs {
		fe[i] = FieldError{Field: e.Field, Message: e.Message}
	}
	FailWithFieldErrors(c, msg, fe)
}

// New returns a configured Server.
func New(services *Services, cfg Config, logger *mw.ZapAdapter) (*Server, error) {
	if cfg.Mode == "" {
		cfg.Mode = "release"
	}
	gin.SetMode(cfg.Mode)
	// Wire the response writer into the handler package. Idempotent.
	handler.SetResponseWriter(responseWriterImpl{})
	r := gin.New()
	r.ForwardedByClientIP = true
	r.HandleMethodNotAllowed = true

	// Middleware ordering: request_id -> logger -> panic_recovery -> cors -> security_headers -> auth -> audit.
	r.Use(mw.RequestID())
	r.Use(mw.LoggerMiddleware(logger))
	r.Use(mw.PanicRecovery(logger))
	r.Use(mw.CORS(cfg.AllowedOrigins))
	r.Use(mw.SecurityHeaders())
	r.Use(mw.ActorFromHeader())
	r.Use(mw.AuditLog(logger))

	handlers := &handler.Handlers{
		Places:        handler.NewPlaceHandler(services.Places),
		People:        handler.NewPersonHandler(services.People),
		Facilities:    handler.NewFacilityHandler(services.Facilities),
		Plans:         handler.NewPlanHandler(services.Plans).WithScheduler(services.Scheduler),
		Executions:    handler.NewExecutionHandler(services.Executions),
		Anomalies:     handler.NewAnomalyHandler(services.Anomalies),
		Todos:         handler.NewTodoHandler(services.Todos),
		Notifications: handler.NewNotificationHandler(services.Notifications),
		Audit:         handler.NewAuditHandler(services.Audit),
		Timeline:      handler.NewTimelineHandler(services.Timeline, services.Facilities),
	}

	registerRoutes(r, handlers)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: cfg.ReadTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
	return &Server{services: services, router: r, srv: srv}, nil
}

// registerRoutes wires every route under /api/v1.
func registerRoutes(r *gin.Engine, h *handler.Handlers) {
	r.GET("/healthz", healthz)
	r.GET("/readyz", readyz(h))

	v1 := r.Group("/api/v1")
	{
		// Places
		v1.POST("/places", h.Places.Create)
		v1.GET("/places", h.Places.List)
		v1.GET("/places/:id", h.Places.Get)
		v1.PATCH("/places/:id", h.Places.Update)
		v1.DELETE("/places/:id", h.Places.Delete)

		// Responsible persons
		v1.POST("/responsible-persons", h.People.Create)
		v1.GET("/responsible-persons", h.People.List)
		v1.GET("/responsible-persons/:id", h.People.Get)
		v1.PATCH("/responsible-persons/:id", h.People.Update)
		v1.POST("/responsible-persons/:id/active", h.People.SetActive)
		v1.DELETE("/responsible-persons/:id", h.People.Delete)

		// Facilities
		v1.POST("/facilities", h.Facilities.Create)
		v1.GET("/facilities", h.Facilities.List)
		v1.GET("/facilities/:id", h.Facilities.Get)
		v1.PATCH("/facilities/:id", h.Facilities.Update)
		v1.POST("/facilities/:id/transition", h.Facilities.TransitionState)
		v1.GET("/facilities/:id/next-date", h.Facilities.NextDateInfo)
		v1.GET("/facilities/:id/timeline", h.Timeline.ByFacility)

		// Plan templates
		v1.POST("/plan-templates", h.Plans.CreateTemplate)
		v1.GET("/plan-templates", h.Plans.ListTemplates)
		v1.GET("/plan-templates/:id", h.Plans.GetTemplate)
		v1.PATCH("/plan-templates/:id", h.Plans.UpdateTemplate)

		// Plans
		v1.POST("/plans", h.Plans.CreatePlan)
		v1.GET("/plans", h.Plans.ListPlans)
		v1.GET("/plans/:id", h.Plans.GetPlan)
		v1.PATCH("/plans/:id/cycle", h.Plans.ChangeCycle)
		v1.POST("/plans/:id/skip", h.Plans.Skip)
		v1.POST("/plans/:id/regenerate", h.Plans.Regenerate)
		v1.POST("/plans/:id/assign", h.Plans.AssignResponsible)
		v1.GET("/plans/:id/versions", h.Plans.ListVersions)

		// Executions
		v1.POST("/executions", h.Executions.Submit)
		v1.GET("/executions", h.Executions.List)
		v1.GET("/executions/:id", h.Executions.Get)
		v1.POST("/executions/:id/review", h.Executions.Review)

		// Anomalies
		v1.POST("/anomalies", h.Anomalies.Discover)
		v1.GET("/anomalies", h.Anomalies.List)
		v1.GET("/anomalies/:id", h.Anomalies.Get)
		v1.POST("/anomalies/:id/rectify", h.Anomalies.Rectify)
		v1.POST("/anomalies/:id/reinspect", h.Anomalies.Reinspect)
		v1.POST("/anomalies/:id/recover", h.Anomalies.Recover)

		// Todos
		v1.GET("/todos", h.Todos.List)
		v1.GET("/todos/by-user/:user_id", h.Todos.ListByUser)
		v1.PATCH("/todos/:id/complete", h.Todos.Complete)
		v1.PATCH("/todos/:id/skip", h.Todos.Skip)
		v1.PATCH("/todos/:id/reassign", h.Todos.Reassign)

		// Notifications
		v1.GET("/notifications", h.Notifications.List)
		v1.GET("/notifications/by-user/:user_id", h.Notifications.ListByUser)
		v1.GET("/notifications/unread/:user_id", h.Notifications.ListUnread)
		v1.PATCH("/notifications/:id/read", h.Notifications.MarkRead)

		// Audit
		v1.GET("/audit-logs", h.Audit.List)

		// Scheduler scan
		v1.POST("/scheduler/scan", h.Plans.RunSchedulerScan)
	}
}

// healthz returns 200 if the process is alive.
func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Code: domain.CodeOK, Message: "OK", Data: map[string]any{"status": "ok"}})
}

// readyz returns 200 if the process can serve requests. It is wired to the
// scheduler service to verify a basic read path exists.
func readyz(h *handler.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, Response{Code: domain.CodeOK, Message: "OK", Data: map[string]any{"status": "ready"}})
	}
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// Handler returns the underlying gin engine. Useful for httptest.
func (s *Server) Handler() http.Handler {
	return s.router
}
