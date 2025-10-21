package temporaltype

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"go.temporal.io/sdk/temporal"
)

var (
	ErrNoEncryptionKeyID  = errors.New("no encryption key ID in metadata")
	ErrCiphertextTooShort = errors.New("ciphertext too short")
)

type ErrorType string

const (
	ErrorTypeRetryable        = ErrorType("retryable")
	ErrorTypeNonRetryable     = ErrorType("non_retryable")
	ErrorTypeThrottle         = ErrorType("throttle")
	ErrorTypeInvalidInput     = ErrorType("invalid_input")
	ErrorTypeResourceNotFound = ErrorType("resource_not_found")
	ErrorTypePermissionDenied = ErrorType("permission_denied")
	ErrorTypeDataIntegrity    = ErrorType("data_integrity")
)

func (e ErrorType) String() string {
	return string(e)
}

type ApplicationError struct {
	Type       ErrorType
	Message    string
	Details    map[string]any
	Cause      error
	Retryable  bool
	RetryAfter int // seconds to wait before retry
}

func (e *ApplicationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *ApplicationError) Unwrap() error {
	return e.Cause
}

func (e *ApplicationError) ToTemporalError() error {
	return temporal.NewApplicationError(
		e.Message,
		string(e.Type),
		e.Details,
	)
}

func NewRetryableError(message string, cause error) *ApplicationError {
	return &ApplicationError{
		Type:      ErrorTypeRetryable,
		Message:   message,
		Cause:     cause,
		Retryable: true,
	}
}

func NewRetryableErrorWithDelay(message string, cause error, retryAfter int) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeRetryable,
		Message:    message,
		Cause:      cause,
		Retryable:  true,
		RetryAfter: retryAfter,
	}
}

func NewNonRetryableError(message string, cause error) *ApplicationError {
	return &ApplicationError{
		Type:      ErrorTypeNonRetryable,
		Message:   message,
		Cause:     cause,
		Retryable: false,
	}
}

func NewInvalidInputError(message string, details map[string]any) *ApplicationError {
	return &ApplicationError{
		Type:      ErrorTypeInvalidInput,
		Message:   message,
		Details:   details,
		Retryable: false,
	}
}

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

func NewDataIntegrityError(message string, details map[string]any) *ApplicationError {
	return &ApplicationError{
		Type:      ErrorTypeDataIntegrity,
		Message:   message,
		Details:   details,
		Retryable: false,
	}
}

func NewThrottleError(message string, retryAfter int) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeThrottle,
		Message:    message,
		Retryable:  true,
		RetryAfter: retryAfter,
	}
}

func IsRetryable(err error) bool {
	var appErr *ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Retryable
	}
	if temporal.IsApplicationError(err) {
		var temporalErr *temporal.ApplicationError
		if errors.As(err, &temporalErr) {
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

func GetRetryDelay(err error) int {
	var appErr *ApplicationError
	if errors.As(err, &appErr) && appErr.RetryAfter > 0 {
		return appErr.RetryAfter
	}
	return 0
}

func ClassifyError(err error) *ApplicationError {
	if err == nil {
		return nil
	}

	var appErr *ApplicationError
	if errors.As(err, &appErr) {
		return appErr
	}

	errMsg := strings.ToLower(err.Error())

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
