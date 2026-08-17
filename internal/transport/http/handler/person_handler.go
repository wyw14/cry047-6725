package handler

import (
	"net/http"

	"github.com/cry047/baseline/internal/application"
	"github.com/cry047/baseline/internal/domain"
	"github.com/gin-gonic/gin"
)

// PersonHandler exposes responsible-person endpoints.
type PersonHandler struct {
	svc *application.PersonService
}

// NewPersonHandler returns a PersonHandler.
func NewPersonHandler(svc *application.PersonService) *PersonHandler {
	return &PersonHandler{svc: svc}
}

// Create handles POST /api/v1/responsible-persons.
func (h *PersonHandler) Create(c *gin.Context) {
	var in domain.ResponsiblePersonInput
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.Create(c.Request.Context(), actor, in)
	if err != nil {
		fail(c, err)
		return
	}
	created(c, out)
}

// List handles GET /api/v1/responsible-persons.
func (h *PersonHandler) List(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Get handles GET /api/v1/responsible-persons/:id.
func (h *PersonHandler) Get(c *gin.Context) {
	out, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Update handles PATCH /api/v1/responsible-persons/:id.
func (h *PersonHandler) Update(c *gin.Context) {
	var in domain.ResponsiblePersonInput
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.Update(c.Request.Context(), actor, c.Param("id"), in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// SetActiveRequest is the body for POST /api/v1/responsible-persons/:id/active.
type SetActiveRequest struct {
	Active bool `json:"active"`
}

// SetActive toggles the active flag.
func (h *PersonHandler) SetActive(c *gin.Context) {
	var in SetActiveRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.SetActive(c.Request.Context(), actor, c.Param("id"), in.Active)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Delete removes a person.
func (h *PersonHandler) Delete(c *gin.Context) {
	actor := actorFromContext(c)
	if err := h.svc.Delete(c.Request.Context(), actor, c.Param("id")); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
