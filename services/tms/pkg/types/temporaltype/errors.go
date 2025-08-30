package temporaltype

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	"go.temporal.io/sdk/temporal"
)

// ErrorType represents the type of error for classification
type ErrorType string

const (
	// ErrorTypeRetryable indicates the error should be retried
	ErrorTypeRetryable = ErrorType("retryable")
	// ErrorTypeNonRetryable indicates the error should not be retried
	ErrorTypeNonRetryable = ErrorType("non_retryable")
	// ErrorTypeThrottle indicates the operation should be throttled
	ErrorTypeThrottle = ErrorType("throttle")
	// ErrorTypeInvalidInput indicates invalid input that won't succeed on retry
	ErrorTypeInvalidInput = ErrorType("invalid_input")
	// ErrorTypeResourceNotFound indicates a resource was not found
	ErrorTypeResourceNotFound = ErrorType("resource_not_found")
	// ErrorTypePermissionDenied indicates insufficient permissions
	ErrorTypePermissionDenied = ErrorType("permission_denied")
	// ErrorTypeDataIntegrity indicates data consistency issues
	ErrorTypeDataIntegrity = ErrorType("data_integrity")
)

func (e ErrorType) String() string {
	return string(e)
}

// ApplicationError wraps errors with additional context for Temporal
type ApplicationError struct {
	Type       ErrorType
	Message    string
	Details    map[string]any
	Cause      error
	Retryable  bool
	RetryAfter int // seconds to wait before retry
}

// Error implements the error interface
func (e *ApplicationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *ApplicationError) Unwrap() error {
	return e.Cause
}

// ToTemporalError converts ApplicationError to Temporal's ApplicationError
func (e *ApplicationError) ToTemporalError() error {
	return temporal.NewApplicationError(
		e.Message,
		string(e.Type),
		e.Details,
	)
}

// NewRetryableError creates a new retryable error
func NewRetryableError(message string, cause error) *ApplicationError {
	return &ApplicationError{
		Type:      ErrorTypeRetryable,
		Message:   message,
		Cause:     cause,
		Retryable: true,
	}
}

// NewRetryableErrorWithDelay creates a retryable error with retry delay
func NewRetryableErrorWithDelay(message string, cause error, retryAfter int) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeRetryable,
		Message:    message,
		Cause:      cause,
		Retryable:  true,
		RetryAfter: retryAfter,
	}
}

// NewNonRetryableError creates a new non-retryable error
func NewNonRetryableError(message string, cause error) *ApplicationError {
	return &ApplicationError{
		Type:      ErrorTypeNonRetryable,
		Message:   message,
		Cause:     cause,
		Retryable: false,
	}
}

// NewInvalidInputError creates an error for invalid input
func NewInvalidInputError(message string, details map[string]any) *ApplicationError {
	return &ApplicationError{
		Type:      ErrorTypeInvalidInput,
		Message:   message,
		Details:   details,
		Retryable: false,
	}
}

// NewResourceNotFoundError creates an error for missing resources
func NewResourceNotFoundError(resourceType, resourceID string) *ApplicationError {
	return &ApplicationError{
		Type:    ErrorTypeResourceNotFound,
		Message: fmt.Sprintf("%s not found: %s", resourceType, resourceID),
		Details: map[string]any{
			"resourceType": resourceType,
			"resourceID":   resourceID,
		},
		Retryable: false,
	}
}

// NewPermissionDeniedError creates an error for permission issues
func NewPermissionDeniedError(action, resource string) *ApplicationError {
	return &ApplicationError{
		Type:    ErrorTypePermissionDenied,
		Message: fmt.Sprintf("Permission denied for action %s on resource %s", action, resource),
		Details: map[string]any{
			"action":   action,
			"resource": resource,
		},
		Retryable: false,
	}
}

// NewDataIntegrityError creates an error for data consistency issues
func NewDataIntegrityError(message string, details map[string]any) *ApplicationError {
	return &ApplicationError{
		Type:      ErrorTypeDataIntegrity,
		Message:   message,
		Details:   details,
		Retryable: false,
	}
}

// NewThrottleError creates an error indicating throttling
func NewThrottleError(message string, retryAfter int) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeThrottle,
		Message:    message,
		Retryable:  true,
		RetryAfter: retryAfter,
	}
}

// IsRetryable checks if an error should be retried
func IsRetryable(err error) bool {
	if appErr, ok := err.(*ApplicationError); ok {
		return appErr.Retryable
	}
	// Check if it's a Temporal ApplicationError
	if temporal.IsApplicationError(err) {
		var temporalErr *temporal.ApplicationError
		if err, ok := err.(*temporal.ApplicationError); ok {
			temporalErr = err
			// Check the error type from Temporal
			switch temporalErr.Type() {
			case ErrorTypeNonRetryable.String(),
				ErrorTypeInvalidInput.String(),
				ErrorTypeResourceNotFound.String(),
				ErrorTypePermissionDenied.String(),
				ErrorTypeDataIntegrity.String():
				return false
			}
		}
	}
	return true // Default to retryable for unknown errors
}

// GetRetryDelay extracts retry delay from error if available
func GetRetryDelay(err error) int {
	if appErr, ok := err.(*ApplicationError); ok && appErr.RetryAfter > 0 {
		return appErr.RetryAfter
	}
	return 0
}

// ClassifyError classifies an error and wraps it appropriately
func ClassifyError(err error) *ApplicationError {
	if err == nil {
		return nil
	}

	// If it's already an ApplicationError, return it
	if appErr, ok := err.(*ApplicationError); ok {
		return appErr
	}

	// Classify based on error message or type
	// This can be extended based on your specific error patterns
	errMsg := strings.ToLower(err.Error())

	// Check for common patterns using lo.ContainsBy
	connectionErrors := []string{"connection refused", "connection reset", "timeout"}
	notFoundErrors := []string{"not found", "does not exist"}
	permissionErrors := []string{"permission denied", "unauthorized", "forbidden"}
	validationErrors := []string{"invalid", "malformed", "bad request"}
	rateLimitErrors := []string{"rate limit", "too many requests"}
	
	switch {
	case lo.ContainsBy(connectionErrors, func(pattern string) bool {
		return strings.Contains(errMsg, pattern)
	}):
		return NewRetryableError("Connection error", err)
	case lo.ContainsBy(notFoundErrors, func(pattern string) bool {
		return strings.Contains(errMsg, pattern)
	}):
		return NewResourceNotFoundError("Resource", "unknown")
	case lo.ContainsBy(permissionErrors, func(pattern string) bool {
		return strings.Contains(errMsg, pattern)
	}):
		return NewPermissionDeniedError("unknown", "unknown")
	case lo.ContainsBy(validationErrors, func(pattern string) bool {
		return strings.Contains(errMsg, pattern)
	}):
		return NewInvalidInputError(err.Error(), nil)
	case lo.ContainsBy(rateLimitErrors, func(pattern string) bool {
		return strings.Contains(errMsg, pattern)
	}):
		return NewThrottleError(err.Error(), 60) // Default 60 second retry
	default:
		// Default to retryable for unknown errors
		return NewRetryableError("Unexpected error", err)
	}
}
