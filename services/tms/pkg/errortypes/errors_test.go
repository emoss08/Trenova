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
