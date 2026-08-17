package handler

import (
	"strconv"
	"strings"

	"github.com/cry047/baseline/internal/domain"
	"github.com/gin-gonic/gin"
)

// responseWriter is the indirection handler uses to write JSON responses.
// It is set by the transport package to avoid an import cycle.
var responseWriter ResponseWriter

// ResponseWriter is the interface the transport package implements to write
// responses without creating an import cycle.
type ResponseWriter interface {
	OK(c *gin.Context, data any)
	Created(c *gin.Context, data any)
	Accepted(c *gin.Context, data any)
	NoContent(c *gin.Context)
	Fail(c *gin.Context, err error)
	FailWithFieldErrors(c *gin.Context, msg string, errs []FieldError)
}

// FieldError describes a single field-level validation error.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// SetResponseWriter is called by the transport package init.
func SetResponseWriter(w ResponseWriter) { responseWriter = w }

// helper methods that use the injected writer.
func ok(c *gin.Context, data any)       { responseWriter.OK(c, data) }
func created(c *gin.Context, data any)  { responseWriter.Created(c, data) }
func accepted(c *gin.Context, data any) { responseWriter.Accepted(c, data) }
func noContent(c *gin.Context)          { responseWriter.NoContent(c) }
func fail(c *gin.Context, err error)    { responseWriter.Fail(c, err) }
func failWithFieldErrors(c *gin.Context, msg string, errs []FieldError) {
	responseWriter.FailWithFieldErrors(c, msg, errs)
}

// Handlers aggregates every HTTP handler.
type Handlers struct {
	Places        *PlaceHandler
	People        *PersonHandler
	Facilities    *FacilityHandler
	Plans         *PlanHandler
	Executions    *ExecutionHandler
	Anomalies     *AnomalyHandler
	Todos         *TodoHandler
	Notifications *NotificationHandler
	Audit         *AuditHandler
	Timeline      *TimelineHandler
}

// pageQuery builds a domain.PageQuery from a gin.Context. It accepts:
//   - limit, page (or offset)
//   - order_by, order
//   - any additional query parameter as a filter, except reserved keys.
func pageQuery(c *gin.Context) domain.PageQuery {
	q := domain.PageQuery{
		Filters: map[string]string{},
	}
	limitStr := c.Query("limit")
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			q.Limit = v
		}
	}
	pageStr := c.Query("page")
	if pageStr != "" {
		if v, err := strconv.Atoi(pageStr); err == nil && v > 0 {
			q.Offset = (v - 1) * q.Limit
		}
	}
	offsetStr := c.Query("offset")
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			q.Offset = v
		}
	}
	q.OrderBy = c.Query("order_by")
	q.Order = c.Query("order")
	reserved := map[string]bool{
		"limit": true, "page": true, "offset": true,
		"order_by": true, "order": true,
	}
	for k, vs := range c.Request.URL.Query() {
		if reserved[k] {
			continue
		}
		if len(vs) == 0 {
			continue
		}
		q.Filters[k] = strings.Join(vs, ",")
		if len(vs) > 1 {
			// multi-value filter: keep comma-separated; repository treats as OR.
		}
	}
	q.Normalize()
	return q
}

// actorFromContext extracts the actor from the gin context.
func actorFromContext(c *gin.Context) domain.Actor {
	v, ok := c.Get("actor")
	if !ok {
		return domain.Actor{ID: "anonymous", Name: "匿名用户", Role: domain.RoleViewer}
	}
	a, _ := v.(domain.Actor)
	return a
}
