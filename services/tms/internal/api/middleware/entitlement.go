package middleware

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const featureCheckResultKey = "feature_check_result"

type EntitlementMiddlewareParams struct {
	fx.In

	EntitlementProvider services.EntitlementProvider
	ErrorHandler        *helpers.ErrorHandler
}

type EntitlementMiddleware struct {
	entitlements services.EntitlementProvider
	errorHandler *helpers.ErrorHandler
}

func NewEntitlementMiddleware(p EntitlementMiddlewareParams) *EntitlementMiddleware {
	return &EntitlementMiddleware{
		entitlements: p.EntitlementProvider,
		errorHandler: p.ErrorHandler,
	}
}

func (m *EntitlementMiddleware) RequireFeature(
	featureKey platformcatalog.FeatureKey,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := m.checkFeature(c, featureKey)
		if err != nil {
			m.errorHandler.HandleError(c, err)
			return
		}

		c.Set(featureCheckResultKey, result)
		if !result.Allowed {
			m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(
				"Your organization is not entitled to this feature",
			))
			return
		}

		c.Next()
	}
}

func (m *EntitlementMiddleware) OptionalFeature(
	featureKey platformcatalog.FeatureKey,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := m.checkFeature(c, featureKey)
		if err != nil {
			m.errorHandler.HandleError(c, err)
			return
		}

		c.Set(featureCheckResultKey, result)
		c.Next()
	}
}

func (m *EntitlementMiddleware) checkFeature(
	c *gin.Context,
	featureKey platformcatalog.FeatureKey,
) (*services.FeatureCheckResult, error) {
	authCtx := authctx.GetAuthContext(c)

	return m.entitlements.CheckFeature(c.Request.Context(), &services.FeatureCheckRequest{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		FeatureKey:     featureKey,
	})
}

func GetFeatureCheckResult(c *gin.Context) *services.FeatureCheckResult {
	result, exists := c.Get(featureCheckResultKey)
	if !exists {
		return nil
	}

	checkResult, ok := result.(*services.FeatureCheckResult)
	if !ok {
		return nil
	}

	return checkResult
}
