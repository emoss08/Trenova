package helpers_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/stretchr/testify/assert"
)

func TestProblemType_Info(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		problemType    helpers.ProblemType
		expectedTitle  string
		expectedStatus int
		expectedLog    bool
	}{
		{
			name:           "validation error",
			problemType:    helpers.ProblemTypeValidation,
			expectedTitle:  "Validation Failed",
			expectedStatus: http.StatusBadRequest,
			expectedLog:    false,
		},
		{
			name:           "business error",
			problemType:    helpers.ProblemTypeBusiness,
			expectedTitle:  "Business Rule Violation",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedLog:    true,
		},
		{
			name:           "database error",
			problemType:    helpers.ProblemTypeDatabase,
			expectedTitle:  "Database Operation Failed",
			expectedStatus: http.StatusInternalServerError,
			expectedLog:    true,
		},
		{
			name:           "authentication error",
			problemType:    helpers.ProblemTypeAuthentication,
			expectedTitle:  "Authentication Required",
			expectedStatus: http.StatusUnauthorized,
			expectedLog:    true,
		},
		{
			name:           "authorization error",
			problemType:    helpers.ProblemTypeAuthorization,
			expectedTitle:  "Authorization Required",
			expectedStatus: http.StatusForbidden,
			expectedLog:    true,
		},
		{
			name:           "not found error",
			problemType:    helpers.ProblemTypeNotFound,
			expectedTitle:  "Resource Not Found",
			expectedStatus: http.StatusNotFound,
			expectedLog:    false,
		},
		{
			name:           "rate limit error",
			problemType:    helpers.ProblemTypeRateLimit,
			expectedTitle:  "Rate Limit Exceeded",
			expectedStatus: http.StatusTooManyRequests,
			expectedLog:    false,
		},
		{
			name:           "internal error",
			problemType:    helpers.ProblemTypeInternal,
			expectedTitle:  "Internal Server Error",
			expectedStatus: http.StatusInternalServerError,
			expectedLog:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info := tt.problemType.Info()
			assert.Equal(t, tt.expectedTitle, info.Title)
			assert.Equal(t, tt.expectedStatus, info.StatusCode)
			assert.Equal(t, tt.expectedLog, info.ShouldLog)
			assert.Equal(t, tt.problemType, info.Type)
		})
	}
}

func TestProblemType_Info_UnknownType(t *testing.T) {
	t.Parallel()

	unknownType := helpers.ProblemType("unknown-type")
	info := unknownType.Info()

	assert.Equal(t, "Internal Server Error", info.Title)
	assert.Equal(t, http.StatusInternalServerError, info.StatusCode)
	assert.True(t, info.ShouldLog)
}

func TestProblemType_IsInternal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		problemType helpers.ProblemType
		expected    bool
	}{
		{"internal is internal", helpers.ProblemTypeInternal, true},
		{"database is internal", helpers.ProblemTypeDatabase, true},
		{"validation is not internal", helpers.ProblemTypeValidation, false},
		{"business is not internal", helpers.ProblemTypeBusiness, false},
		{"authentication is not internal", helpers.ProblemTypeAuthentication, false},
		{"authorization is not internal", helpers.ProblemTypeAuthorization, false},
		{"not found is not internal", helpers.ProblemTypeNotFound, false},
		{"rate limit is not internal", helpers.ProblemTypeRateLimit, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.problemType.IsInternal())
		})
	}
}

func TestProblemTypeConstants(t *testing.T) {
	t.Parallel()

	t.Run("blank type is about:blank", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, helpers.ProblemType("about:blank"), helpers.ProblemTypeBlank)
	})

	t.Run("all types have unique values", func(t *testing.T) {
		t.Parallel()
		types := []helpers.ProblemType{
			helpers.ProblemTypeBlank,
			helpers.ProblemTypeValidation,
			helpers.ProblemTypeBusiness,
			helpers.ProblemTypeDatabase,
			helpers.ProblemTypeAuthentication,
			helpers.ProblemTypeAuthorization,
			helpers.ProblemTypeNotFound,
			helpers.ProblemTypeRateLimit,
			helpers.ProblemTypeInternal,
		}

		seen := make(map[helpers.ProblemType]bool)
		for _, pt := range types {
			assert.False(t, seen[pt], "duplicate problem type: %s", pt)
			seen[pt] = true
		}
	})
}
