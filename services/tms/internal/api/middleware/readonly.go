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
	graphQLPath:                                                        {},
	"/api/v1/auth/logout":                                              {},
	"/api/v1/users/me/change-password":                                 {},
	"/api/v1/users/me/switch-organization":                             {},
	"/api/v1/users/me/settings":                                        {},
	"/api/v1/users/:userID/permissions/simulate":                       {},
	"/api/v1/assistant/turns/:turnID/stop":                             {},
	"/api/v1/shipments/:shipmentID/comments/typing":                    {},
	"/api/v1/shipments/:shipmentID/comments/presence":                  {},
	"/api/v1/realtime/presence/:resource":                              {},
	"/api/v1/realtime/presence/:resource/:recordID":                    {},
	"/api/v1/shipments/calculate-totals":                               {},
	"/api/v1/shipments/calculate-distance":                             {},
	"/api/v1/shipments/check-for-duplicate-bols":                       {},
	"/api/v1/shipments/check-hazmat-segregation":                       {},
	"/api/v1/shipments/loading-optimization":                           {},
	"/api/v1/shipments/previous-rates":                                 {},
	"/api/v1/assignments/check-worker-compliance":                      {},
	"/api/v1/recurring-shipments/match":                                {},
	"/api/v1/rate-quotes/shipment/:shipmentID/explain":                 {},
	"/api/v1/rate-agreements/rate-increase/preview":                    {},
	"/api/v1/formula-templates/:templateID/backtest":                   {},
	"/api/v1/detention/backtest":                                       {},
	"/api/v1/detention-policies/preview":                               {},
	"/api/v1/billing/invoices/:invoiceID/preview":                      {},
	"/api/v1/billing/invoice-adjustments/preview":                      {},
	"/api/v1/billing/invoice-adjustments/bulk-preview":                 {},
	"/api/v1/billing/invoice-adjustments/drafts/:adjustmentID/preview": {},
	"/api/v1/service-failures/:serviceFailureID/edi-214-payload":       {},
	"/api/v1/reports/dashboards/:dashboardID/export":                   {},
	"/api/v1/tables/:resource/compose":                                 {},
	"/api/v1/agent-definitions/preview-prompt":                         {},
	"/api/v1/document-parsing-rules/versions/:versionID/simulate":      {},
	"/api/v1/edi/documents/preview":                                    {},
	"/api/v1/edi/test-cases/:testCaseID/preview":                       {},
	"/api/v1/edi/communication-profiles/inspect-certificate":           {},
	"/api/v1/edi/templates/:templateID/versions/:versionID/validate":   {},
	"/api/v1/edi/catalog/partner-settings/validate":                    {},
	"/api/v1/edi/x12/inspect":                                          {},
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
		if g.plans == nil || !g.plans.EnforcesPlans() {
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
	_, ok := readOnlyWriteAllowedPaths[strings.TrimSuffix(route, "/")]
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
