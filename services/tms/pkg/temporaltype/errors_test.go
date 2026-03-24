package temporaltype

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		et       ErrorType
		expected string
	}{
		{"retryable", ErrorTypeRetryable, "retryable"},
		{"non_retryable", ErrorTypeNonRetryable, "non_retryable"},
		{"throttle", ErrorTypeThrottle, "throttle"},
		{"invalid_input", ErrorTypeInvalidInput, "invalid_input"},
		{"resource_not_found", ErrorTypeResourceNotFound, "resource_not_found"},
		{"permission_denied", ErrorTypePermissionDenied, "permission_denied"},
		{"data_integrity", ErrorTypeDataIntegrity, "data_integrity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.et.String())
		})
	}
}

func TestApplicationError_Error(t *testing.T) {
	t.Parallel()

	t.Run("without cause", func(t *testing.T) {
		t.Parallel()
		err := &ApplicationError{
			Type:    ErrorTypeRetryable,
			Message: "something failed",
		}
		assert.Equal(t, "retryable: something failed", err.Error())
	})

	t.Run("with cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("root cause")
		err := &ApplicationError{
			Type:    ErrorTypeNonRetryable,
			Message: "operation failed",
			Cause:   cause,
		}
		assert.Contains(t, err.Error(), "non_retryable: operation failed")
		assert.Contains(t, err.Error(), "root cause")
	})
}

func TestApplicationError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("wrapped error")
	err := &ApplicationError{
		Type:  ErrorTypeRetryable,
		Cause: cause,
	}
	assert.Equal(t, cause, err.Unwrap())
}

func TestNewRetryableError(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection refused")
	err := NewRetryableError("connect failed", cause)

	assert.Equal(t, ErrorTypeRetryable, err.Type)
	assert.Equal(t, "connect failed", err.Message)
	assert.Equal(t, cause, err.Cause)
	assert.True(t, err.Retryable)
	assert.Zero(t, err.RetryAfter)
}

func TestNewRetryableErrorWithDelay(t *testing.T) {
	t.Parallel()

	cause := errors.New("timeout")
	err := NewRetryableErrorWithDelay("timed out", cause, 30)

	assert.Equal(t, ErrorTypeRetryable, err.Type)
	assert.Equal(t, "timed out", err.Message)
	assert.Equal(t, cause, err.Cause)
	assert.True(t, err.Retryable)
	assert.Equal(t, 30, err.RetryAfter)
}

func TestNewNonRetryableError(t *testing.T) {
	t.Parallel()

	cause := errors.New("bad data")
	err := NewNonRetryableError("corrupt input", cause)

	assert.Equal(t, ErrorTypeNonRetryable, err.Type)
	assert.Equal(t, "corrupt input", err.Message)
	assert.Equal(t, cause, err.Cause)
	assert.False(t, err.Retryable)
}

func TestNewInvalidInputError(t *testing.T) {
	t.Parallel()

	details := map[string]any{"field": "email"}
	err := NewInvalidInputError("invalid email", details)

	assert.Equal(t, ErrorTypeInvalidInput, err.Type)
	assert.Equal(t, "invalid email", err.Message)
	assert.Equal(t, details, err.Details)
	assert.False(t, err.Retryable)
}

func TestNewResourceNotFoundError(t *testing.T) {
	t.Parallel()

	err := NewResourceNotFoundError("User", "usr_123")

	assert.Equal(t, ErrorTypeResourceNotFound, err.Type)
	assert.Contains(t, err.Message, "User")
	assert.Contains(t, err.Message, "usr_123")
	assert.Equal(t, "User", err.Details["resourceType"])
	assert.Equal(t, "usr_123", err.Details["resourceID"])
	assert.False(t, err.Retryable)
}

func TestNewPermissionDeniedError(t *testing.T) {
	t.Parallel()

	err := NewPermissionDeniedError("delete", "user")

	assert.Equal(t, ErrorTypePermissionDenied, err.Type)
	assert.Contains(t, err.Message, "delete")
	assert.Contains(t, err.Message, "user")
	assert.Equal(t, "delete", err.Details["action"])
	assert.Equal(t, "user", err.Details["resource"])
	assert.False(t, err.Retryable)
}

func TestNewDataIntegrityError(t *testing.T) {
	t.Parallel()

	details := map[string]any{"constraint": "unique_email"}
	err := NewDataIntegrityError("duplicate entry", details)

	assert.Equal(t, ErrorTypeDataIntegrity, err.Type)
	assert.Equal(t, "duplicate entry", err.Message)
	assert.Equal(t, details, err.Details)
	assert.False(t, err.Retryable)
}

func TestNewThrottleError(t *testing.T) {
	t.Parallel()

	err := NewThrottleError("too many requests", 60)

	assert.Equal(t, ErrorTypeThrottle, err.Type)
	assert.Equal(t, "too many requests", err.Message)
	assert.True(t, err.Retryable)
	assert.Equal(t, 60, err.RetryAfter)
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"retryable app error", NewRetryableError("msg", nil), true},
		{"non-retryable app error", NewNonRetryableError("msg", nil), false},
		{"invalid input error", NewInvalidInputError("msg", nil), false},
		{"resource not found", NewResourceNotFoundError("x", "y"), false},
		{"permission denied", NewPermissionDeniedError("a", "b"), false},
		{"data integrity", NewDataIntegrityError("msg", nil), false},
		{"throttle error", NewThrottleError("msg", 30), true},
		{"unknown error", errors.New("unknown"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsRetryable(tt.err))
		})
	}
}

func TestGetRetryDelay(t *testing.T) {
	t.Parallel()

	t.Run("with delay", func(t *testing.T) {
		t.Parallel()
		err := NewRetryableErrorWithDelay("msg", nil, 45)
		assert.Equal(t, 45, GetRetryDelay(err))
	})

	t.Run("without delay", func(t *testing.T) {
		t.Parallel()
		err := NewRetryableError("msg", nil)
		assert.Equal(t, 0, GetRetryDelay(err))
	})

	t.Run("non app error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("regular error")
		assert.Equal(t, 0, GetRetryDelay(err))
	})
}

func TestClassifyError(t *testing.T) {
	t.Parallel()

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, ClassifyError(nil))
	})

	t.Run("already classified", func(t *testing.T) {
		t.Parallel()
		original := NewRetryableError("already classified", nil)
		result := ClassifyError(original)
		assert.Equal(t, original, result)
	})

	t.Run("connection error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("connection refused by host")
		result := ClassifyError(err)
		require.NotNil(t, result)
		assert.Equal(t, ErrorTypeRetryable, result.Type)
	})

	t.Run("timeout error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("request timeout exceeded")
		result := ClassifyError(err)
		require.NotNil(t, result)
		assert.Equal(t, ErrorTypeRetryable, result.Type)
	})

	t.Run("not found error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("resource not found")
		result := ClassifyError(err)
		require.NotNil(t, result)
		assert.Equal(t, ErrorTypeResourceNotFound, result.Type)
	})

	t.Run("permission error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("permission denied for user")
		result := ClassifyError(err)
		require.NotNil(t, result)
		assert.Equal(t, ErrorTypePermissionDenied, result.Type)
	})

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("invalid email format")
		result := ClassifyError(err)
		require.NotNil(t, result)
		assert.Equal(t, ErrorTypeInvalidInput, result.Type)
	})

	t.Run("rate limit error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("rate limit exceeded")
		result := ClassifyError(err)
		require.NotNil(t, result)
		assert.Equal(t, ErrorTypeThrottle, result.Type)
		assert.Equal(t, 60, result.RetryAfter)
	})

	t.Run("unknown error defaults to retryable", func(t *testing.T) {
		t.Parallel()
		err := errors.New("something completely unexpected happened")
		result := ClassifyError(err)
		require.NotNil(t, result)
		assert.Equal(t, ErrorTypeRetryable, result.Type)
	})
}

func TestTaskQueue_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "audit-queue", TaskQueueAudit.String())
	assert.Equal(t, "dispatch-queue", TaskQueueDispatch.String())
	assert.Equal(t, "thumbnail-queue", TaskQueueThumbnail.String())
}

func TestBasePayload_GetOrganizationID(t *testing.T) {
	t.Parallel()

	p := &BasePayload{OrganizationID: "org_123"}
	assert.Equal(t, "org_123", string(p.GetOrganizationID()))
}

func TestBasePayload_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	p := &BasePayload{BusinessUnitID: "bu_456"}
	assert.Equal(t, "bu_456", string(p.GetBusinessUnitID()))
}

func TestConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "audit-queue", AuditTaskQueue)
	assert.Equal(t, "thumbnail-queue", ThumbnailTaskQueue)
}
