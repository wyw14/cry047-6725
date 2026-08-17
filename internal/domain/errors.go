package domain

import (
	"errors"
	"fmt"
)

// Stable error codes returned by the API layer.
const (
	CodeOK             = "OK"
	CodeInvalid        = "INVALID"
	CodeNotFound       = "NOT_FOUND"
	CodeConflict       = "CONFLICT"
	CodeStateForbidden = "STATE_FORBIDDEN"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeInternal       = "INTERNAL"
	CodeTimeout        = "TIMEOUT"
)

// Error is the canonical domain error type. Stable Code value enables the
// transport layer to map a domain failure to a deterministic HTTP status.
type Error struct {
	Code    string
	Message string
	Field   string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

func WrapErr(code, message, field string, cause error) *Error {
	return &Error{Code: code, Message: message, Field: field, Cause: cause}
}

func ErrInvalid(message, field string, cause error) *Error {
	return &Error{Code: CodeInvalid, Message: message, Field: field, Cause: cause}
}

func ErrNotFound(entity, id string) *Error {
	return &Error{Code: CodeNotFound, Message: fmt.Sprintf("%s not found: %s", entity, id)}
}

func ErrConflict(message string, cause error) *Error {
	return &Error{Code: CodeConflict, Message: message, Cause: cause}
}

func ErrStateForbidden(message string) *Error {
	return &Error{Code: CodeStateForbidden, Message: message}
}

func ErrUnauthorized(message string) *Error {
	return &Error{Code: CodeUnauthorized, Message: message}
}

func ErrForbidden(message string) *Error {
	return &Error{Code: CodeForbidden, Message: message}
}

// IsDomainError reports whether err is a *domain.Error with the given code.
func IsDomainError(err error, code string) bool {
	var de *Error
	if errors.As(err, &de) {
		return de.Code == code
	}
	return false
}
