package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ControlPlaneAccessMiddlewareParams struct {
	fx.In

	Config           *config.Config
	Registry         *platformcatalog.Registry
	AccessAuthorizer services.AccessAuthorizer `optional:"true"`
	UsageProvider    services.UsageProvider
	ErrorHandler     *helpers.ErrorHandler
	Logger           *zap.Logger
}

type ControlPlaneAccessMiddleware struct {
	cfg          *config.Config
	registry     *platformcatalog.Registry
	authorizer   services.AccessAuthorizer
	usage        services.UsageProvider
	errorHandler *helpers.ErrorHandler
	logger       *zap.Logger
	now          func() time.Time
}

func NewControlPlaneAccessMiddleware(
	p ControlPlaneAccessMiddlewareParams,
) *ControlPlaneAccessMiddleware {
	return &ControlPlaneAccessMiddleware{
		cfg:          p.Config,
		registry:     p.Registry,
		authorizer:   p.AccessAuthorizer,
		usage:        p.UsageProvider,
		errorHandler: p.ErrorHandler,
		logger:       p.Logger.Named("control-plane-access"),
		now:          time.Now,
	}
}

func (m *ControlPlaneAccessMiddleware) RequireAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.cfg.Platform.ControlPlane.Enabled {
			c.Next()
			return
		}
		routePattern := c.FullPath()

		policy := m.registry.PolicyForRoute(c.Request.Method, routePattern)
		switch policy.AccessClass {
		case platformcatalog.RouteAccessClassAccountShell:
			c.Next()
			return
		case platformcatalog.RouteAccessClassUnclassified:
			m.logUnclassifiedRoute(c, routePattern)
			c.Next()
			return
		case platformcatalog.RouteAccessClassProduct:
		default:
			m.logUnclassifiedRoute(c, routePattern)
			c.Next()
			return
		}

		if m.authorizer == nil {
			m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(
				"Control-plane authorization is not configured",
			))
			return
		}

		feature, ok := m.registry.GetFeature(policy.FeatureKey)
		if !ok {
			m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(
				"Route is not mapped to a platform feature",
			))
			return
		}

		authCtx := authctx.GetAuthContext(c)
		checkedAt := m.now().Unix()
		idempotencyKey, err := apiRequestIdempotencyKey(c, routePattern)
		if err != nil {
			m.errorHandler.HandleError(c, err)
			return
		}

		limitResult, err := m.usage.CheckLimit(
			c.Request.Context(),
			&services.UsageLimitCheckRequest{
				OrganizationID: authCtx.OrganizationID,
				BusinessUnitID: authCtx.BusinessUnitID,
				PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
				PrincipalID:    authCtx.PrincipalID,
				UserID:         authCtx.UserID,
				APIKeyID:       authCtx.APIKeyID,
				MeterKey:       platformcatalog.MeterAPIRequests,
				Quantity:       1,
				CheckedAt:      checkedAt,
				IdempotencyKey: idempotencyKey,
			},
		)
		if err != nil {
			m.errorHandler.HandleError(c, err)
			return
		}
		if !limitResult.Allowed {
			m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(limitResult.Reason))
			return
		}

		result, err := m.authorizer.AuthorizeAccess(
			c.Request.Context(),
			&services.AccessAuthorizeRequest{
				OrganizationID: authCtx.OrganizationID,
				BusinessUnitID: authCtx.BusinessUnitID,
				PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
				PrincipalID:    authCtx.PrincipalID,
				UserID:         authCtx.UserID,
				APIKeyID:       authCtx.APIKeyID,
				HTTPMethod:     c.Request.Method,
				HTTPPath:       c.Request.URL.Path,
				RoutePattern:   routePattern,
				FeatureKey:     feature.Key,
				CheckedAt:      checkedAt,
			},
		)
		if err != nil {
			m.errorHandler.HandleError(c, err)
			return
		}
		if !result.Allowed {
			m.logDeniedAccess(c, routePattern, result.FeatureKey, result.Reason)
			m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(result.Reason))
			return
		}

		c.Set(featureCheckResultKey, &services.FeatureCheckResult{
			FeatureKey: result.FeatureKey,
			Allowed:    result.Allowed,
			Reason:     result.Reason,
			CheckedAt:  result.CheckedAt,
			FailOpen:   result.FailOpen,
		})

		c.Next()

		if c.Writer.Status() >= http.StatusBadRequest {
			return
		}
		if _, err = m.usage.RecordUsage(c.Request.Context(), &services.UsageRecordRequest{
			OrganizationID: authCtx.OrganizationID,
			BusinessUnitID: authCtx.BusinessUnitID,
			PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
			PrincipalID:    authCtx.PrincipalID,
			UserID:         authCtx.UserID,
			APIKeyID:       authCtx.APIKeyID,
			MeterKey:       platformcatalog.MeterAPIRequests,
			Quantity:       1,
			RecordedAt:     m.now().Unix(),
			IdempotencyKey: idempotencyKey,
		}); err != nil {
			m.logger.Warn(
				"failed to record control-plane API request usage",
				zap.String("route", routePattern),
				zap.Error(err),
			)
		}
	}
}

func (m *ControlPlaneAccessMiddleware) logUnclassifiedRoute(c *gin.Context, routePattern string) {
	authCtx := authctx.GetAuthContext(c)
	m.logger.Error(
		"allowing unclassified protected route through control-plane access middleware",
		zap.String("method", c.Request.Method),
		zap.String("routePattern", routePattern),
		zap.String("path", c.Request.URL.Path),
		zap.String("organizationID", authCtx.OrganizationID.String()),
		zap.String("businessUnitID", authCtx.BusinessUnitID.String()),
	)
}

func (m *ControlPlaneAccessMiddleware) logDeniedAccess(
	c *gin.Context,
	routePattern string,
	featureKey platformcatalog.FeatureKey,
	reason string,
) {
	authCtx := authctx.GetAuthContext(c)
	m.logger.Error(
		"control-plane access denied",
		zap.String("method", c.Request.Method),
		zap.String("routePattern", routePattern),
		zap.String("path", c.Request.URL.Path),
		zap.String("featureKey", string(featureKey)),
		zap.String("reason", reason),
		zap.String("organizationID", authCtx.OrganizationID.String()),
		zap.String("businessUnitID", authCtx.BusinessUnitID.String()),
	)
}

func apiRequestIdempotencyKey(c *gin.Context, routePattern string) (string, error) {
	requestID := strings.TrimSpace(c.GetString("request_id"))
	if requestID == "" {
		requestID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
	}
	if requestID == "" {
		requestID = strings.TrimSpace(c.GetString("X-Request-ID"))
	}
	if requestID == "" {
		return "", errortypes.NewValidationError(
			"requestId",
			errortypes.ErrRequired,
			"request ID is required for deterministic API usage idempotency",
		)
	}

	return "api-request:" + requestID + ":" + c.Request.Method + ":" + routePattern, nil
}
