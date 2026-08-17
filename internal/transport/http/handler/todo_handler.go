package handler

import (
	"github.com/cry047/baseline/internal/application"
	"github.com/gin-gonic/gin"
)

// TodoHandler exposes todo endpoints.
type TodoHandler struct {
	svc *application.TodoService
}

// NewTodoHandler returns a TodoHandler.
func NewTodoHandler(svc *application.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

// List handles GET /api/v1/todos.
func (h *TodoHandler) List(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// ListByUser handles GET /api/v1/todos/by-user/:user_id.
func (h *TodoHandler) ListByUser(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.ListByUser(c.Request.Context(), c.Param("user_id"), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Complete handles PATCH /api/v1/todos/:id/complete.
func (h *TodoHandler) Complete(c *gin.Context) {
	actor := actorFromContext(c)
	out, err := h.svc.Complete(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// TodoSkipRequest is the body for PATCH /api/v1/todos/:id/skip.
type TodoSkipRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// Skip handles PATCH /api/v1/todos/:id/skip.
func (h *TodoHandler) Skip(c *gin.Context) {
	var in TodoSkipRequest
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

// ReassignRequest is the body for PATCH /api/v1/todos/:id/reassign.
type ReassignRequest struct {
	PersonID string `json:"person_id" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

// Reassign handles PATCH /api/v1/todos/:id/reassign.
func (h *TodoHandler) Reassign(c *gin.Context) {
	var in ReassignRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.Reassign(c.Request.Context(), actor, c.Param("id"), in.PersonID, in.Reason)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}
