package errors

import (
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

// Common error types
var (
	ErrNotFound          = errors.New("resource not found")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrValidation        = errors.New("validation error")
	ErrInternal          = errors.New("internal server error")
	ErrBadRequest        = errors.New("bad request")
	ErrConflict          = errors.New("conflict")
	ErrTooManyRequests   = errors.New("too many requests")
	ErrInvalidTenant     = errors.New("invalid tenant")
	ErrTenantNotFound    = errors.New("tenant not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrTaskNotFound      = errors.New("task not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenExpired      = errors.New("token expired")
	ErrInvalidToken      = errors.New("invalid token")
)

// AppError represents an application error with additional context
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WithDetails adds details to an application error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// Common error constructors
func BadRequest(message string, err error) *AppError {
	return NewAppError(http.StatusBadRequest, message, err)
}

func Unauthorized(message string, err error) *AppError {
	return NewAppError(http.StatusUnauthorized, message, err)
}

func Forbidden(message string, err error) *AppError {
	return NewAppError(http.StatusForbidden, message, err)
}

func NotFound(message string, err error) *AppError {
	return NewAppError(http.StatusNotFound, message, err)
}

func Conflict(message string, err error) *AppError {
	return NewAppError(http.StatusConflict, message, err)
}

func InternalServer(message string, err error) *AppError {
	return NewAppError(http.StatusInternalServerError, message, err)
}

func TooManyRequests(message string, err error) *AppError {
	return NewAppError(http.StatusTooManyRequests, message, err)
}

// Validation error helpers
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (v *ValidationErrors) Error() string {
	return "validation failed"
}

func (v *ValidationErrors) Add(field, message string, value any) {
	v.Errors = append(v.Errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

func (v *ValidationErrors) HasErrors() bool {
	return len(v.Errors) > 0
}

func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]ValidationError, 0),
	}
}

// HandleDBError converts GORM errors to appropriate AppErrors
func HandleDBError(err error, resourceName string) *AppError {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return NotFound(fmt.Sprintf("%s not found", resourceName), err)
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return Conflict(fmt.Sprintf("%s already exists", resourceName), err)
	}

	// For other database errors, return a generic internal error
	return InternalServer("Database operation failed", err)
}