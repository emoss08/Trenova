package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeEntitlementProvider struct {
	result *services.FeatureCheckResult
	err    error
	req    *services.FeatureCheckRequest
}

func (p *fakeEntitlementProvider) CheckFeature(
	_ context.Context,
	req *services.FeatureCheckRequest,
) (*services.FeatureCheckResult, error) {
	p.req = req
	if p.err != nil {
		return nil, p.err
	}
	return p.result, nil
}

func (p *fakeEntitlementProvider) ListEntitlements(
	context.Context,
	*services.EntitlementsRequest,
) (*services.EntitlementsResult, error) {
	return nil, nil
}

func TestEntitlementMiddleware_RequireFeatureAllowed(t *testing.T) {
	t.Parallel()

	provider := &fakeEntitlementProvider{
		result: &services.FeatureCheckResult{
			FeatureKey: platformcatalog.FeatureCoreTMS,
			Allowed:    true,
		},
	}
	middleware := NewEntitlementMiddleware(EntitlementMiddlewareParams{
		EntitlementProvider: provider,
		ErrorHandler:        newEntitlementTestErrorHandler(),
	})

	router := gin.New()
	router.GET("/test", setEntitlementTestAuthContext(), middleware.RequireFeature(platformcatalog.FeatureCoreTMS), func(c *gin.Context) {
		require.NotNil(t, GetFeatureCheckResult(c))
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Equal(t, platformcatalog.FeatureCoreTMS, provider.req.FeatureKey)
	require.Equal(t, pulid.ID("org_01H00000000000000000000000"), provider.req.OrganizationID)
}

func TestEntitlementMiddleware_RequireFeatureDenied(t *testing.T) {
	t.Parallel()

	provider := &fakeEntitlementProvider{
		result: &services.FeatureCheckResult{
			FeatureKey: platformcatalog.FeatureCoreTMS,
			Allowed:    false,
		},
	}
	middleware := NewEntitlementMiddleware(EntitlementMiddlewareParams{
		EntitlementProvider: provider,
		ErrorHandler:        newEntitlementTestErrorHandler(),
	})

	router := gin.New()
	router.GET("/test", setEntitlementTestAuthContext(), middleware.RequireFeature(platformcatalog.FeatureCoreTMS), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestEntitlementMiddleware_OptionalFeaturePropagatesErrors(t *testing.T) {
	t.Parallel()

	middleware := NewEntitlementMiddleware(EntitlementMiddlewareParams{
		EntitlementProvider: &fakeEntitlementProvider{err: errors.New("control plane down")},
		ErrorHandler:        newEntitlementTestErrorHandler(),
	})

	router := gin.New()
	router.GET("/test", setEntitlementTestAuthContext(), middleware.OptionalFeature(platformcatalog.FeatureCoreTMS), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func setEntitlementTestAuthContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		authctx.SetAuthContext(
			c,
			pulid.ID("usr_01H00000000000000000000000"),
			pulid.ID("bu_01H00000000000000000000000"),
			pulid.ID("org_01H00000000000000000000000"),
		)
		c.Next()
	}
}

func newEntitlementTestErrorHandler() *helpers.ErrorHandler {
	return helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{
			App: config.AppConfig{Debug: true},
		},
	})
}
