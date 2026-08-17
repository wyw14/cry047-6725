package handler

import (
	"net/http"

	"github.com/cry047/baseline/internal/application"
	"github.com/cry047/baseline/internal/domain"
	"github.com/gin-gonic/gin"
)

// PlaceHandler exposes place endpoints.
type PlaceHandler struct {
	svc *application.PlaceService
}

// NewPlaceHandler returns a PlaceHandler.
func NewPlaceHandler(svc *application.PlaceService) *PlaceHandler {
	return &PlaceHandler{svc: svc}
}

// Create handles POST /api/v1/places.
func (h *PlaceHandler) Create(c *gin.Context) {
	var in domain.PlaceInput
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

// List handles GET /api/v1/places.
func (h *PlaceHandler) List(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Get handles GET /api/v1/places/:id.
func (h *PlaceHandler) Get(c *gin.Context) {
	out, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Update handles PATCH /api/v1/places/:id.
func (h *PlaceHandler) Update(c *gin.Context) {
	var in domain.PlaceInput
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

// Delete handles DELETE /api/v1/places/:id.
func (h *PlaceHandler) Delete(c *gin.Context) {
	actor := actorFromContext(c)
	if err := h.svc.Delete(c.Request.Context(), actor, c.Param("id")); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
