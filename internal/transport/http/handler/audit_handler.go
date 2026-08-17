package handler

import (
	"github.com/cry047/baseline/internal/application"
	"github.com/gin-gonic/gin"
)

// AuditHandler exposes audit-log endpoints.
type AuditHandler struct {
	svc *application.AuditService
}

// NewAuditHandler returns an AuditHandler.
func NewAuditHandler(svc *application.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List handles GET /api/v1/audit-logs.
func (h *AuditHandler) List(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}
