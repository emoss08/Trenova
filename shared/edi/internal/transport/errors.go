package transport

import (
	"errors"
	"fmt"
	"net/http"
)

// Error types for different failure scenarios
var (
	ErrInvalidRequest   = errors.New("invalid request")
	ErrNotFound         = errors.New("resource not found")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrInternal         = errors.New("internal server error")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// ErrorType represents the type of error for proper HTTP status mapping
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeConflict     ErrorType = "conflict"
	ErrorTypeUnauthorized ErrorType = "unauthorized"
	ErrorTypeForbidden    ErrorType = "forbidden"
	ErrorTypeInternal     ErrorType = "internal"
	ErrorTypeUnavailable  ErrorType = "unavailable"
	ErrorTypeRateLimit    ErrorType = "rate_limit"
)

// ServiceError represents a service-level error with additional context
type ServiceError struct {
	Type    ErrorType              `json:"type"`
	Message string                 `json:"message"`
	Field   string                 `json:"field,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e ServiceError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (field: %s)", e.Type, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewServiceError creates a new service error
func NewServiceError(errType ErrorType, message string) ServiceError {
	return ServiceError{
		Type:    errType,
		Message: message,
	}
}

// WithField adds field information to the error
func (e ServiceError) WithField(field string) ServiceError {
	e.Field = field
	return e
}

// WithDetails adds additional details to the error
func (e ServiceError) WithDetails(details map[string]interface{}) ServiceError {
	e.Details = details
	return e
}

// ValidationError represents validation errors with field-level details
type ValidationError struct {
	Errors []FieldError `json:"errors"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func (e ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation error"
	}
	return fmt.Sprintf("validation failed: %s", e.Errors[0].Message)
}

// HTTPStatusFromError maps errors to HTTP status codes
func HTTPStatusFromError(err error) int {
	if err == nil {
		return http.StatusOK
	}

	var serviceErr ServiceError
	if errors.As(err, &serviceErr) {
		switch serviceErr.Type {
		case ErrorTypeValidation:
			return http.StatusBadRequest
		case ErrorTypeNotFound:
			return http.StatusNotFound
		case ErrorTypeConflict:
			return http.StatusConflict
		case ErrorTypeUnauthorized:
			return http.StatusUnauthorized
		case ErrorTypeForbidden:
			return http.StatusForbidden
		case ErrorTypeUnavailable:
			return http.StatusServiceUnavailable
		case ErrorTypeRateLimit:
			return http.StatusTooManyRequests
		default:
			return http.StatusInternalServerError
		}
	}

	var validationErr ValidationError
	if errors.As(err, &validationErr) {
		return http.StatusBadRequest
	}

	// Check for standard errors
	switch {
	case errors.Is(err, ErrInvalidRequest):
		return http.StatusBadRequest
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, ErrRateLimitExceeded):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// ErrorResponse represents the error response structure
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Message   string                 `json:"message,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewErrorResponse creates an error response from an error
func NewErrorResponse(err error, requestID string) ErrorResponse {
	resp := ErrorResponse{
		Error:     err.Error(),
		RequestID: requestID,
	}

	var serviceErr ServiceError
	if errors.As(err, &serviceErr) {
		resp.Message = serviceErr.Message
		resp.Details = serviceErr.Details
	}

	return resp
}