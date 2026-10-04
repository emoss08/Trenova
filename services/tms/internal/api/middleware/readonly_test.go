package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type guardCase struct {
	plans  services.PlanService
	tenant pagination.TenantInfo
	apiKey bool
	method string
	route  string
	path   string
}

func serveThroughGuard(t *testing.T, tc guardCase) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(recorder)

	guard := NewReadOnlyGuard(ReadOnlyGuardParams{
		Plans: tc.plans,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: &config.Config{},
		}),
		Logger: zap.NewNop(),
	})

	engine.Use(func(c *gin.Context) {
		if tc.apiKey {
			authctx.SetAPIKeyContext(c, "key_test", tc.tenant.BuID, tc.tenant.OrgID)
		} else {
			authctx.SetUserID(c, tc.tenant.UserID)
			authctx.SetBusinessUnitID(c, tc.tenant.BuID)
			authctx.SetOrganizationID(c, tc.tenant.OrgID)
		}
		c.Next()
	})
	engine.Use(guard.Guard())

	reached := false
	engine.Handle(tc.method, tc.route, func(c *gin.Context) {
		reached = true
		c.Status(http.StatusNoContent)
	})

	path := tc.path
	if path == "" {
		path = tc.route
	}
	engine.ServeHTTP(recorder, httptest.NewRequest(tc.method, path, nil))

	return recorder, reached
}

func TestReadOnlyGuardIsInertOutsideCloudMode(t *testing.T) {
	t.Parallel()

	recorder, reached := serveThroughGuard(t, guardCase{
		plans:  planservice.NewUnlimited(),
		tenant: plantest.Tenant(),
		method: http.MethodPost,
		route:  "/api/v1/shipments/",
	})

	assert.True(t, reached)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestReadOnlyGuardRefusesWritesFromAReadOnlyOrganization(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusReadOnly))

	recorder, reached := serveThroughGuard(t, guardCase{
		plans:  plans,
		tenant: tenant,
		method: http.MethodPost,
		route:  "/api/v1/shipments/",
	})

	assert.False(t, reached)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "plan-restricted")
	assert.Contains(t, recorder.Body.String(), platformplan.ReasonSubscriptionReadOnly)
}

func TestReadOnlyGuardLetsReadsAndAllowlistedWritesThroughWhileReadOnly(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusReadOnly))

	for _, tc := range []guardCase{
		{method: http.MethodGet, route: "/api/v1/shipments/"},
		{method: http.MethodPost, route: "/api/v1/users/me/change-password/"},
		{method: http.MethodPost, route: "/api/v1/shipments/calculate-totals/"},
		{
			method: http.MethodPost,
			route:  "/api/v1/billing/invoice-adjustments/drafts/:adjustmentID/preview/",
			path:   "/api/v1/billing/invoice-adjustments/drafts/ia_1/preview/",
		},
		{method: http.MethodPost, route: "/graphql"},
	} {
		tc.plans = plans
		tc.tenant = tenant
		recorder, reached := serveThroughGuard(t, tc)
		assert.True(t, reached, tc.route)
		assert.Equal(t, http.StatusNoContent, recorder.Code, tc.route)
	}
}

func TestReadOnlyGuardRefusesReadsOnceExpiredExceptTheAccountShell(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusExpired))

	recorder, reached := serveThroughGuard(t, guardCase{
		plans:  plans,
		tenant: tenant,
		method: http.MethodGet,
		route:  "/api/v1/shipments/",
	})
	assert.False(t, reached)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), platformplan.ReasonSubscriptionExpired)

	for _, route := range []string{"/api/v1/users/me/", "/api/v1/me/billing"} {
		recorder, reached = serveThroughGuard(t, guardCase{
			plans:  plans,
			tenant: tenant,
			method: http.MethodGet,
			route:  route,
		})
		assert.True(t, reached, route)
		assert.Equal(t, http.StatusNoContent, recorder.Code, route)
	}
}

func TestReadOnlyGuardRefusesAPIKeysWhenThePlanRestrictsThem(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusTrialing))

	recorder, reached := serveThroughGuard(t, guardCase{
		plans:  plans,
		tenant: tenant,
		apiKey: true,
		method: http.MethodGet,
		route:  "/api/v1/shipments/",
	})

	assert.False(t, reached)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), platformplan.CapabilityAPIKeys.String())
}

func TestReadOnlyGuardLetsATrialOrganizationWrite(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	plans := plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusTrialing))

	recorder, reached := serveThroughGuard(t, guardCase{
		plans:  plans,
		tenant: tenant,
		method: http.MethodDelete,
		route:  "/api/v1/shipments/:id/",
		path:   "/api/v1/shipments/shp_1/",
	})

	require.True(t, reached)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestReadOnlyWriteAllowedMatchesPathsAndReadActions(t *testing.T) {
	t.Parallel()

	assert.True(t, ReadOnlyWriteAllowed("/api/v1/users/me/settings/"))
	assert.True(t, ReadOnlyWriteAllowed("/api/v1/rate-quotes/shipment/:shipmentID/explain/"))
	assert.True(t, ReadOnlyWriteAllowed("/graphql"))
	assert.False(t, ReadOnlyWriteAllowed("/api/v1/shipments/"))
	assert.False(t, ReadOnlyWriteAllowed("/api/v1/users/me/profile-picture/"))
	assert.False(t, ReadOnlyWriteAllowed("/api/v1/shipments/:id/duplicate/"))
}

func TestIsUnsafeMethod(t *testing.T) {
	t.Parallel()

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		assert.False(t, IsUnsafeMethod(method), method)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		assert.True(t, IsUnsafeMethod(method), method)
	}
}
