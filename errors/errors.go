package errors

import (
	"fmt"
	"net/http"
)

// AppError represents application error
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Err        error                  `json:"-"`
}

// Error implements error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap implements error unwrapping
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func New(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Wrap wraps an error with AppError
func Wrap(err error, code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// WithDetails adds details to error
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// Common errors
var (
	// Client errors (4xx)
	ErrBadRequest      = New("bad_request", "Bad request", http.StatusBadRequest)
	ErrUnauthorized    = New("unauthorized", "Unauthorized", http.StatusUnauthorized)
	ErrForbidden       = New("forbidden", "Forbidden", http.StatusForbidden)
	ErrNotFound        = New("not_found", "Resource not found", http.StatusNotFound)
	ErrConflict        = New("conflict", "Resource already exists", http.StatusConflict)
	ErrValidation      = New("validation_error", "Validation failed", http.StatusUnprocessableEntity)
	ErrTooManyRequests = New("too_many_requests", "Too many requests", http.StatusTooManyRequests)

	// Server errors (5xx)
	ErrInternal           = New("internal_error", "Internal server error", http.StatusInternalServerError)
	ErrNotImplemented     = New("not_implemented", "Not implemented", http.StatusNotImplemented)
	ErrServiceUnavailable = New("service_unavailable", "Service unavailable", http.StatusServiceUnavailable)
)

// IsAppError checks if error is AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError returns AppError or creates one from error
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return Wrap(err, "internal_error", "Internal server error", http.StatusInternalServerError)
}
