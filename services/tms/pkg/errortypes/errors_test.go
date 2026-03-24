package errortypes_test

import (
	"encoding/json"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	val "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	t.Run("NewValidationError", func(t *testing.T) {
		err := errortypes.NewValidationError(
			"email",
			errortypes.ErrInvalidFormat,
			"invalid email format",
		)
		assert.Equal(t, "email", err.Field)
		assert.Equal(t, errortypes.ErrInvalidFormat, err.Code)
		assert.Equal(t, "invalid email format", err.Message)
		assert.Equal(t, "invalid email format", err.Error())
	})

	t.Run("IsError", func(t *testing.T) {
		err := errortypes.NewValidationError(
			"email",
			errortypes.ErrInvalidFormat,
			"invalid email format",
		)
		assert.True(t, errortypes.IsError(err))
		assert.False(t, errortypes.IsError(assert.AnError))
	})
}

func TestMultiError(t *testing.T) {
	t.Run("NewMultiError", func(t *testing.T) {
		me := errortypes.NewMultiError()
		assert.NotNil(t, me)
		assert.Empty(t, me.Errors)
	})

	t.Run("Add and HasErrors", func(t *testing.T) {
		me := errortypes.NewMultiError()
		assert.False(t, me.HasErrors())

		me.Add("email", errortypes.ErrInvalidFormat, "invalid email")
		assert.True(t, me.HasErrors())
		assert.Len(t, me.Errors, 1)
		assert.Equal(t, "email", me.Errors[0].Field)
	})

	t.Run("WithPrefix", func(t *testing.T) {
		me := errortypes.NewMultiError()
		child := me.WithPrefix("user")
		child.Add("email", errortypes.ErrInvalidFormat, "invalid email")

		assert.Equal(t, "user.email", me.Errors[0].Field)
	})

	t.Run("WithIndex", func(t *testing.T) {
		me := errortypes.NewMultiError()
		child := me.WithIndex("items", 0)
		child.Add("name", errortypes.ErrRequired, "required")

		assert.Equal(t, "items[0].name", me.Errors[0].Field)
	})

	t.Run("Nested Prefixes", func(t *testing.T) {
		me := errortypes.NewMultiError()
		level1 := me.WithPrefix("user")
		level2 := level1.WithPrefix("address")
		level3 := level2.WithIndex("phones", 0)

		level3.Add("number", errortypes.ErrInvalidFormat, "invalid phone number")
		assert.Equal(t, "user.address.phones[0].number", me.Errors[0].Field)
	})

	t.Run("Error String", func(t *testing.T) {
		me := errortypes.NewMultiError()
		me.Add("email", errortypes.ErrInvalidFormat, "invalid email")
		me.Add("password", errortypes.ErrRequired, "required")

		expected := "validation failed:\n- invalid email\n- required"
		assert.Equal(t, expected, me.Error())
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		me := errortypes.NewMultiError()
		me.Add("email", errortypes.ErrInvalidFormat, "invalid email")

		data, err := json.Marshal(me)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		rErrors, ok := result["errors"].([]any)
		assert.True(t, ok)
		assert.Len(t, rErrors, 1)
	})

	t.Run("ToJSON", func(t *testing.T) {
		me := errortypes.NewMultiError()
		me.Add("email", errortypes.ErrInvalidFormat, "invalid email")

		jsonStr := me.ToJSON()
		assert.Contains(t, jsonStr, "errors")
		assert.Contains(t, jsonStr, "email")
	})
}

func TestBusinessError(t *testing.T) {
	t.Run("NewBusinessError", func(t *testing.T) {
		err := errortypes.NewBusinessError("invalid operation")
		assert.Equal(t, errortypes.ErrBusinessLogic, err.Code)
		assert.Equal(t, "invalid operation", err.Message)
	})

	t.Run("WithParam", func(t *testing.T) {
		err := errortypes.NewBusinessError("invalid operation").
			WithParam("key", "value")
		assert.Equal(t, "value", err.Params["key"])
	})

	t.Run("WithInternal", func(t *testing.T) {
		internal := assert.AnError
		err := errortypes.NewBusinessError("invalid operation").
			WithInternal(internal)
		assert.Equal(t, internal, err.Internal)
	})

	t.Run("IsBusinessError", func(t *testing.T) {
		err := errortypes.NewBusinessError("invalid operation")
		assert.True(t, errortypes.IsBusinessError(err))
		assert.False(t, errortypes.IsBusinessError(assert.AnError))
	})
}

func TestDatabaseError(t *testing.T) {
	t.Run("NewDatabaseError", func(t *testing.T) {
		err := errortypes.NewDatabaseError("connection failed")
		assert.Equal(t, errortypes.ErrSystemError, err.Code)
		assert.Equal(t, "connection failed", err.Message)
	})

	t.Run("WithInternal", func(t *testing.T) {
		internal := assert.AnError
		err := errortypes.NewDatabaseError("connection failed").
			WithInternal(internal)
		assert.Equal(t, internal, err.Internal)
	})

	t.Run("IsDatabaseError", func(t *testing.T) {
		err := errortypes.NewDatabaseError("connection failed")
		assert.True(t, errortypes.IsDatabaseError(err))
		assert.False(t, errortypes.IsDatabaseError(assert.AnError))
	})
}

func TestAuthenticationError(t *testing.T) {
	t.Run("NewAuthenticationError", func(t *testing.T) {
		err := errortypes.NewAuthenticationError("invalid credentials")
		assert.Equal(t, errortypes.ErrUnauthorized, err.Code)
		assert.Equal(t, "invalid credentials", err.Message)
	})

	t.Run("WithInternal", func(t *testing.T) {
		internal := assert.AnError
		err := errortypes.NewAuthenticationError("invalid credentials").
			WithInternal(internal)
		assert.Equal(t, internal, err.Internal)
	})

	t.Run("IsAuthenticationError", func(t *testing.T) {
		err := errortypes.NewAuthenticationError("invalid credentials")
		assert.True(t, errortypes.IsAuthenticationError(err))
		assert.False(t, errortypes.IsAuthenticationError(assert.AnError))
	})
}

func TestAuthorizationError(t *testing.T) {
	t.Run("NewAuthorizationError", func(t *testing.T) {
		err := errortypes.NewAuthorizationError("insufficient permissions")
		assert.Equal(t, errortypes.ErrForbidden, err.Code)
		assert.Equal(t, "insufficient permissions", err.Message)
	})

	t.Run("WithInternal", func(t *testing.T) {
		internal := assert.AnError
		err := errortypes.NewAuthorizationError("insufficient permissions").
			WithInternal(internal)
		assert.Equal(t, internal, err.Internal)
	})

	t.Run("IsAuthorizationError", func(t *testing.T) {
		err := errortypes.NewAuthorizationError("insufficient permissions")
		assert.True(t, errortypes.IsAuthorizationError(err))
		assert.False(t, errortypes.IsAuthorizationError(assert.AnError))
	})
}

func TestNotFoundError(t *testing.T) {
	t.Run("NewNotFoundError", func(t *testing.T) {
		err := errortypes.NewNotFoundError("resource not found")
		assert.Equal(t, errortypes.ErrNotFound, err.Code)
		assert.Equal(t, "resource not found", err.Message)
	})

	t.Run("IsNotFoundError", func(t *testing.T) {
		err := errortypes.NewNotFoundError("resource not found")
		assert.True(t, errortypes.IsNotFoundError(err))
		assert.False(t, errortypes.IsNotFoundError(assert.AnError))
	})

	t.Run("WithInternal", func(t *testing.T) {
		internal := assert.AnError
		err := errortypes.NewNotFoundError("resource not found").
			WithInternal(internal)
		assert.Equal(t, internal, err.Internal)
	})
}

func TestRateLimitError(t *testing.T) {
	t.Run("NewRateLimitError", func(t *testing.T) {
		err := errortypes.NewRateLimitError("api", "too many requests")
		assert.Equal(t, errortypes.ErrTooManyRequests, err.Code)
		assert.Equal(t, "api", err.Field)
		assert.Equal(t, "too many requests", err.Message)
	})

	t.Run("WithInternal", func(t *testing.T) {
		internal := assert.AnError
		err := errortypes.NewRateLimitError("api", "too many requests").
			WithInternal(internal)
		assert.Equal(t, internal, err.Internal)
	})

	t.Run("IsRateLimitError", func(t *testing.T) {
		err := errortypes.NewRateLimitError("api", "too many requests")
		assert.True(t, errortypes.IsRateLimitError(err))
		assert.False(t, errortypes.IsRateLimitError(assert.AnError))
	})
}

func TestInferErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected errortypes.ErrorCode
	}{
		{
			name:     "required error",
			err:      val.ErrRequired,
			expected: errortypes.ErrInvalid,
		},
		{
			name:     "length error",
			err:      val.ErrLengthOutOfRange,
			expected: errortypes.ErrInvalidLength,
		},
		{
			name:     "format error",
			err:      val.NewError("format", "invalid format"),
			expected: errortypes.ErrInvalidFormat,
		},
		{
			name:     "match error",
			err:      val.NewError("match", "no match"),
			expected: errortypes.ErrInvalidFormat,
		},
		{
			name:     "rate limit error",
			err:      val.NewError("rate limit", "rate limit exceeded"),
			expected: errortypes.ErrTooManyRequests,
		},
		{
			name:     "default error",
			err:      val.NewError("other", "other error"),
			expected: errortypes.ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := errortypes.InferErrorCode(tt.err)
			assert.Equal(t, tt.expected, code)
		})
	}
}

func TestFromOzzoErrors(t *testing.T) {
	t.Run("Basic Validation Errors", func(t *testing.T) {
		ozzoErrors := val.Errors{
			"email":    val.ErrRequired,
			"password": val.ErrLengthOutOfRange,
		}

		me := errortypes.NewMultiError()
		errortypes.FromOzzoErrors(ozzoErrors, me)

		assert.Len(t, me.Errors, 2)
		assert.Contains(t, []string{"email", "password"}, me.Errors[0].Field)
		assert.Contains(t, []string{"email", "password"}, me.Errors[1].Field)
	})

	t.Run("With Prefix", func(t *testing.T) {
		ozzoErrors := val.Errors{
			"email": val.ErrRequired,
		}

		me := errortypes.NewMultiError()
		child := me.WithPrefix("user")
		errortypes.FromOzzoErrors(ozzoErrors, child)

		require.True(t, me.HasErrors(), "MultiError should have errors")
		require.NotEmpty(t, me.Errors, "Errors slice should not be empty")
		assert.Equal(t, "user.email", me.Errors[0].Field)
	})

	t.Run("Nested Prefixes", func(t *testing.T) {
		ozzoErrors := val.Errors{
			"street": val.ErrRequired,
		}

		me := errortypes.NewMultiError()
		level1 := me.WithPrefix("user")
		level2 := level1.WithPrefix("address")
		errortypes.FromOzzoErrors(ozzoErrors, level2)

		require.True(t, me.HasErrors(), "MultiError should have errors")
		require.NotEmpty(t, me.Errors, "Errors slice should not be empty")
		assert.Equal(t, "user.address.street", me.Errors[0].Field)
	})
}

func TestUnwrap(t *testing.T) {
	internalErr := assert.AnError

	t.Run("Error Unwrap", func(t *testing.T) {
		err := errortypes.NewValidationError("field", errortypes.ErrRequired, "required")
		err.Internal = internalErr
		assert.Equal(t, internalErr, err.Unwrap())
	})

	t.Run("BusinessError Unwrap", func(t *testing.T) {
		err := errortypes.NewBusinessError("failed").WithInternal(internalErr)
		assert.Equal(t, internalErr, err.Unwrap())
	})

	t.Run("DatabaseError Unwrap", func(t *testing.T) {
		err := errortypes.NewDatabaseError("failed").WithInternal(internalErr)
		assert.Equal(t, internalErr, err.Unwrap())
	})

	t.Run("AuthenticationError Unwrap", func(t *testing.T) {
		err := errortypes.NewAuthenticationError("failed").WithInternal(internalErr)
		assert.Equal(t, internalErr, err.Unwrap())
	})

	t.Run("NotFoundError Unwrap", func(t *testing.T) {
		err := errortypes.NewNotFoundError("failed").WithInternal(internalErr)
		assert.Equal(t, internalErr, err.Unwrap())
	})
}

func TestMultiErrorWithLimit(t *testing.T) {
	t.Run("NewMultiErrorWithLimit", func(t *testing.T) {
		me := errortypes.NewMultiErrorWithLimit(3)
		assert.NotNil(t, me)
		assert.False(t, me.IsFull())
	})

	t.Run("Respects Limit", func(t *testing.T) {
		me := errortypes.NewMultiErrorWithLimit(2)
		me.Add("field1", errortypes.ErrRequired, "error 1")
		me.Add("field2", errortypes.ErrRequired, "error 2")
		me.Add("field3", errortypes.ErrRequired, "error 3")

		assert.Len(t, me.Errors, 2)
		assert.True(t, me.IsFull())
	})

	t.Run("Limit With Prefix", func(t *testing.T) {
		me := errortypes.NewMultiErrorWithLimit(2)
		child := me.WithPrefix("user")
		child.Add("field1", errortypes.ErrRequired, "error 1")
		child.Add("field2", errortypes.ErrRequired, "error 2")
		child.Add("field3", errortypes.ErrRequired, "error 3")

		assert.Len(t, me.Errors, 2)
		assert.True(t, me.IsFull())
	})

	t.Run("No Limit When Zero", func(t *testing.T) {
		me := errortypes.NewMultiError()
		for range 100 {
			me.Add("field", errortypes.ErrRequired, "error")
		}
		assert.Len(t, me.Errors, 100)
		assert.False(t, me.IsFull())
	})
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil error", nil, 200},
		{"MultiError", errortypes.NewMultiError(), 422},
		{"NotFoundError", errortypes.NewNotFoundError("not found"), 404},
		{"AuthenticationError", errortypes.NewAuthenticationError("unauthorized"), 401},
		{"AuthorizationError", errortypes.NewAuthorizationError("forbidden"), 403},
		{"RateLimitError", errortypes.NewRateLimitError("api", "too many"), 429},
		{"BusinessError", errortypes.NewBusinessError("invalid"), 422},
		{"DatabaseError", errortypes.NewDatabaseError("failed"), 500},
		{"unknown error", assert.AnError, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := errortypes.HTTPStatus(tt.err)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestHTTPStatusWithCode(t *testing.T) {
	tests := []struct {
		code     errortypes.ErrorCode
		expected int
	}{
		{errortypes.ErrRequired, 422},
		{errortypes.ErrInvalid, 422},
		{errortypes.ErrDuplicate, 409},
		{errortypes.ErrAlreadyExists, 409},
		{errortypes.ErrNotFound, 404},
		{errortypes.ErrUnauthorized, 401},
		{errortypes.ErrForbidden, 403},
		{errortypes.ErrTooManyRequests, 429},
		{errortypes.ErrBusinessLogic, 422},
		{errortypes.ErrVersionMismatch, 409},
		{errortypes.ErrSystemError, 500},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			status := errortypes.HTTPStatusWithCode(tt.code)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestMultiError_AddOzzoError(t *testing.T) {
	t.Run("Nil Error", func(t *testing.T) {
		me := errortypes.NewMultiError()
		result := me.AddOzzoError(nil)
		assert.False(t, result)
		assert.False(t, me.HasErrors())
	})

	t.Run("Ozzo Validation Error", func(t *testing.T) {
		me := errortypes.NewMultiError()
		ozzoErr := val.Errors{"email": val.ErrRequired}
		result := me.AddOzzoError(ozzoErr)
		assert.True(t, result)
		assert.True(t, me.HasErrors())
		assert.Equal(t, "email", me.Errors[0].Field)
	})

	t.Run("Non-Validation Error", func(t *testing.T) {
		me := errortypes.NewMultiError()
		result := me.AddOzzoError(assert.AnError)
		assert.False(t, result)
		assert.False(t, me.HasErrors())
	})

	t.Run("With Prefix", func(t *testing.T) {
		me := errortypes.NewMultiError()
		child := me.WithPrefix("user")
		ozzoErr := val.Errors{"email": val.ErrRequired}
		child.AddOzzoError(ozzoErr)
		assert.Equal(t, "user.email", me.Errors[0].Field)
	})

	t.Run("Multiple Fields", func(t *testing.T) {
		me := errortypes.NewMultiError()
		ozzoErr := val.Errors{
			"email":    val.ErrRequired,
			"password": val.ErrLengthOutOfRange,
		}
		result := me.AddOzzoError(ozzoErr)
		assert.True(t, result)
		assert.Len(t, me.Errors, 2)
	})
}

func TestErrorContext(t *testing.T) {
	t.Run("NewErrorContext", func(t *testing.T) {
		ctx := errortypes.NewErrorContext()
		assert.NotNil(t, ctx)
	})

	t.Run("WithRequestID", func(t *testing.T) {
		ctx := errortypes.NewErrorContext().WithRequestID("req-123")
		assert.Equal(t, "req-123", ctx.RequestID)
	})

	t.Run("WithUserID", func(t *testing.T) {
		ctx := errortypes.NewErrorContext().WithUserID("user-456")
		assert.Equal(t, "user-456", ctx.UserID)
	})

	t.Run("WithTraceID", func(t *testing.T) {
		ctx := errortypes.NewErrorContext().WithTraceID("trace-789")
		assert.Equal(t, "trace-789", ctx.TraceID)
	})

	t.Run("WithExtra", func(t *testing.T) {
		ctx := errortypes.NewErrorContext().WithExtra("key", "value")
		assert.Equal(t, "value", ctx.Extra["key"])
	})

	t.Run("LogFields", func(t *testing.T) {
		ctx := errortypes.NewErrorContext().
			WithRequestID("req-123").
			WithUserID("user-456").
			WithExtra("custom", "data")

		fields := ctx.LogFields()
		assert.Equal(t, "req-123", fields["request_id"])
		assert.Equal(t, "user-456", fields["user_id"])
		assert.Equal(t, "data", fields["custom"])
	})
}

func TestStackTraces(t *testing.T) {
	t.Run("Disabled by default", func(t *testing.T) {
		assert.False(t, errortypes.StackTracesEnabled())
	})

	t.Run("Enable and disable", func(t *testing.T) {
		errortypes.EnableStackTraces()
		assert.True(t, errortypes.StackTracesEnabled())

		errortypes.DisableStackTraces()
		assert.False(t, errortypes.StackTracesEnabled())
	})

	t.Run("Captures stack when enabled", func(t *testing.T) {
		errortypes.EnableStackTraces()
		defer errortypes.DisableStackTraces()

		ctx := errortypes.NewErrorContext()
		assert.NotEmpty(t, ctx.Stack)
		assert.NotEmpty(t, ctx.Stack[0].Function)
		assert.NotEmpty(t, ctx.Stack[0].File)
	})

	t.Run("No stack when disabled", func(t *testing.T) {
		errortypes.DisableStackTraces()
		ctx := errortypes.NewErrorContext()
		assert.Empty(t, ctx.Stack)
	})
}

func TestErrorWithContext(t *testing.T) {
	t.Run("BusinessError WithContext", func(t *testing.T) {
		ctx := errortypes.NewErrorContext().WithRequestID("req-123")
		err := errortypes.NewBusinessError("failed").WithContext(ctx)
		assert.Equal(t, "req-123", err.Context.RequestID)
	})

	t.Run("MultiError WithContext", func(t *testing.T) {
		ctx := errortypes.NewErrorContext().WithRequestID("req-456")
		me := errortypes.NewMultiError().WithContext(ctx)
		assert.Equal(t, "req-456", me.Context.RequestID)
	})

	t.Run("NotFoundError WithContext", func(t *testing.T) {
		ctx := errortypes.NewErrorContext().WithUserID("user-789")
		err := errortypes.NewNotFoundError("not found").WithContext(ctx)
		assert.Equal(t, "user-789", err.Context.UserID)
	})
}

func TestLogFields(t *testing.T) {
	t.Run("BaseError LogFields", func(t *testing.T) {
		ctx := errortypes.NewErrorContext().WithRequestID("req-123")
		err := errortypes.NewBusinessError("operation failed").
			WithContext(ctx).
			WithInternal(assert.AnError)

		fields := err.LogFields()
		assert.Equal(t, errortypes.ErrBusinessLogic, fields["error_code"])
		assert.Equal(t, "operation failed", fields["error_message"])
		assert.Equal(t, "req-123", fields["request_id"])
		assert.Equal(t, assert.AnError.Error(), fields["internal_error"])
	})

	t.Run("BusinessError LogFields with params", func(t *testing.T) {
		err := errortypes.NewBusinessError("failed").
			WithParam("entity", "user").
			WithParam("action", "create")

		fields := err.LogFields()
		assert.Equal(t, "user", fields["param_entity"])
		assert.Equal(t, "create", fields["param_action"])
	})

	t.Run("MultiError LogFields", func(t *testing.T) {
		ctx := errortypes.NewErrorContext().WithRequestID("req-789")
		me := errortypes.NewMultiError().WithContext(ctx)
		me.Add("email", errortypes.ErrRequired, "required")
		me.Add("name", errortypes.ErrInvalidLength, "too short")

		fields := me.LogFields()
		assert.Equal(t, 2, fields["error_count"])
		assert.Equal(t, "req-789", fields["request_id"])
		assert.Contains(t, fields["error_fields"], "email")
		assert.Contains(t, fields["error_fields"], "name")
	})

	t.Run("RateLimitError LogFields", func(t *testing.T) {
		err := errortypes.NewRateLimitError("api", "too many requests")
		fields := err.LogFields()
		assert.Equal(t, "api", fields["rate_limit_field"])
	})
}

func TestConflictError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		message    string
		usageStats any
		internal   error
	}{
		{
			name:    "basic conflict error",
			message: "resource is in use",
		},
		{
			name:       "with usage stats",
			message:    "cannot delete",
			usageStats: map[string]int{"shipments": 5},
		},
		{
			name:     "with internal error",
			message:  "conflict occurred",
			internal: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := errortypes.NewConflictError(tt.message)
			assert.Equal(t, errortypes.ErrResourceInUse, err.Code)
			assert.Equal(t, tt.message, err.Error())

			if tt.usageStats != nil {
				err = err.WithUsageStats(tt.usageStats)
				assert.Equal(t, tt.usageStats, err.UsageStats)
			}

			if tt.internal != nil {
				err = err.WithInternal(tt.internal)
				assert.Equal(t, tt.internal, err.Unwrap())
			}
		})
	}
}

func TestIsConflictError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"conflict error", errortypes.NewConflictError("conflict"), true},
		{"non-conflict error", assert.AnError, false},
		{"business error", errortypes.NewBusinessError("fail"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, errortypes.IsConflictError(tt.err))
		})
	}
}

func TestConflictError_LogFields(t *testing.T) {
	t.Parallel()

	t.Run("without usage stats", func(t *testing.T) {
		t.Parallel()
		err := errortypes.NewConflictError("conflict")
		fields := err.LogFields()
		assert.Equal(t, errortypes.ErrResourceInUse, fields["error_code"])
		assert.Equal(t, "conflict", fields["error_message"])
		assert.Nil(t, fields["usage_stats"])
	})

	t.Run("with usage stats", func(t *testing.T) {
		t.Parallel()
		stats := map[string]int{"refs": 3}
		err := errortypes.NewConflictError("in use").WithUsageStats(stats)
		fields := err.LogFields()
		assert.Equal(t, stats, fields["usage_stats"])
	})
}

func TestConflictError_WithContext(t *testing.T) {
	t.Parallel()

	ctx := errortypes.NewErrorContext().WithRequestID("req-conflict")
	err := errortypes.NewConflictError("conflict").WithContext(ctx)
	assert.Equal(t, "req-conflict", err.Context.RequestID)
}

func TestNewValidationErrorWithPriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    string
		code     errortypes.ErrorCode
		message  string
		priority errortypes.ValidationPriority
	}{
		{"high priority", "email", errortypes.ErrRequired, "required", errortypes.PriorityHigh},
		{
			"medium priority",
			"name",
			errortypes.ErrInvalidLength,
			"too short",
			errortypes.PriorityMedium,
		},
		{"low priority", "notes", errortypes.ErrInvalid, "invalid", errortypes.PriorityLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := errortypes.NewValidationErrorWithPriority(
				tt.field,
				tt.code,
				tt.message,
				tt.priority,
			)
			assert.Equal(t, tt.field, err.Field)
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.priority, err.Priority)
			assert.Equal(t, tt.message, err.Error())
			assert.Equal(t, tt.code, err.GetCode())
		})
	}
}

func TestIsMultiError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"multi error", errortypes.NewMultiError(), true},
		{"non-multi error", assert.AnError, false},
		{
			"validation error",
			errortypes.NewValidationError("f", errortypes.ErrRequired, "r"),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, errortypes.IsMultiError(tt.err))
		})
	}
}

func TestMultiError_MarshalJSON_NilAndEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty multi error", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		data, err := me.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, "null", string(data))
	})

	t.Run("nil multi error", func(t *testing.T) {
		t.Parallel()
		var me *errortypes.MultiError
		data, err := me.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, "null", string(data))
	})
}

func TestMultiError_ErrorEmpty(t *testing.T) {
	t.Parallel()

	me := errortypes.NewMultiError()
	assert.Equal(t, "", me.Error())
}

func TestMultiError_AddError(t *testing.T) {
	t.Parallel()

	t.Run("nil error is ignored", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		me.AddError(nil)
		assert.False(t, me.HasErrors())
	})

	t.Run("adds error to root", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		e := &errortypes.Error{
			Field:   "email",
			Code:    errortypes.ErrRequired,
			Message: "required",
		}
		me.AddError(e)
		assert.True(t, me.HasErrors())
		assert.Equal(t, "email", me.Errors[0].Field)
	})

	t.Run("respects limit via AddError", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiErrorWithLimit(1)
		me.AddError(&errortypes.Error{Field: "a", Code: errortypes.ErrRequired, Message: "err1"})
		me.AddError(&errortypes.Error{Field: "b", Code: errortypes.ErrRequired, Message: "err2"})
		assert.Len(t, me.Errors, 1)
	})

	t.Run("with prefix", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		child := me.WithPrefix("address")
		child.AddError(
			&errortypes.Error{Field: "city", Code: errortypes.ErrRequired, Message: "required"},
		)
		assert.Equal(t, "address.city", me.Errors[0].Field)
	})

	t.Run("empty field with prefix", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		child := me.WithPrefix("root")
		child.AddError(
			&errortypes.Error{Field: "", Code: errortypes.ErrRequired, Message: "required"},
		)
		assert.Equal(t, "", me.Errors[0].Field)
	})
}

func TestMultiError_SetPriority(t *testing.T) {
	t.Parallel()

	t.Run("sets priority on root", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		me.SetPriority(errortypes.PriorityLow)
		me.Add("field", errortypes.ErrRequired, "required")
		assert.Equal(t, errortypes.PriorityLow, me.Errors[0].Priority)
	})

	t.Run("child propagates to parent", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		child := me.WithPrefix("user")
		child.SetPriority(errortypes.PriorityMedium)
		child.Add("name", errortypes.ErrRequired, "required")
		assert.Equal(t, errortypes.PriorityMedium, me.Errors[0].Priority)
	})
}

func TestMultiError_AddWithPriority(t *testing.T) {
	t.Parallel()

	t.Run("explicit priority overrides current", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		me.AddWithPriority("field", errortypes.ErrRequired, "required", errortypes.PriorityLow)
		assert.Equal(t, errortypes.PriorityLow, me.Errors[0].Priority)
	})

	t.Run("empty priority uses current", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		me.SetPriority(errortypes.PriorityMedium)
		me.AddWithPriority("field", errortypes.ErrRequired, "required", "")
		assert.Equal(t, errortypes.PriorityMedium, me.Errors[0].Priority)
	})

	t.Run("respects limit", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiErrorWithLimit(1)
		me.AddWithPriority("a", errortypes.ErrRequired, "err1", errortypes.PriorityHigh)
		me.AddWithPriority("b", errortypes.ErrRequired, "err2", errortypes.PriorityHigh)
		assert.Len(t, me.Errors, 1)
	})

	t.Run("with prefix and empty field", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		child := me.WithPrefix("item")
		child.AddWithPriority("", errortypes.ErrInvalid, "invalid item", errortypes.PriorityHigh)
		assert.Equal(t, "item", me.Errors[0].Field)
	})
}

func TestBusinessError_ErrorWithDetails(t *testing.T) {
	t.Parallel()

	t.Run("without details", func(t *testing.T) {
		t.Parallel()
		err := errortypes.NewBusinessError("operation failed")
		assert.Equal(t, "operation failed", err.Error())
	})

	t.Run("with details", func(t *testing.T) {
		t.Parallel()
		err := &errortypes.BusinessError{
			BaseError: errortypes.BaseError{
				Code:    errortypes.ErrBusinessLogic,
				Message: "operation failed",
			},
			Details: "insufficient funds",
		}
		assert.Equal(t, "operation failed: insufficient funds", err.Error())
	})
}

func TestBaseError_GetCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      errortypes.Errorable
		expected errortypes.ErrorCode
	}{
		{"database error", errortypes.NewDatabaseError("fail"), errortypes.ErrSystemError},
		{"auth error", errortypes.NewAuthenticationError("fail"), errortypes.ErrUnauthorized},
		{"authz error", errortypes.NewAuthorizationError("fail"), errortypes.ErrForbidden},
		{"not found error", errortypes.NewNotFoundError("fail"), errortypes.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.err.GetCode())
		})
	}
}

func TestErrorContext_WithSpanID(t *testing.T) {
	t.Parallel()

	ctx := errortypes.NewErrorContext().WithSpanID("span-abc")
	assert.Equal(t, "span-abc", ctx.SpanID)
}

func TestErrorContext_LogFieldsWithSpanAndTrace(t *testing.T) {
	t.Parallel()

	ctx := errortypes.NewErrorContext().
		WithTraceID("trace-123").
		WithSpanID("span-456")

	fields := ctx.LogFields()
	assert.Equal(t, "trace-123", fields["trace_id"])
	assert.Equal(t, "span-456", fields["span_id"])
}

func TestMultiError_IsError(t *testing.T) {
	t.Parallel()

	me := errortypes.NewMultiError()
	me.Add("field", errortypes.ErrRequired, "required")
	assert.True(t, errortypes.IsError(me))
}

func TestRateLimitError_WithContext(t *testing.T) {
	t.Parallel()

	ctx := errortypes.NewErrorContext().WithRequestID("req-rl")
	err := errortypes.NewRateLimitError("api", "slow down").WithContext(ctx)
	assert.Equal(t, "req-rl", err.Context.RequestID)
}

func TestDatabaseError_WithContext(t *testing.T) {
	t.Parallel()

	ctx := errortypes.NewErrorContext().WithUserID("user-db")
	err := errortypes.NewDatabaseError("query failed").WithContext(ctx)
	assert.Equal(t, "user-db", err.Context.UserID)
}

func TestAuthenticationError_WithContext(t *testing.T) {
	t.Parallel()

	ctx := errortypes.NewErrorContext().WithTraceID("trace-auth")
	err := errortypes.NewAuthenticationError("bad token").WithContext(ctx)
	assert.Equal(t, "trace-auth", err.Context.TraceID)
}

func TestAuthorizationError_WithContext(t *testing.T) {
	t.Parallel()

	ctx := errortypes.NewErrorContext().WithSpanID("span-authz")
	err := errortypes.NewAuthorizationError("denied").WithContext(ctx)
	assert.Equal(t, "span-authz", err.Context.SpanID)
}

func TestNotFoundError_WithContext(t *testing.T) {
	t.Parallel()

	ctx := errortypes.NewErrorContext().WithExtra("entity", "user")
	err := errortypes.NewNotFoundError("user not found").WithContext(ctx)
	assert.Equal(t, "user", err.Context.Extra["entity"])
}

func TestInferErrorCode_OzzoSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected errortypes.ErrorCode
	}{
		{"min greater equal", val.ErrMinGreaterEqualThanRequired, errortypes.ErrInvalid},
		{"max less equal", val.ErrMaxLessEqualThanRequired, errortypes.ErrInvalid},
		{"nil or not empty", val.ErrNilOrNotEmpty, errortypes.ErrInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, errortypes.InferErrorCode(tt.err))
		})
	}
}

func TestMultiError_ToJSON_Empty(t *testing.T) {
	t.Parallel()

	me := errortypes.NewMultiError()
	result := me.ToJSON()
	assert.Equal(t, "null", result)
}
