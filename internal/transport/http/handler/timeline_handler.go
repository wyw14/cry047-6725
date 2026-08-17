package handler

import (
	"strconv"

	"github.com/cry047/baseline/internal/application"
	"github.com/gin-gonic/gin"
)

// TimelineHandler exposes timeline endpoints.
type TimelineHandler struct {
	timeline   *application.TimelineService
	facilities *application.FacilityService
}

// NewTimelineHandler returns a TimelineHandler.
func NewTimelineHandler(t *application.TimelineService, f *application.FacilityService) *TimelineHandler {
	return &TimelineHandler{timeline: t, facilities: f}
}

// ByFacility handles GET /api/v1/facilities/:id/timeline.
func (h *TimelineHandler) ByFacility(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	out, err := h.timeline.ListByFacility(c.Request.Context(), c.Param("id"), limit)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}
