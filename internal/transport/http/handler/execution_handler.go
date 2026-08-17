package handler

import (
	"github.com/cry047/baseline/internal/application"
	"github.com/cry047/baseline/internal/domain"
	"github.com/gin-gonic/gin"
)

// ExecutionHandler exposes execution endpoints.
type ExecutionHandler struct {
	svc *application.ExecutionService
}

// NewExecutionHandler returns an ExecutionHandler.
func NewExecutionHandler(svc *application.ExecutionService) *ExecutionHandler {
	return &ExecutionHandler{svc: svc}
}

// Submit handles POST /api/v1/executions.
func (h *ExecutionHandler) Submit(c *gin.Context) {
	var in domain.ExecutionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	in.ExecutedBy = actor.ID
	out, err := h.svc.Submit(c.Request.Context(), actor, in)
	if err != nil {
		fail(c, err)
		return
	}
	created(c, out)
}

// List handles GET /api/v1/executions.
func (h *ExecutionHandler) List(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Get handles GET /api/v1/executions/:id.
func (h *ExecutionHandler) Get(c *gin.Context) {
	out, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// ReviewRequest is the body for POST /api/v1/executions/:id/review.
type ReviewRequest struct {
	Comment  string `json:"comment" binding:"required"`
	Approved bool   `json:"approved"`
}

// Review handles POST /api/v1/executions/:id/review.
func (h *ExecutionHandler) Review(c *gin.Context) {
	var in ReviewRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.Review(c.Request.Context(), actor, c.Param("id"), in.Comment, in.Approved)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}
