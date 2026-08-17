package handler

import (
	"net/http"

	"github.com/cry047/baseline/internal/application"
	"github.com/cry047/baseline/internal/domain"
	"github.com/gin-gonic/gin"
)

// FacilityHandler exposes facility endpoints.
type FacilityHandler struct {
	svc *application.FacilityService
}

// NewFacilityHandler returns a FacilityHandler.
func NewFacilityHandler(svc *application.FacilityService) *FacilityHandler {
	return &FacilityHandler{svc: svc}
}

// Create handles POST /api/v1/facilities.
func (h *FacilityHandler) Create(c *gin.Context) {
	var in domain.FacilityInput
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

// List handles GET /api/v1/facilities.
func (h *FacilityHandler) List(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Get handles GET /api/v1/facilities/:id.
func (h *FacilityHandler) Get(c *gin.Context) {
	out, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Update handles PATCH /api/v1/facilities/:id.
func (h *FacilityHandler) Update(c *gin.Context) {
	var in domain.FacilityInput
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

// TransitionRequest is the body for POST /api/v1/facilities/:id/transition.
type TransitionRequest struct {
	To     domain.FacilityStatus `json:"to"`
	Reason string                `json:"reason"`
}

// TransitionState handles POST /api/v1/facilities/:id/transition.
func (h *FacilityHandler) TransitionState(c *gin.Context) {
	var in TransitionRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	out, err := h.svc.TransitionState(c.Request.Context(), actor, c.Param("id"), in.To, in.Reason)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// NextDateInfo handles GET /api/v1/facilities/:id/next-date.
func (h *FacilityHandler) NextDateInfo(c *gin.Context) {
	out, err := h.svc.NextDateInfo(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

var _ = http.StatusOK
