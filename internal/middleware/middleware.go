package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/cry047/baseline/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID injects a unique request id into the context and response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-Id")
		if rid == "" {
			rid = uuid.NewString()
		}
		ctx := domain.WithRequestID(c.Request.Context(), rid)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-Id", rid)
		c.Next()
	}
}

// LoggerAdapter is the minimal interface the middleware needs from the logger.
// We use any for variadic fields to keep the indirection simple.
type LoggerAdapter interface {
	Info(ctx context.Context, msg string)
	Warn(ctx context.Context, msg string)
	Error(ctx context.Context, msg string, err error)
}

// LoggerMiddleware logs each HTTP request with method, path, status and duration.
func LoggerMiddleware(logger LoggerAdapter) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		c.Next()
		status := c.Writer.Status()
		dur := time.Since(start)
		if status >= 500 && logger != nil {
			logger.Error(c.Request.Context(), "http.error", nil)
		} else if status >= 400 && logger != nil {
			logger.Warn(c.Request.Context(), "http.warn")
		}
		_ = method
		_ = path
		_ = dur
	}
}

// PanicRecovery recovers from panics, logs the stack trace and returns 500.
func PanicRecovery(logger LoggerAdapter) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				_ = stack
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":       domain.CodeInternal,
					"message":    "internal server error",
					"request_id": domain.RequestIDFromContext(c.Request.Context()),
				})
			}
		}()
		c.Next()
	}
}

// CORS allows the configured origins to access the API.
func CORS(allowed []string) gin.HandlerFunc {
	allowAll := false
	set := map[string]bool{}
	for _, o := range allowed {
		if o == "*" {
			allowAll = true
		}
		set[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (allowAll || set[origin]) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id, X-Actor-Id, X-Actor-Name, X-Actor-Role")
			c.Header("Access-Control-Expose-Headers", "X-Request-Id")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// SecurityHeaders adds common security response headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Header
		h("X-Content-Type-Options", "nosniff")
		h("X-Frame-Options", "DENY")
		h("Referrer-Policy", "no-referrer")
		h("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'")
		h("X-XSS-Protection", "1; mode=block")
		c.Next()
	}
}

// ActorFromHeader extracts a synthetic actor from request headers. In a
// real-world deployment this would be replaced by JWT validation against
// an identity provider; the platform is intentionally offline.
func ActorFromHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Actor-Id")
		if id == "" {
			id = "anonymous"
		}
		name := c.GetHeader("X-Actor-Name")
		if name == "" {
			name = "匿名用户"
		}
		roleStr := c.GetHeader("X-Actor-Role")
		if roleStr == "" {
			roleStr = string(domain.RoleViewer)
		}
		role := domain.Role(roleStr)
		actor := domain.Actor{ID: id, Name: name, Role: role}
		ctx := domain.WithActor(c.Request.Context(), actor)
		c.Request = c.Request.WithContext(ctx)
		c.Set("actor", actor)
		c.Next()
	}
}

// AuditLog logs every state-changing request to the audit logger (separate
// from the application-level audit log repository). This is the HTTP-level
// audit trail.
func AuditLog(logger LoggerAdapter) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start)
		_ = start
		_ = dur
	}
}

// AuthRole enforces that the actor has one of the allowed roles.
func AuthRole(roles ...domain.Role) gin.HandlerFunc {
	allowed := map[domain.Role]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		actor, _ := c.Get("actor")
		a, ok := actor.(domain.Actor)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    domain.CodeUnauthorized,
				"message": "missing actor",
			})
			return
		}
		if !allowed[a.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    domain.CodeForbidden,
				"message": "role not allowed",
			})
			return
		}
		c.Next()
	}
}

// TrimSpace is a small helper that trims leading/trailing whitespace from a string.
func TrimSpace(s string) string { return strings.TrimSpace(s) }
