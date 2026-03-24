package helpers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProblemJSONContentType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "application/problem+json", helpers.ProblemJSONContentType)
}

func TestNewProblemBuilder(t *testing.T) {
	t.Parallel()

	t.Run("creates builder with defaults", func(t *testing.T) {
		t.Parallel()
		builder := helpers.NewProblemBuilder("https://api.example.com/problems/")
		problem := builder.Build()

		assert.Equal(t, "https://api.example.com/problems/internal-error", problem.Type)
		assert.Equal(t, "Internal Server Error", problem.Title)
		assert.Equal(t, http.StatusInternalServerError, problem.Status)
	})
}

func TestProblemBuilder_WithType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		problemType    helpers.ProblemType
		expectedType   string
		expectedTitle  string
		expectedStatus int
	}{
		{
			name:           "validation type",
			problemType:    helpers.ProblemTypeValidation,
			expectedType:   "https://api.test.com/validation-error",
			expectedTitle:  "Validation Failed",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "not found type",
			problemType:    helpers.ProblemTypeNotFound,
			expectedType:   "https://api.test.com/resource-not-found",
			expectedTitle:  "Resource Not Found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "blank type defaults to about:blank",
			problemType:    helpers.ProblemTypeBlank,
			expectedType:   "about:blank",
			expectedTitle:  "Internal Server Error",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			problem := helpers.NewProblemBuilder("https://api.test.com/").
				WithType(tt.problemType).
				Build()

			assert.Equal(t, tt.expectedType, problem.Type)
			assert.Equal(t, tt.expectedTitle, problem.Title)
			assert.Equal(t, tt.expectedStatus, problem.Status)
		})
	}
}

func TestProblemBuilder_WithDetail(t *testing.T) {
	t.Parallel()

	problem := helpers.NewProblemBuilder("https://api.test.com/").
		WithType(helpers.ProblemTypeValidation).
		WithDetail("Email format is invalid").
		Build()

	assert.Equal(t, "Email format is invalid", problem.Detail)
}

func TestProblemBuilder_WithInstance(t *testing.T) {
	t.Parallel()

	t.Run("with request ID", func(t *testing.T) {
		t.Parallel()
		problem := helpers.NewProblemBuilder("https://api.test.com/").
			WithInstance("/api/users", "req-12345").
			Build()

		assert.Equal(t, "/api/users#req-12345", problem.Instance)
	})

	t.Run("without request ID", func(t *testing.T) {
		t.Parallel()
		problem := helpers.NewProblemBuilder("https://api.test.com/").
			WithInstance("/api/users", "").
			Build()

		assert.Equal(t, "/api/users", problem.Instance)
	})
}

func TestProblemBuilder_WithTraceID(t *testing.T) {
	t.Parallel()

	problem := helpers.NewProblemBuilder("https://api.test.com/").
		WithTraceID("trace-abc123").
		Build()

	assert.Equal(t, "trace-abc123", problem.TraceID)
}

func TestProblemBuilder_WithErrors(t *testing.T) {
	t.Parallel()

	errors := []helpers.ValidationError{
		{Field: "email", Message: "Invalid format", Code: "INVALID_FORMAT", Location: "body"},
		{Field: "password", Message: "Too short", Code: "MIN_LENGTH", Location: "body"},
	}

	problem := helpers.NewProblemBuilder("https://api.test.com/").
		WithType(helpers.ProblemTypeValidation).
		WithErrors(errors).
		Build()

	require.Len(t, problem.Errors, 2)
	assert.Equal(t, "email", problem.Errors[0].Field)
	assert.Equal(t, "Invalid format", problem.Errors[0].Message)
	assert.Equal(t, "password", problem.Errors[1].Field)
}

func TestProblemBuilder_FullChain(t *testing.T) {
	t.Parallel()

	errors := []helpers.ValidationError{
		{Field: "email", Message: "Required", Code: "REQUIRED", Location: "body"},
	}

	problem := helpers.NewProblemBuilder("https://api.example.com/problems/").
		WithType(helpers.ProblemTypeValidation).
		WithDetail("Request validation failed").
		WithInstance("/api/users/create", "req-999").
		WithTraceID("trace-xyz").
		WithErrors(errors).
		Build()

	assert.Equal(t, "https://api.example.com/problems/validation-error", problem.Type)
	assert.Equal(t, "Validation Failed", problem.Title)
	assert.Equal(t, http.StatusBadRequest, problem.Status)
	assert.Equal(t, "Request validation failed", problem.Detail)
	assert.Equal(t, "/api/users/create#req-999", problem.Instance)
	assert.Equal(t, "trace-xyz", problem.TraceID)
	require.Len(t, problem.Errors, 1)
}

func TestProblemDetail_JSONSerialization(t *testing.T) {
	t.Parallel()

	t.Run("serializes all fields", func(t *testing.T) {
		t.Parallel()
		problem := helpers.NewProblemBuilder("https://api.test.com/").
			WithType(helpers.ProblemTypeValidation).
			WithDetail("Validation failed").
			WithInstance("/api/test", "req-123").
			WithTraceID("trace-456").
			WithErrors([]helpers.ValidationError{
				{Field: "name", Message: "Required", Code: "REQUIRED", Location: "body"},
			}).
			Build()

		data, err := json.Marshal(problem)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "https://api.test.com/validation-error", result["type"])
		assert.Equal(t, "Validation Failed", result["title"])
		assert.Equal(t, float64(400), result["status"])
		assert.Equal(t, "Validation failed", result["detail"])
		assert.Equal(t, "/api/test#req-123", result["instance"])
		assert.Equal(t, "trace-456", result["traceId"])

		errors, ok := result["errors"].([]any)
		require.True(t, ok)
		require.Len(t, errors, 1)
	})

	t.Run("omits empty optional fields", func(t *testing.T) {
		t.Parallel()
		problem := helpers.NewProblemBuilder("https://api.test.com/").
			WithType(helpers.ProblemTypeInternal).
			Build()

		data, err := json.Marshal(problem)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Contains(t, result, "type")
		assert.Contains(t, result, "title")
		assert.Contains(t, result, "status")
		assert.NotContains(t, result, "detail")
		assert.NotContains(t, result, "instance")
		assert.NotContains(t, result, "traceId")
		assert.NotContains(t, result, "errors")
	})
}

func TestValidationError_JSONSerialization(t *testing.T) {
	t.Parallel()

	ve := helpers.ValidationError{
		Field:    "email",
		Message:  "Invalid email format",
		Code:     "INVALID_FORMAT",
		Location: "body",
	}

	data, err := json.Marshal(ve)
	require.NoError(t, err)

	var result map[string]string
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "email", result["field"])
	assert.Equal(t, "Invalid email format", result["message"])
	assert.Equal(t, "INVALID_FORMAT", result["code"])
	assert.Equal(t, "body", result["location"])
}

func TestValidationError_OmitsEmptyFields(t *testing.T) {
	t.Parallel()

	ve := helpers.ValidationError{
		Field:   "name",
		Message: "Required",
	}

	data, err := json.Marshal(ve)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "field")
	assert.Contains(t, result, "message")
	assert.NotContains(t, result, "code")
	assert.NotContains(t, result, "location")
}
