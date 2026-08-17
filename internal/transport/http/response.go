package http

import (
	"errors"
	"net/http"

	"github.com/cry047/baseline/internal/domain"
	"github.com/gin-gonic/gin"
)

// Response is the standard JSON envelope returned by every API endpoint.
type Response struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	Data      any          `json:"data,omitempty"`
	Errors    []FieldError `json:"errors,omitempty"`
	RequestID string       `json:"request_id,omitempty"`
}

// FieldError describes a single field-level validation error.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// OK writes a 200 with a success response.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:      domain.CodeOK,
		Message:   "OK",
		Data:      data,
		RequestID: domain.RequestIDFromContext(c.Request.Context()),
	})
}

// Created writes a 201 with a success response.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{
		Code:      domain.CodeOK,
		Message:   "Created",
		Data:      data,
		RequestID: domain.RequestIDFromContext(c.Request.Context()),
	})
}

// Accepted writes a 202.
func Accepted(c *gin.Context, data any) {
	c.JSON(http.StatusAccepted, Response{
		Code:      domain.CodeOK,
		Message:   "Accepted",
		Data:      data,
		RequestID: domain.RequestIDFromContext(c.Request.Context()),
	})
}

// NoContent writes a 204.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Fail writes a domain error to the response. It performs error mapping from
// domain.Error codes to HTTP status codes.
func Fail(c *gin.Context, err error) {
	if err == nil {
		c.Status(http.StatusNoContent)
		return
	}
	var de *domain.Error
	if errors.As(err, &de) {
		writeDomainError(c, de)
		return
	}
	// Fallback: unknown error.
	c.JSON(http.StatusInternalServerError, Response{
		Code:      domain.CodeInternal,
		Message:   err.Error(),
		RequestID: domain.RequestIDFromContext(c.Request.Context()),
	})
}

// writeDomainError writes the appropriate HTTP status based on the domain code.
func writeDomainError(c *gin.Context, de *domain.Error) {
	status := http.StatusInternalServerError
	switch de.Code {
	case domain.CodeInvalid:
		status = http.StatusBadRequest
	case domain.CodeNotFound:
		status = http.StatusNotFound
	case domain.CodeConflict:
		status = http.StatusConflict
	case domain.CodeStateForbidden:
		status = http.StatusUnprocessableEntity
	case domain.CodeUnauthorized:
		status = http.StatusUnauthorized
	case domain.CodeForbidden:
		status = http.StatusForbidden
	case domain.CodeTimeout:
		status = http.StatusGatewayTimeout
	}
	resp := Response{
		Code:      de.Code,
		Message:   de.Message,
		RequestID: domain.RequestIDFromContext(c.Request.Context()),
	}
	if de.Field != "" {
		resp.Errors = []FieldError{{Field: de.Field, Message: de.Message}}
	}
	c.JSON(status, resp)
}

// FailWithFieldErrors writes a 400 with multiple field errors.
func FailWithFieldErrors(c *gin.Context, msg string, errs []FieldError) {
	c.JSON(http.StatusBadRequest, Response{
		Code:      domain.CodeInvalid,
		Message:   msg,
		Errors:    errs,
		RequestID: domain.RequestIDFromContext(c.Request.Context()),
	})
}
