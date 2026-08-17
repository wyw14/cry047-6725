package handler

import (
	"net/http"
	"time"

	"github.com/cry047/baseline/internal/application"
	"github.com/cry047/baseline/internal/domain"
	"github.com/gin-gonic/gin"
)

// PlanHandler exposes plan endpoints.
type PlanHandler struct {
	svc   *application.PlanService
	sched *application.SchedulerService
}

// NewPlanHandler returns a PlanHandler.
func NewPlanHandler(svc *application.PlanService) *PlanHandler {
	return &PlanHandler{svc: svc}
}

// WithScheduler attaches a SchedulerService to the PlanHandler so the
// /scheduler/scan endpoint can trigger scans.
func (h *PlanHandler) WithScheduler(s *application.SchedulerService) *PlanHandler {
	h.sched = s
	return h
}

// CreateTemplate handles POST /api/v1/plan-templates.
func (h *PlanHandler) CreateTemplate(c *gin.Context) {
	var in domain.PlanTemplateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.CreateTemplate(c.Request.Context(), actor, in)
	if err != nil {
		fail(c, err)
		return
	}
	created(c, out)
}

// ListTemplates handles GET /api/v1/plan-templates.
func (h *PlanHandler) ListTemplates(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.ListTemplates(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// GetTemplate handles GET /api/v1/plan-templates/:id.
func (h *PlanHandler) GetTemplate(c *gin.Context) {
	out, err := h.svc.GetTemplate(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// UpdateTemplate handles PATCH /api/v1/plan-templates/:id.
func (h *PlanHandler) UpdateTemplate(c *gin.Context) {
	var in domain.PlanTemplateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.UpdateTemplate(c.Request.Context(), actor, c.Param("id"), in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// CreatePlanRequest is the body for POST /api/v1/plans.
type CreatePlanRequest struct {
	FacilityID string    `json:"facility_id" binding:"required"`
	TemplateID string    `json:"template_id" binding:"required"`
	StartAt    time.Time `json:"start_at"`
}

// CreatePlan handles POST /api/v1/plans.
func (h *PlanHandler) CreatePlan(c *gin.Context) {
	var body CreatePlanRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.CreatePlan(c.Request.Context(), actor, domain.MaintenancePlanInput{
		FacilityID: body.FacilityID,
		TemplateID: body.TemplateID,
		StartAt:    body.StartAt,
	})
	if err != nil {
		fail(c, err)
		return
	}
	created(c, out)
}

// ListPlans handles GET /api/v1/plans.
func (h *PlanHandler) ListPlans(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// GetPlan handles GET /api/v1/plans/:id.
func (h *PlanHandler) GetPlan(c *gin.Context) {
	out, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// ChangeCycleRequest is the body for PATCH /api/v1/plans/:id/cycle.
type ChangeCycleRequest struct {
	CycleDays int    `json:"cycle_days" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
}

// ChangeCycle handles PATCH /api/v1/plans/:id/cycle.
func (h *PlanHandler) ChangeCycle(c *gin.Context) {
	var in ChangeCycleRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.ChangeCycle(c.Request.Context(), actor, c.Param("id"), in.CycleDays, in.Reason)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// SkipRequest is the body for POST /api/v1/plans/:id/skip.
type SkipRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// Skip handles POST /api/v1/plans/:id/skip.
func (h *PlanHandler) Skip(c *gin.Context) {
	var in SkipRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.Skip(c.Request.Context(), actor, c.Param("id"), in.Reason)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// RegenerateRequest is the body for POST /api/v1/plans/:id/regenerate.
type RegenerateRequest struct {
	Reason string `json:"reason"`
}

// Regenerate handles POST /api/v1/plans/:id/regenerate.
func (h *PlanHandler) Regenerate(c *gin.Context) {
	var in RegenerateRequest
	_ = c.ShouldBindJSON(&in)
	actor := actorFromContext(c)
	out, err := h.svc.Regenerate(c.Request.Context(), actor, c.Param("id"), in.Reason)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// AssignRequest is the body for POST /api/v1/plans/:id/assign.
type AssignRequest struct {
	PersonID string `json:"person_id" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

// AssignResponsible handles POST /api/v1/plans/:id/assign.
func (h *PlanHandler) AssignResponsible(c *gin.Context) {
	var in AssignRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.AssignResponsible(c.Request.Context(), actor, c.Param("id"), in.PersonID, in.Reason)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// ListVersions handles GET /api/v1/plans/:id/versions.
func (h *PlanHandler) ListVersions(c *gin.Context) {
	out, err := h.svc.ListVersions(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// RunSchedulerScan triggers one scheduler scan synchronously.
func (h *PlanHandler) RunSchedulerScan(c *gin.Context) {
	// We need access to the SchedulerService; it's stored on h.sched.
	if h.sched == nil {
		fail(c, domain.ErrNotFound("SchedulerService", "missing"))
		return
	}
	out, err := h.sched.Run(c.Request.Context())
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

var _ = http.StatusOK
