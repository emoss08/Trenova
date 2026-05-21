package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

func TestControlPlaneAccessMiddleware_DeniesInactiveProductRoute(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zapcore.WarnLevel)
	authorizer := &fakeAccessAuthorizer{
		result: &services.AccessAuthorizeResult{
			FeatureKey: platformcatalog.FeatureDispatch,
			Allowed:    false,
			Reason:     "subscription_inactive",
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
		Registry:         newControlPlaneTestRegistry(t),
		AccessAuthorizer: authorizer,
		UsageProvider:    usage,
		ErrorHandler:     newEntitlementTestErrorHandler(),
		Logger:           zap.New(core),
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

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "subscription_inactive")
	require.Equal(t, platformcatalog.FeatureDispatch, authorizer.req.FeatureKey)
	require.Nil(t, usage.recordReq)
	require.Len(t, logs.All(), 1)

	fields := logs.All()[0].ContextMap()
	require.Equal(t, "control-plane access denied", logs.All()[0].Message)
	require.Equal(t, string(platformcatalog.FeatureDispatch), fields["featureKey"])
	require.Equal(t, "subscription_inactive", fields["reason"])
	require.Equal(t, "/api/v1/shipments/:id", fields["routePattern"])
}

func TestControlPlaneAccessMiddleware_AllowsUnclassifiedProtectedRouteWithWarning(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zapcore.WarnLevel)
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
		Logger:           zap.New(core),
	})

	router := gin.New()
	router.Use(setEntitlementTestAuthContext())
	router.GET("/api/v1/unmapped/", middleware.RequireAccess(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/unmapped/", nil))

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Len(t, logs.All(), 1)
	require.Equal(
		t,
		"allowing unclassified protected route through control-plane access middleware",
		logs.All()[0].Message,
	)
}

func TestControlPlaneAccessMiddleware_BypassesAccountShellRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		method       string
		routePattern string
		path         string
	}{
		{
			name:         "permissions manifest",
			method:       http.MethodGet,
			routePattern: "/api/v1/me/permissions",
			path:         "/api/v1/me/permissions",
		},
		{
			name:         "current user organizations",
			method:       http.MethodGet,
			routePattern: "/api/v1/users/me/organizations/",
			path:         "/api/v1/users/me/organizations/",
		},
		{
			name:         "notification unread count",
			method:       http.MethodGet,
			routePattern: "/api/v1/notifications/unread-count",
			path:         "/api/v1/notifications/unread-count",
		},
		{
			name:         "page favorite check",
			method:       http.MethodGet,
			routePattern: "/api/v1/page-favorites/check",
			path:         "/api/v1/page-favorites/check",
		},
		{
			name:         "realtime token request",
			method:       http.MethodGet,
			routePattern: "/api/v1/realtime/token-request/",
			path:         "/api/v1/realtime/token-request/",
		},
		{
			name:         "organization read",
			method:       http.MethodGet,
			routePattern: "/api/v1/organizations/:id",
			path:         "/api/v1/organizations/org_01H00000000000000000000000",
		},
		{
			name:         "organization logo",
			method:       http.MethodPost,
			routePattern: "/api/v1/organizations/:id/logo",
			path:         "/api/v1/organizations/org_01H00000000000000000000000/logo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := &fakeAccessAuthorizer{}
			usage := &fakeUsageProvider{}
			middleware := NewControlPlaneAccessMiddleware(ControlPlaneAccessMiddlewareParams{
				Config: &config.Config{
					Platform: config.PlatformConfig{
						ControlPlane: config.PlatformControlPlaneConfig{Enabled: true},
					},
				},
				Registry:         newControlPlaneTestRegistry(t),
				AccessAuthorizer: authorizer,
				UsageProvider:    usage,
				ErrorHandler:     newEntitlementTestErrorHandler(),
				Logger:           zap.NewNop(),
			})

			router := gin.New()
			router.Use(setEntitlementTestAuthContext())
			router.Handle(tt.method, tt.routePattern, middleware.RequireAccess(), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.path, nil))

			require.Equal(t, http.StatusNoContent, recorder.Code)
			require.Nil(t, authorizer.req)
			require.Nil(t, usage.limitReq)
			require.Nil(t, usage.recordReq)
		})
	}
}

func TestControlPlaneAccessMiddleware_AccountShellRoutesStillRequireAuth(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		App: config.AppConfig{Debug: true},
		Security: config.SecurityConfig{
			Session: config.SessionConfig{
				Name: "trenova_session",
			},
		},
		Platform: config.PlatformConfig{
			ControlPlane: config.PlatformControlPlaneConfig{Enabled: true},
		},
	}
	authService := mocks.NewMockAuthService(t)
	authMiddleware := NewAuthMiddleware(AuthMiddlewareParams{
		Config:       cfg,
		Service:      authService,
		ErrorHandler: newEntitlementTestErrorHandler(),
	})
	controlPlaneMiddleware := NewControlPlaneAccessMiddleware(ControlPlaneAccessMiddlewareParams{
		Config:           cfg,
		Registry:         newControlPlaneTestRegistry(t),
		AccessAuthorizer: &fakeAccessAuthorizer{},
		UsageProvider:    &fakeUsageProvider{},
		ErrorHandler:     newEntitlementTestErrorHandler(),
		Logger:           zap.NewNop(),
	})

	router := gin.New()
	router.GET("/api/v1/me/permissions",
		authMiddleware.RequireAuth(),
		controlPlaneMiddleware.RequireAccess(),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/me/permissions", nil))

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	authService.AssertNotCalled(t, "ValidateSession")
	authService.AssertNotCalled(t, "AuthenticateAPIKey")
}

func TestControlPlaneAccessMiddleware_AllowsLegacyBillingAndCurrentUserShellRoutes(t *testing.T) {
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
	router.GET("/api/v1/users/me/", middleware.RequireAccess(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/me/billing", nil))
	require.Equal(t, http.StatusNoContent, recorder.Code)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/users/me/", nil))

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestControlPlaneAccessMiddleware_LogFieldsForUnclassifiedRoute(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zapcore.WarnLevel)
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
		Logger:           zap.New(core),
	})

	router := gin.New()
	router.Use(setEntitlementTestAuthContext())
	router.POST("/api/v1/unmapped/:id", middleware.RequireAccess(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/unmapped/test", nil))

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Len(t, logs.All(), 1)

	fields := logs.All()[0].ContextMap()
	require.Equal(t, http.MethodPost, fields["method"])
	require.Equal(t, "/api/v1/unmapped/:id", fields["routePattern"])
	require.Equal(t, "/api/v1/unmapped/test", fields["path"])
	require.True(t, strings.HasPrefix(fields["organizationID"].(string), "org_"))
	require.True(t, strings.HasPrefix(fields["businessUnitID"].(string), "bu_"))
}

func newControlPlaneTestRegistry(t *testing.T) *platformcatalog.Registry {
	t.Helper()

	registry, err := platformcatalog.NewRegistry(platformcatalog.RegistryParams{
		Providers: []platformcatalog.CatalogProvider{platformcatalog.NewStaticProvider()},
	})
	require.NoError(t, err)
	return registry
}
