package middleware

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const FeatureCheckResultKey = "feature_check_result"

type EntitlementMiddlewareParams struct {
	fx.In

	EntitlementProvider platformcatalogports.EntitlementProvider
	ErrorHandler        *helpers.ErrorHandler
}

type EntitlementMiddleware struct {
	entitlements platformcatalogports.EntitlementProvider
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

		c.Set(FeatureCheckResultKey, result)
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

		c.Set(FeatureCheckResultKey, result)
		c.Next()
	}
}

func (m *EntitlementMiddleware) checkFeature(
	c *gin.Context,
	featureKey platformcatalog.FeatureKey,
) (*platformcatalogports.FeatureCheckResult, error) {
	authCtx := authctx.GetAuthContext(c)

	return m.entitlements.CheckFeature(
		c.Request.Context(),
		&platformcatalogports.FeatureCheckRequest{
			OrganizationID: authCtx.OrganizationID,
			BusinessUnitID: authCtx.BusinessUnitID,
			PrincipalType:  kernelports.PrincipalType(authCtx.PrincipalType),
			PrincipalID:    authCtx.PrincipalID,
			UserID:         authCtx.UserID,
			APIKeyID:       authCtx.APIKeyID,
			FeatureKey:     featureKey,
		},
	)
}

func GetFeatureCheckResult(c *gin.Context) *platformcatalogports.FeatureCheckResult {
	result, exists := c.Get(FeatureCheckResultKey)
	if !exists {
		return nil
	}

	checkResult, ok := result.(*platformcatalogports.FeatureCheckResult)
	if !ok {
		return nil
	}

	return checkResult
}
