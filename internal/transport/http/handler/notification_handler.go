package handler

import (
	"github.com/cry047/baseline/internal/application"
	"github.com/gin-gonic/gin"
)

// NotificationHandler exposes notification endpoints.
type NotificationHandler struct {
	svc *application.NotificationService
}

// NewNotificationHandler returns a NotificationHandler.
func NewNotificationHandler(svc *application.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// List handles GET /api/v1/notifications.
func (h *NotificationHandler) List(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// ListByUser handles GET /api/v1/notifications/by-user/:user_id.
func (h *NotificationHandler) ListByUser(c *gin.Context) {
	q := pageQuery(c)
	out, err := h.svc.ListByUser(c.Request.Context(), c.Param("user_id"), q)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// ListUnread handles GET /api/v1/notifications/unread/:user_id.
func (h *NotificationHandler) ListUnread(c *gin.Context) {
	out, err := h.svc.ListUnread(c.Request.Context(), c.Param("user_id"))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}

// MarkRead handles PATCH /api/v1/notifications/:id/read.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	actor := actorFromContext(c)
	if err := h.svc.MarkRead(c.Request.Context(), actor, c.Param("id")); err != nil {
		fail(c, err)
		return
	}
	c.Status(204)
}
