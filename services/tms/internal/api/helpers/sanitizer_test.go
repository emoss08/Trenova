package helpers_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSanitizer(t *testing.T) {
	t.Parallel()

	t.Run("creates sanitizer with debug mode", func(t *testing.T) {
		t.Parallel()
		s := helpers.NewSanitizer(true)
		assert.NotNil(t, s)
	})

	t.Run("creates sanitizer without debug mode", func(t *testing.T) {
		t.Parallel()
		s := helpers.NewSanitizer(false)
		assert.NotNil(t, s)
	})
}

func TestSanitizer_SanitizeMessage(t *testing.T) {
	t.Parallel()

	t.Run("returns original message for non-internal errors", func(t *testing.T) {
		t.Parallel()
		s := helpers.NewSanitizer(false)
		err := errors.New("validation failed")

		result := s.SanitizeMessage(err, helpers.ProblemTypeValidation)
		assert.Equal(t, "validation failed", result)
	})

	t.Run("hides internal error in production", func(t *testing.T) {
		t.Parallel()
		s := helpers.NewSanitizer(false)
		err := errors.New("database connection failed: password incorrect")

		result := s.SanitizeMessage(err, helpers.ProblemTypeInternal)
		assert.Equal(t, "An unexpected error occurred. Please try again later.", result)
	})

	t.Run("hides database error in production", func(t *testing.T) {
		t.Parallel()
		s := helpers.NewSanitizer(false)
		err := errors.New("SQLSTATE 42P01: relation does not exist")

		result := s.SanitizeMessage(err, helpers.ProblemTypeDatabase)
		assert.Equal(t, "An unexpected error occurred. Please try again later.", result)
	})

	t.Run("shows internal error in debug mode", func(t *testing.T) {
		t.Parallel()
		s := helpers.NewSanitizer(true)
		err := errors.New("database connection failed: password incorrect")

		result := s.SanitizeMessage(err, helpers.ProblemTypeInternal)
		assert.Equal(t, "database connection failed: password incorrect", result)
	})

	t.Run("shows database error in debug mode", func(t *testing.T) {
		t.Parallel()
		s := helpers.NewSanitizer(true)
		err := errors.New("SQLSTATE 42P01: relation does not exist")

		result := s.SanitizeMessage(err, helpers.ProblemTypeDatabase)
		assert.Equal(t, "SQLSTATE 42P01: relation does not exist", result)
	})

	t.Run("shows business error regardless of debug mode", func(t *testing.T) {
		t.Parallel()
		s := helpers.NewSanitizer(false)
		err := errors.New("insufficient funds")

		result := s.SanitizeMessage(err, helpers.ProblemTypeBusiness)
		assert.Equal(t, "insufficient funds", result)
	})
}

func TestSanitizer_ExtractErrors_MultiError(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	t.Run("extracts errors from MultiError", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		me.Add("email", errortypes.ErrRequired, "Email is required")
		me.Add("password", errortypes.ErrInvalidLength, "Password too short")

		result := s.ExtractErrors(me)

		require.Len(t, result, 2)
		assert.Equal(t, "email", result[0].Field)
		assert.Equal(t, "Email is required", result[0].Message)
		assert.Equal(t, string(errortypes.ErrRequired), result[0].Code)
		assert.Equal(t, "body", result[0].Location)
	})

	t.Run("handles nested prefixes", func(t *testing.T) {
		t.Parallel()
		me := errortypes.NewMultiError()
		child := me.WithPrefix("user").WithPrefix("address")
		child.Add("street", errortypes.ErrRequired, "Street is required")

		result := s.ExtractErrors(me)

		require.Len(t, result, 1)
		assert.Equal(t, "user.address.street", result[0].Field)
	})
}

func TestSanitizer_ExtractErrors_ValidationError(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	t.Run("extracts single validation error", func(t *testing.T) {
		t.Parallel()
		err := errortypes.NewValidationError(
			"email",
			errortypes.ErrInvalidFormat,
			"Invalid email format",
		)

		result := s.ExtractErrors(err)

		require.Len(t, result, 1)
		assert.Equal(t, "email", result[0].Field)
		assert.Equal(t, "Invalid email format", result[0].Message)
		assert.Equal(t, string(errortypes.ErrInvalidFormat), result[0].Code)
		assert.Equal(t, "body", result[0].Location)
	})
}

func TestSanitizer_ExtractErrors_ValidatorErrors(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	type testStruct struct {
		Email    string `validate:"required,email"`
		Password string `validate:"required,min=8"`
		Age      int    `validate:"min=0,max=150"`
	}

	validate := validator.New()
	err := validate.Struct(testStruct{})

	result := s.ExtractErrors(err)

	require.NotEmpty(t, result)

	fieldNames := make([]string, len(result))
	for i, ve := range result {
		fieldNames[i] = ve.Field
		assert.Equal(t, "body", ve.Location)
		assert.NotEmpty(t, ve.Code)
	}

	assert.Contains(t, fieldNames, "email")
	assert.Contains(t, fieldNames, "password")
}

func TestSanitizer_ExtractErrors_ValidatorMessages(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	tests := []struct {
		name            string
		validation      string
		expectedMessage string
	}{
		{
			name:            "required field",
			validation:      "required",
			expectedMessage: "This field is required",
		},
		{
			name:            "email field",
			validation:      "email",
			expectedMessage: "Must be a valid email address",
		},
		{
			name:            "min constraint",
			validation:      "min=5",
			expectedMessage: "Value is below the minimum allowed",
		},
		{
			name:            "max constraint",
			validation:      "max=10",
			expectedMessage: "Value exceeds the maximum allowed",
		},
		{
			name:            "uuid field",
			validation:      "uuid",
			expectedMessage: "Must be a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			type dynamicStruct struct {
				Field string `validate:"required"`
			}

			validate := validator.New()

			switch tt.validation {
			case "required":
				err := validate.Struct(dynamicStruct{})
				result := s.ExtractErrors(err)
				require.NotEmpty(t, result)
				assert.Equal(t, tt.expectedMessage, result[0].Message)
			case "email":
				type emailStruct struct {
					Email string `validate:"email"`
				}
				err := validate.Struct(emailStruct{Email: "invalid"})
				result := s.ExtractErrors(err)
				require.NotEmpty(t, result)
				assert.Equal(t, tt.expectedMessage, result[0].Message)
			case "uuid":
				type uuidStruct struct {
					ID string `validate:"uuid"`
				}
				err := validate.Struct(uuidStruct{ID: "not-a-uuid"})
				result := s.ExtractErrors(err)
				require.NotEmpty(t, result)
				assert.Equal(t, tt.expectedMessage, result[0].Message)
			}
		})
	}
}

func TestSanitizer_ExtractErrors_BusinessError(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	t.Run("extracts business error without details", func(t *testing.T) {
		t.Parallel()
		err := errortypes.NewBusinessError("Insufficient funds")

		result := s.ExtractErrors(err)

		require.Len(t, result, 1)
		assert.Equal(t, "business", result[0].Field)
		assert.Equal(t, "Insufficient funds", result[0].Message)
		assert.Equal(t, string(errortypes.ErrBusinessLogic), result[0].Code)
		assert.Equal(t, "business", result[0].Location)
	})

	t.Run("extracts business error with details", func(t *testing.T) {
		t.Parallel()
		err := errortypes.NewBusinessError("Payment failed")
		err.Details = "Card declined"

		result := s.ExtractErrors(err)

		require.Len(t, result, 1)
		assert.Equal(t, "Payment failed: Card declined", result[0].Message)
	})
}

func TestSanitizer_ExtractErrors_RateLimitError(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	t.Run("extracts rate limit error", func(t *testing.T) {
		t.Parallel()
		err := errortypes.NewRateLimitError("api/login", "Too many login attempts")

		result := s.ExtractErrors(err)

		require.Len(t, result, 1)
		assert.Equal(t, "api/login", result[0].Field)
		assert.Equal(t, "Too many login attempts", result[0].Message)
		assert.Equal(t, string(errortypes.ErrTooManyRequests), result[0].Code)
		assert.Equal(t, "rate-limit", result[0].Location)
	})
}

func TestSanitizer_ExtractErrors_UnknownError(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	t.Run("returns nil for unknown error types", func(t *testing.T) {
		t.Parallel()
		err := errors.New("some generic error")

		result := s.ExtractErrors(err)
		assert.Nil(t, result)
	})
}

func TestSanitizer_ExtractErrors_FieldNameFormatting(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	type testStruct struct {
		FirstName string `validate:"required"`
	}

	validate := validator.New()
	err := validate.Struct(testStruct{})

	result := s.ExtractErrors(err)

	require.NotEmpty(t, result)
	assert.Equal(t, "firstName", result[0].Field)
}

func TestSanitizer_ExtractErrors_OneOfValidation(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	type testStruct struct {
		Status string `validate:"oneof=pending active completed"`
	}

	validate := validator.New()
	err := validate.Struct(testStruct{Status: "invalid"})

	result := s.ExtractErrors(err)

	require.NotEmpty(t, result)
	assert.Contains(t, result[0].Message, "Must be one of:")
	assert.Contains(t, result[0].Message, "pending active completed")
}

func TestSanitizer_ExtractErrors_LenValidation(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	type testStruct struct {
		Code string `validate:"len=6"`
	}

	validate := validator.New()
	err := validate.Struct(testStruct{Code: "123"})

	result := s.ExtractErrors(err)

	require.NotEmpty(t, result)
	assert.Equal(t, "Value must be exactly 6 characters", result[0].Message)
}

func TestSanitizer_ExtractErrors_UnknownValidationTag(t *testing.T) {
	t.Parallel()

	s := helpers.NewSanitizer(false)

	type testStruct struct {
		Value string `validate:"alphanum"`
	}

	validate := validator.New()
	err := validate.Struct(testStruct{Value: "not-alphanum!"})

	result := s.ExtractErrors(err)

	require.NotEmpty(t, result)
	assert.Contains(t, result[0].Message, "Validation failed on 'alphanum' constraint")
}

func TestSanitizer_ExtractUsageStats_ConflictError(t *testing.T) {
	t.Parallel()

	sanitizer := helpers.NewSanitizer(true)

	t.Run("returns usage stats from conflict error", func(t *testing.T) {
		t.Parallel()
		stats := map[string]int{"references": 5}
		err := errortypes.NewConflictError("in use").WithUsageStats(stats)

		result := sanitizer.ExtractUsageStats(err)

		require.NotNil(t, result)
		assert.Equal(t, stats, result)
	})

	t.Run("returns nil for non-conflict error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("some error")

		result := sanitizer.ExtractUsageStats(err)

		assert.Nil(t, result)
	})

	t.Run("returns nil for nil usage stats", func(t *testing.T) {
		t.Parallel()
		err := errortypes.NewConflictError("conflict without stats")

		result := sanitizer.ExtractUsageStats(err)

		assert.Nil(t, result)
	})

	t.Run("returns usage stats from wrapped conflict error", func(t *testing.T) {
		t.Parallel()
		stats := []string{"table_a", "table_b"}
		conflictErr := errortypes.NewConflictError("used").WithUsageStats(stats)
		wrapped := errors.Join(errors.New("wrapper"), conflictErr)

		result := sanitizer.ExtractUsageStats(wrapped)

		require.NotNil(t, result)
		assert.Equal(t, stats, result)
	})
}

func TestSanitizer_ExtractErrors_ConflictError(t *testing.T) {
	t.Parallel()

	sanitizer := helpers.NewSanitizer(true)

	err := errortypes.NewConflictError("resource is in use")
	valErrors := sanitizer.ExtractErrors(err)

	assert.Nil(t, valErrors)
}
