package handler

import (
	"time"

	"github.com/cry047/baseline/internal/application"
	"github.com/cry047/baseline/internal/domain"
	"github.com/gin-gonic/gin"
)

// AnomalyHandler exposes anomaly endpoints.
type AnomalyHandler struct {
	svc *application.AnomalyService
}

// NewAnomalyHandler returns an AnomalyHandler.
func NewAnomalyHandler(svc *application.AnomalyService) *AnomalyHandler {
	return &AnomalyHandler{svc: svc}
}

// Discover handles POST /api/v1/anomalies.
func (h *AnomalyHandler) Discover(c *gin.Context) {
	var in domain.AnomalyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	in.DiscoveredBy = actor.ID
	if in.DiscoveredAt.IsZero() {
		in.DiscoveredAt = time.Now().UTC()
	}
	out, err := h.svc.Discover(c.Request.Context(), actor, in)
	if err != nil {
		fail(c, err)
		return
	}
	created(c, out)
}

// List handles GET /api/v1/anomalies.
func (h *AnomalyHandler) List(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Get handles GET /api/v1/anomalies/:id.
func (h *AnomalyHandler) Get(c *gin.Context) {
	out, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Rectify handles POST /api/v1/anomalies/:id/rectify.
func (h *AnomalyHandler) Rectify(c *gin.Context) {
	var in domain.RectificationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	in.AnomalyID = c.Param("id")
	in.RectifiedBy = actor.ID
	out, err := h.svc.Rectify(c.Request.Context(), actor, in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// ReinspectRequest is the body for POST /api/v1/anomalies/:id/reinspect.
type ReinspectRequest struct {
	Result string    `json:"result" binding:"required"`
	Pass   bool      `json:"pass"`
	At     time.Time `json:"at"`
}

// Reinspect handles POST /api/v1/anomalies/:id/reinspect.
func (h *AnomalyHandler) Reinspect(c *gin.Context) {
	var body ReinspectRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		failWithFieldErrors(c, "请求体无效", []FieldError{{Field: "body", Message: err.Error()}})
		return
	}
	actor := actorFromContext(c)
	at := body.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	in := domain.ReinspectionInput{
		AnomalyID:     c.Param("id"),
		Result:        body.Result,
		ReinspectedBy: actor.ID,
		ReinspectedAt: at,
	}
	pass := body.Pass || body.Result != ""
	out, err := h.svc.Reinspect(c.Request.Context(), actor, in, pass)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// Recover handles POST /api/v1/anomalies/:id/recover.
func (h *AnomalyHandler) Recover(c *gin.Context) {
	actor := actorFromContext(c)
	in := domain.RecoverInput{
		AnomalyID:   c.Param("id"),
		RecoveredBy: actor.ID,
	}
	out, err := h.svc.Recover(c.Request.Context(), actor, in)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}
