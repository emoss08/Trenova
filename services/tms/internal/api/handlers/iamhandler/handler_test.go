package iamhandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/iamhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/iamportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newIAMHandler(t *testing.T) *iamhandler.Handler {
	t.Helper()

	return newIAMHandlerWithBriefing(t, iamportsmock.NewMockSecurityBriefingService(t))
}

func newIAMHandlerWithBriefing(
	t *testing.T,
	briefing iamports.SecurityBriefingService,
) *iamhandler.Handler {
	t.Helper()

	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: cfg,
	})

	return iamhandler.New(iamhandler.Params{
		Service:          iamportsmock.NewMockIAMService(t),
		SecurityBriefing: briefing,
		ErrorHandler:     errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: &mocks.AllowAllPermissionEngine{},
				ErrorHandler:     errorHandler,
			},
		),
	})
}

func TestIAMRoutes_RefuseAnOrganizationOtherThanTheSessions(t *testing.T) {
	t.Parallel()

	otherOrg := pulid.MustNew("org_").String()
	cases := []struct {
		method string
		path   string
		body   map[string]any
	}{
		{method: http.MethodGet, path: "identity-providers"},
		{method: http.MethodGet, path: "security-briefing"},
		{method: http.MethodPost, path: "identity-providers", body: map[string]any{"name": "Okta"}},
		{method: http.MethodPost, path: "scim-directories", body: map[string]any{"name": "Directory"}},
		{method: http.MethodPost, path: "access-policies", body: map[string]any{"name": "Policy"}},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()

			handler := newIAMHandler(t)
			ginCtx := testutil.NewGinTestContext().
				WithMethod(tc.method).
				WithPath("/api/v1/organizations/" + otherOrg + "/iam/" + tc.path).
				WithDefaultAuthContext()
			if tc.body != nil {
				ginCtx = ginCtx.WithJSONBody(tc.body)
			}

			handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
			ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

			assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
		})
	}
}

func TestSecurityBriefingRoute_ReturnsTheBriefingForTheSessionsOrganization(t *testing.T) {
	t.Parallel()

	riskDenied := int64(0)
	briefing := iam.NewSecurityBriefing(1_760_000_000)
	briefing.Session = iam.SecuritySessionPolicy{
		MaxAgeSeconds:      43_200,
		IdleTimeoutSeconds: 1_800,
		IdleWarningSeconds: 120,
	}
	briefing.MFA = &iam.SecurityMFACoverage{Enrolled: 1, Total: 2}
	briefing.SignIns = &iam.SecuritySignInSummary{Succeeded: 5, RiskDenied: &riskDenied}
	briefing.Findings = iam.BuildSecurityFindings(briefing)

	service := iamportsmock.NewMockSecurityBriefingService(t)
	service.EXPECT().
		Get(mock.Anything, mock.MatchedBy(func(actor *permissionports.RequestActor) bool {
			return actor.OrganizationID == testutil.TestOrgID &&
				actor.BusinessUnitID == testutil.TestBuID &&
				actor.UserID == testutil.TestUserID
		})).
		Return(briefing, nil)

	handler := newIAMHandlerWithBriefing(t, service)
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/organizations/" + testutil.TestOrgID.String() +
			"/iam/security-briefing").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))

	assert.InDelta(t, 1_760_000_000, resp["generatedAt"], 0)
	assert.InDelta(t, 7, resp["signInWindowDays"], 0)
	assert.InDelta(t, 90, resp["dormantAfterDays"], 0)
	assert.InDelta(t, 90, resp["apiKeyUnusedAfterDays"], 0)
	assert.InDelta(t, 14, resp["apiKeyExpiringWithinDays"], 0)
	assert.Equal(t, map[string]any{
		"maxAgeSeconds":      float64(43_200),
		"idleTimeoutSeconds": float64(1_800),
		"idleWarningSeconds": float64(120),
	}, resp["session"])
	assert.Equal(t, map[string]any{
		"enrolled":   float64(1),
		"total":      float64(2),
		"unenrolled": nil,
	}, resp["mfa"])
	assert.Equal(t, map[string]any{
		"succeeded":  float64(5),
		"failed":     float64(0),
		"denied":     float64(0),
		"throttled":  float64(0),
		"riskDenied": float64(0),
	}, resp["signIns"])
	for _, section := range []string{"members", "apiKeys", "sso", "provisioning", "policies"} {
		value, present := resp[section]
		assert.True(t, present, section)
		assert.Nil(t, value, section)
	}
	assert.Equal(t, []any{
		finding("mfa_coverage", "critical", 1, 2),
		finding("failed_sign_ins", "pass", 0, 5),
		finding("idle_timeout", "pass", 0, 0),
	}, resp["findings"])
}

func finding(code, severity string, count, total float64) map[string]any {
	return map[string]any{"code": code, "severity": severity, "count": count, "total": total}
}
