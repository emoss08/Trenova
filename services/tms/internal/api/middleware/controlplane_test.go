package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeAccessAuthorizer struct {
	result *services.AccessAuthorizeResult
	req    *services.AccessAuthorizeRequest
	err    error
}

func (a *fakeAccessAuthorizer) AuthorizeAccess(
	_ context.Context,
	req *services.AccessAuthorizeRequest,
) (*services.AccessAuthorizeResult, error) {
	a.req = req
	if a.err != nil {
		return nil, a.err
	}
	return a.result, nil
}

type fakeUsageProvider struct {
	limitResult *services.UsageLimitCheckResult
	limitReq    *services.UsageLimitCheckRequest
	recordReq   *services.UsageRecordRequest
	limitErr    error
	recordErr   error
}

func (p *fakeUsageProvider) CheckLimit(
	_ context.Context,
	req *services.UsageLimitCheckRequest,
) (*services.UsageLimitCheckResult, error) {
	p.limitReq = req
	if p.limitErr != nil {
		return nil, p.limitErr
	}
	return p.limitResult, nil
}

func (p *fakeUsageProvider) RecordUsage(
	_ context.Context,
	req *services.UsageRecordRequest,
) (*services.UsageRecordResult, error) {
	p.recordReq = req
	if p.recordErr != nil {
		return nil, p.recordErr
	}
	return &services.UsageRecordResult{
		MeterKey:       req.MeterKey,
		Recorded:       true,
		Quantity:       req.Quantity,
		RecordedAt:     req.RecordedAt,
		IdempotencyKey: req.IdempotencyKey,
	}, nil
}

func TestControlPlaneAccessMiddleware_AuthorizesMappedRoute(t *testing.T) {
	t.Parallel()

	registry := newControlPlaneTestRegistry(t)
	authorizer := &fakeAccessAuthorizer{
		result: &services.AccessAuthorizeResult{
			FeatureKey: platformcatalog.FeatureDispatch,
			Allowed:    true,
			CheckedAt:  123,
		},
	}
	usage := &fakeUsageProvider{
		limitResult: &services.UsageLimitCheckResult{
			MeterKey:  platformcatalog.MeterAPIRequests,
			Allowed:   true,
			CheckedAt: 123,
		},
	}

	middleware := NewControlPlaneAccessMiddleware(ControlPlaneAccessMiddlewareParams{
		Config: &config.Config{
			Platform: config.PlatformConfig{
				ControlPlane: config.PlatformControlPlaneConfig{Enabled: true},
			},
		},
		Registry:         registry,
		AccessAuthorizer: authorizer,
		UsageProvider:    usage,
		ErrorHandler:     newEntitlementTestErrorHandler(),
		Logger:           zap.NewNop(),
	})
	middleware.now = func() time.Time { return time.Unix(123, 0) }

	router := gin.New()
	router.Use(setEntitlementTestAuthContext())
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req_123")
		c.Next()
	})
	router.GET("/api/v1/shipments/:id", middleware.RequireAccess(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/shipments/shp_01H00000000000000000000000", nil))

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Equal(t, platformcatalog.FeatureDispatch, authorizer.req.FeatureKey)
	require.Equal(t, "/api/v1/shipments/:id", authorizer.req.RoutePattern)
	require.Equal(t, platformcatalog.MeterAPIRequests, usage.limitReq.MeterKey)
	require.Equal(t, usage.limitReq.IdempotencyKey, usage.recordReq.IdempotencyKey)
	require.Equal(t, pulid.ID("org_01H00000000000000000000000"), authorizer.req.OrganizationID)
}

func TestControlPlaneAccessMiddleware_DeniesUnmappedRoute(t *testing.T) {
	t.Parallel()

	middleware := NewControlPlaneAccessMiddleware(ControlPlaneAccessMiddlewareParams{
		Config: &config.Config{
			Platform: config.PlatformConfig{
				ControlPlane: config.PlatformControlPlaneConfig{Enabled: true},
			},
		},
		Registry:         newControlPlaneTestRegistry(t),
		AccessAuthorizer: &fakeAccessAuthorizer{},
		UsageProvider:    &fakeUsageProvider{},
		ErrorHandler:     newEntitlementTestErrorHandler(),
		Logger:           zap.NewNop(),
	})

	router := gin.New()
	router.Use(setEntitlementTestAuthContext())
	router.GET("/api/v1/unmapped/", middleware.RequireAccess(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/unmapped/", nil))

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestControlPlaneAccessMiddleware_AllowsBillingStatusRoute(t *testing.T) {
	t.Parallel()

	middleware := NewControlPlaneAccessMiddleware(ControlPlaneAccessMiddlewareParams{
		Config: &config.Config{
			Platform: config.PlatformConfig{
				ControlPlane: config.PlatformControlPlaneConfig{Enabled: true},
			},
		},
		Registry:         newControlPlaneTestRegistry(t),
		AccessAuthorizer: nil,
		UsageProvider:    nil,
		ErrorHandler:     newEntitlementTestErrorHandler(),
		Logger:           zap.NewNop(),
	})

	router := gin.New()
	router.Use(setEntitlementTestAuthContext())
	router.GET("/api/v1/me/billing", middleware.RequireAccess(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/me/billing", nil))

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func newControlPlaneTestRegistry(t *testing.T) *platformcatalog.Registry {
	t.Helper()

	registry, err := platformcatalog.NewRegistry(platformcatalog.RegistryParams{
		Providers: []platformcatalog.CatalogProvider{platformcatalog.NewStaticProvider()},
	})
	require.NoError(t, err)
	return registry
}
