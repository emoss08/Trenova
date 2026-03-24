package helpers_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestClassifierFunc(t *testing.T) {
	t.Parallel()

	t.Run("implements ErrorClassifier interface", func(t *testing.T) {
		t.Parallel()
		classifier := helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
			return helpers.ProblemTypeValidation, true
		})

		problemType, ok := classifier.Classify(errors.New("test"))
		assert.True(t, ok)
		assert.Equal(t, helpers.ProblemTypeValidation, problemType)
	})

	t.Run("returns false when not matched", func(t *testing.T) {
		t.Parallel()
		classifier := helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
			return "", false
		})

		_, ok := classifier.Classify(errors.New("test"))
		assert.False(t, ok)
	})
}

func TestChainClassifier(t *testing.T) {
	t.Parallel()

	t.Run("returns first matching classification", func(t *testing.T) {
		t.Parallel()
		first := helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
			return "", false
		})
		second := helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
			return helpers.ProblemTypeValidation, true
		})
		third := helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
			return helpers.ProblemTypeBusiness, true
		})

		chain := helpers.NewChainClassifier(first, second, third)
		result := chain.Classify(errors.New("test"))

		assert.Equal(t, helpers.ProblemTypeValidation, result)
	})

	t.Run("returns internal when no match", func(t *testing.T) {
		t.Parallel()
		noMatch := helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
			return "", false
		})

		chain := helpers.NewChainClassifier(noMatch)
		result := chain.Classify(errors.New("test"))

		assert.Equal(t, helpers.ProblemTypeInternal, result)
	})

	t.Run("Register adds classifier", func(t *testing.T) {
		t.Parallel()
		chain := helpers.NewChainClassifier()

		chain.Register(helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
			if err.Error() == "custom" {
				return helpers.ProblemTypeBusiness, true
			}
			return "", false
		}))

		result := chain.Classify(errors.New("custom"))
		assert.Equal(t, helpers.ProblemTypeBusiness, result)
	})
}

func TestNewDefaultClassifier(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	tests := []struct {
		name         string
		err          error
		expectedType helpers.ProblemType
	}{
		{
			name:         "MultiError classifies as validation",
			err:          errortypes.NewMultiError(),
			expectedType: helpers.ProblemTypeValidation,
		},
		{
			name: "ValidationError classifies as validation",
			err: errortypes.NewValidationError(
				"field",
				errortypes.ErrRequired,
				"required",
			),
			expectedType: helpers.ProblemTypeValidation,
		},
		{
			name:         "BusinessError classifies as business",
			err:          errortypes.NewBusinessError("business rule violated"),
			expectedType: helpers.ProblemTypeBusiness,
		},
		{
			name:         "DatabaseError classifies as database",
			err:          errortypes.NewDatabaseError("connection failed"),
			expectedType: helpers.ProblemTypeDatabase,
		},
		{
			name:         "AuthenticationError classifies as authentication",
			err:          errortypes.NewAuthenticationError("invalid token"),
			expectedType: helpers.ProblemTypeAuthentication,
		},
		{
			name:         "AuthorizationError classifies as authorization",
			err:          errortypes.NewAuthorizationError("forbidden"),
			expectedType: helpers.ProblemTypeAuthorization,
		},
		{
			name:         "NotFoundError classifies as not found",
			err:          errortypes.NewNotFoundError("resource not found"),
			expectedType: helpers.ProblemTypeNotFound,
		},
		{
			name:         "RateLimitError classifies as rate limit",
			err:          errortypes.NewRateLimitError("api", "too many requests"),
			expectedType: helpers.ProblemTypeRateLimit,
		},
		{
			name:         "unknown error classifies as internal",
			err:          errors.New("unknown error"),
			expectedType: helpers.ProblemTypeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := classifier.Classify(tt.err)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestDefaultClassifier_ValidatorErrors(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Email string `validate:"required,email"`
	}

	validate := validator.New()
	err := validate.Struct(testStruct{})

	classifier := helpers.NewDefaultClassifier()
	result := classifier.Classify(err)

	assert.Equal(t, helpers.ProblemTypeValidation, result)
}

func TestDefaultClassifier_WrappedErrors(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	t.Run("classifies wrapped business error", func(t *testing.T) {
		t.Parallel()
		businessErr := errortypes.NewBusinessError("original")
		wrapped := errors.Join(errors.New("wrapper"), businessErr)

		result := classifier.Classify(wrapped)
		assert.Equal(t, helpers.ProblemTypeBusiness, result)
	})

	t.Run("classifies wrapped not found error", func(t *testing.T) {
		t.Parallel()
		notFoundErr := errortypes.NewNotFoundError("user not found")
		wrapped := errors.Join(errors.New("wrapper"), notFoundErr)

		result := classifier.Classify(wrapped)
		assert.Equal(t, helpers.ProblemTypeNotFound, result)
	})
}

type customError struct {
	message string
}

func (e *customError) Error() string {
	return e.message
}

func TestChainClassifier_CustomClassifier(t *testing.T) {
	t.Parallel()

	customClassifier := helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
		var ce *customError
		if errors.As(err, &ce) {
			return helpers.ProblemTypeBusiness, true
		}
		return "", false
	})

	chain := helpers.NewDefaultClassifier()
	chain.Register(customClassifier)

	t.Run("custom classifier takes precedence when registered last", func(t *testing.T) {
		t.Parallel()
		result := chain.Classify(&customError{message: "custom"})
		assert.Equal(t, helpers.ProblemTypeBusiness, result)
	})

	t.Run("default classifiers still work", func(t *testing.T) {
		t.Parallel()
		result := chain.Classify(errortypes.NewNotFoundError("not found"))
		assert.Equal(t, helpers.ProblemTypeNotFound, result)
	})
}

func TestChainClassifier_EmptyChain(t *testing.T) {
	t.Parallel()

	chain := helpers.NewChainClassifier()
	result := chain.Classify(errors.New("any error"))

	assert.Equal(t, helpers.ProblemTypeInternal, result)
}

func TestDefaultClassifier_ConflictError(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()
	err := errortypes.NewConflictError("resource already exists")

	result := classifier.Classify(err)

	assert.Equal(t, helpers.ProblemTypeConflict, result)
}

func TestDefaultClassifier_PulidInvalidLength(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	result := classifier.Classify(pulid.ErrInvalidLength)

	assert.Equal(t, helpers.ProblemTypeValidation, result)
}

func TestDefaultClassifier_JSONSyntaxError(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	result := classifier.Classify(&json.SyntaxError{Offset: 1})

	assert.Equal(t, helpers.ProblemTypeValidation, result)
}

func TestDefaultClassifier_JSONUnmarshalTypeError(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	result := classifier.Classify(&json.UnmarshalTypeError{Field: "weight"})

	assert.Equal(t, helpers.ProblemTypeValidation, result)
}

func TestDefaultClassifier_UnexpectedEOF(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	result := classifier.Classify(io.ErrUnexpectedEOF)

	assert.Equal(t, helpers.ProblemTypeValidation, result)
}

func TestDefaultClassifier_EOF(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	result := classifier.Classify(io.EOF)

	assert.Equal(t, helpers.ProblemTypeValidation, result)
}

func TestDefaultClassifier_WrappedConflictError(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()
	conflictErr := errortypes.NewConflictError("conflict")
	wrapped := errors.Join(errors.New("wrapper"), conflictErr)

	result := classifier.Classify(wrapped)

	assert.Equal(t, helpers.ProblemTypeConflict, result)
}

func TestDefaultClassifier_WrappedPulidError(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()
	wrapped := fmt.Errorf("parse failed: %w", pulid.ErrInvalidLength)

	result := classifier.Classify(wrapped)

	assert.Equal(t, helpers.ProblemTypeValidation, result)
}

func TestDefaultClassifier_WrappedAuthenticationError(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()
	authErr := errortypes.NewAuthenticationError("token expired")
	wrapped := fmt.Errorf("middleware: %w", authErr)

	result := classifier.Classify(wrapped)

	assert.Equal(t, helpers.ProblemTypeAuthentication, result)
}

func TestDefaultClassifier_WrappedAuthorizationError(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()
	authzErr := errortypes.NewAuthorizationError("insufficient permissions")
	wrapped := fmt.Errorf("handler: %w", authzErr)

	result := classifier.Classify(wrapped)

	assert.Equal(t, helpers.ProblemTypeAuthorization, result)
}

func TestDefaultClassifier_WrappedDatabaseError(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()
	dbErr := errortypes.NewDatabaseError("connection timeout")
	wrapped := fmt.Errorf("repo: %w", dbErr)

	result := classifier.Classify(wrapped)

	assert.Equal(t, helpers.ProblemTypeDatabase, result)
}

func TestDefaultClassifier_WrappedRateLimitError(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()
	rlErr := errortypes.NewRateLimitError("api", "limit exceeded")
	wrapped := fmt.Errorf("rate: %w", rlErr)

	result := classifier.Classify(wrapped)

	assert.Equal(t, helpers.ProblemTypeRateLimit, result)
}

func TestChainClassifier_RegisterMultiple(t *testing.T) {
	t.Parallel()

	chain := helpers.NewChainClassifier()

	chain.Register(helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
		if err.Error() == "first" {
			return helpers.ProblemTypeValidation, true
		}
		return "", false
	}))

	chain.Register(helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
		if err.Error() == "second" {
			return helpers.ProblemTypeBusiness, true
		}
		return "", false
	}))

	assert.Equal(t, helpers.ProblemTypeValidation, chain.Classify(errors.New("first")))
	assert.Equal(t, helpers.ProblemTypeBusiness, chain.Classify(errors.New("second")))
	assert.Equal(t, helpers.ProblemTypeInternal, chain.Classify(errors.New("unknown")))
}

func TestClassifierFunc_ReturnsFalse(t *testing.T) {
	t.Parallel()

	cf := helpers.ClassifierFunc(func(err error) (helpers.ProblemType, bool) {
		return "", false
	})

	pt, ok := cf.Classify(errors.New("anything"))
	assert.False(t, ok)
	assert.Equal(t, helpers.ProblemType(""), pt)
}
