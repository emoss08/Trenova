package middleware

import (
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const graphQLPath = "/graphql"

var readOnlyWriteAllowedPaths = map[string]struct{}{
	graphQLPath:                                     {},
	"/api/v1/auth/logout":                           {},
	"/api/v1/users/me/change-password":              {},
	"/api/v1/users/me/switch-organization":          {},
	"/api/v1/users/me/settings":                     {},
	"/api/v1/assistant/turns/:turnID/stop":          {},
	"/api/v1/shipments/:shipmentID/comments/typing": {},
}

var readOnlyWriteAllowedActions = map[string]struct{}{
	"backtest":                 {},
	"bulk-preview":             {},
	"calculate-distance":       {},
	"calculate-totals":         {},
	"check-for-duplicate-bols": {},
	"check-hazmat-segregation": {},
	"check-worker-compliance":  {},
	"compose":                  {},
	"edi-214-payload":          {},
	"explain":                  {},
	"export":                   {},
	"inspect":                  {},
	"inspect-certificate":      {},
	"loading-optimization":     {},
	"match":                    {},
	"presence":                 {},
	"preview":                  {},
	"preview-prompt":           {},
	"previous-rates":           {},
	"shop":                     {},
	"simulate":                 {},
	"validate":                 {},
}

var expiredAllowedPaths = map[string]struct{}{
	"/api/v1/auth/logout":                  {},
	"/api/v1/auth/csrf":                    {},
	"/api/v1/users/me":                     {},
	"/api/v1/users/me/organizations":       {},
	"/api/v1/users/me/switch-organization": {},
	"/api/v1/me/billing":                   {},
	"/api/v1/me/entitlements":              {},
	"/api/v1/me/platform-catalog":          {},
}

type ReadOnlyGuardParams struct {
	fx.In

	Plans        services.PlanService
	ErrorHandler *helpers.ErrorHandler
	Logger       *zap.Logger
}

type ReadOnlyGuard struct {
	plans        services.PlanService
	errorHandler *helpers.ErrorHandler
	l            *zap.Logger
}

func NewReadOnlyGuard(p ReadOnlyGuardParams) *ReadOnlyGuard {
	return &ReadOnlyGuard{
		plans:        p.Plans,
		errorHandler: p.ErrorHandler,
		l:            p.Logger.Named("read-only-guard"),
	}
}

func (g *ReadOnlyGuard) Guard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if g.plans == nil || !g.plans.IsCloud() {
			c.Next()
			return
		}

		authCtx := authctx.GetAuthContext(c)
		if authCtx == nil || authCtx.OrganizationID.IsNil() || authCtx.BusinessUnitID.IsNil() {
			c.Next()
			return
		}

		route := routeOf(c)
		err := planservice.Admit(c.Request.Context(), g.plans, &planservice.AdmissionRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			Write:     IsUnsafeMethod(c.Request.Method) && !ReadOnlyWriteAllowed(route),
			APIKey:    authCtx.IsAPIKey(),
			ExpiredOK: ExpiredAllowed(route),
		})
		if err != nil {
			g.l.Debug("request refused by the organization's plan",
				zap.String("route", route),
				zap.String("method", c.Request.Method),
				zap.String("organizationId", authCtx.OrganizationID.String()),
				zap.Error(err),
			)
			g.errorHandler.HandleError(c, err)
			c.Abort()
			return
		}

		c.Next()
	}
}

func IsUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func ReadOnlyWriteAllowed(route string) bool {
	normalized := strings.TrimSuffix(route, "/")
	if _, ok := readOnlyWriteAllowedPaths[normalized]; ok {
		return true
	}

	action := normalized[strings.LastIndexByte(normalized, '/')+1:]
	_, ok := readOnlyWriteAllowedActions[action]

	return ok
}

func ExpiredAllowed(route string) bool {
	_, ok := expiredAllowedPaths[strings.TrimSuffix(route, "/")]
	return ok
}

func routeOf(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}

	return c.Request.URL.Path
}
