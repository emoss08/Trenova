package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	sessionUserID         = pulid.ID("usr_01JQ0000000000000000000001")
	sessionOrganizationID = pulid.ID("org_01JQ0000000000000000000001")
	sessionBusinessUnitID = pulid.ID("bu_01JQ0000000000000000000001")
	otherUserID           = "usr_01JQ0000000000000000000002"
	otherOrganizationID   = "org_01JQ0000000000000000000002"
	otherBusinessUnitID   = "bu_01JQ0000000000000000000002"
)

func currentSessionHeaders() map[string]string {
	return map[string]string{
		UserHeader:         sessionUserID.String(),
		OrganizationHeader: sessionOrganizationID.String(),
		BusinessUnitHeader: sessionBusinessUnitID.String(),
	}
}

func withHeader(name, value string) map[string]string {
	headers := currentSessionHeaders()
	headers[name] = value
	return headers
}

func serveWithSessionHeaders(
	t *testing.T,
	headers map[string]string,
) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(recorder)

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    zap.NewNop(),
		AppConfig: &appconfig.Config{},
	})

	reached := false
	engine.Use(func(c *gin.Context) {
		authctx.SetSessionAuthContext(c, authctx.SessionAuthContextParams{
			UserID:         sessionUserID,
			OrganizationID: sessionOrganizationID,
			BusinessUnitID: sessionBusinessUnitID,
		})
		c.Next()
	})
	engine.Use(NewSessionContextMiddleware(errorHandler).RequireCurrent())
	engine.POST("/api/v1/shipments/", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/shipments/", nil)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	engine.ServeHTTP(recorder, request)

	return recorder, reached
}

func TestRequireCurrent_PassesARequestThatAssertsNoSession(t *testing.T) {
	t.Parallel()

	recorder, reached := serveWithSessionHeaders(t, nil)

	assert.True(t, reached)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestRequireCurrent_PassesARequestForTheCurrentSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		headers map[string]string
	}{
		{name: "every header", headers: currentSessionHeaders()},
		{
			name:    "organization only",
			headers: map[string]string{OrganizationHeader: sessionOrganizationID.String()},
		},
		{
			name:    "surrounding whitespace",
			headers: withHeader(UserHeader, " "+sessionUserID.String()+" "),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder, reached := serveWithSessionHeaders(t, tt.headers)

			assert.True(t, reached)
			assert.Equal(t, http.StatusOK, recorder.Code)
		})
	}
}

func TestRequireCurrent_RefusesARequestForAnotherSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		headers map[string]string
	}{
		{name: "another person", headers: withHeader(UserHeader, otherUserID)},
		{name: "other organization", headers: withHeader(OrganizationHeader, otherOrganizationID)},
		{name: "other business unit", headers: withHeader(BusinessUnitHeader, otherBusinessUnitID)},
		{name: "malformed user", headers: map[string]string{UserHeader: "not-an-id"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder, reached := serveWithSessionHeaders(t, tt.headers)

			require.False(t, reached, "the handler must not run")
			assert.Equal(t, http.StatusConflict, recorder.Code)
			body := recorder.Body.String()
			assert.Contains(t, body, `session-context-stale`)
			assert.Contains(t, body, `"userId":"`+sessionUserID.String()+`"`)
			assert.Contains(t, body, `"organizationId":"`+sessionOrganizationID.String()+`"`)
			assert.Contains(t, body, `"businessUnitId":"`+sessionBusinessUnitID.String()+`"`)
		})
	}
}

func TestWithSessionContextHeaders_AllowsTheSessionHeadersWhateverTheConfigLists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		allowed  []string
		expected []string
	}{
		{
			name:    "config lists none",
			allowed: []string{"Content-Type", "X-CSRF-Token"},
			expected: []string{
				"Content-Type",
				"X-CSRF-Token",
				UserHeader,
				OrganizationHeader,
				BusinessUnitHeader,
			},
		},
		{
			name:    "config lists one in another case",
			allowed: []string{"Content-Type", "x-trenova-organization-id"},
			expected: []string{
				"Content-Type",
				"x-trenova-organization-id",
				UserHeader,
				BusinessUnitHeader,
			},
		},
		{
			name:     "config lists all",
			allowed:  []string{BusinessUnitHeader, OrganizationHeader, UserHeader},
			expected: []string{BusinessUnitHeader, OrganizationHeader, UserHeader},
		},
		{
			name:     "config lists nothing",
			allowed:  nil,
			expected: []string{UserHeader, OrganizationHeader, BusinessUnitHeader},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, WithSessionContextHeaders(tt.allowed))
		})
	}
}

func TestWithSessionContextHeaders_LeavesTheConfiguredListUntouched(t *testing.T) {
	t.Parallel()

	backing := make([]string, 4)
	backing[0] = "Content-Type"
	allowed := backing[:1]

	merged := WithSessionContextHeaders(allowed)
	merged[0] = "Accept"

	assert.Equal(t, []string{"Content-Type", "", "", ""}, backing)
}
