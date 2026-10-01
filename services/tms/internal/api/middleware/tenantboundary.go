package middleware

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/tenantboundary"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TenantBoundaryParams struct {
	fx.In

	Auditor services.SecurityAuditor
	Logger  *zap.Logger
}

type TenantBoundaryMiddleware struct {
	auditor services.SecurityAuditor
	l       *zap.Logger
}

func NewTenantBoundaryMiddleware(p TenantBoundaryParams) *TenantBoundaryMiddleware {
	return &TenantBoundaryMiddleware{
		auditor: p.Auditor,
		l:       p.Logger.Named("middleware.tenant-boundary"),
	}
}

func (m *TenantBoundaryMiddleware) Track() gin.HandlerFunc {
	return func(c *gin.Context) {
		tracker := tenantboundary.NewTracker()
		c.Set(tenantboundary.GinContextKey, tracker)
		c.Request = c.Request.WithContext(tenantboundary.With(c.Request.Context(), tracker))

		c.Next()

		if violations := tracker.Violations(); len(violations) > 0 {
			m.report(c, violations)
		}
	}
}

func (m *TenantBoundaryMiddleware) report(c *gin.Context, violations []tenantboundary.Violation) {
	authCtx := authctx.GetAuthContext(c)
	route := c.FullPath()

	for _, violation := range violations {
		m.l.Warn("refused a cross-tenant request",
			zap.String("source", string(violation.Source)),
			zap.String("field", violation.Field),
			zap.String("attemptedOrganizationID", violation.OrganizationID.String()),
			zap.String("attemptedBusinessUnitID", violation.BusinessUnitID.String()),
			zap.String("organizationID", authCtx.OrganizationID.String()),
			zap.String("principalID", authCtx.PrincipalID.String()),
			zap.String("method", c.Request.Method),
			zap.String("route", route),
		)

		if authCtx.OrganizationID.IsNil() || authCtx.BusinessUnitID.IsNil() {
			continue
		}

		m.auditor.RecordChange(c.Request.Context(), &services.SecurityChange{
			Resource:       permission.ResourceOrganization,
			ResourceID:     violationTarget(violation, authCtx),
			Operation:      operationForMethod(c.Request.Method),
			Actor:          actorutil.FromAuthContext(authCtx).AuditActor(),
			OrganizationID: authCtx.OrganizationID,
			BusinessUnitID: authCtx.BusinessUnitID,
			Comment:        "Cross-tenant request refused",
			Metadata: map[string]any{
				"source":                  string(violation.Source),
				"field":                   violation.Field,
				"attemptedOrganizationId": violation.OrganizationID.String(),
				"attemptedBusinessUnitId": violation.BusinessUnitID.String(),
				"method":                  c.Request.Method,
				"route":                   route,
				"status":                  c.Writer.Status(),
			},
		})
	}
}

func violationTarget(violation tenantboundary.Violation, authCtx *authctx.AuthContext) string {
	switch {
	case violation.OrganizationID.IsNotNil():
		return violation.OrganizationID.String()
	case violation.BusinessUnitID.IsNotNil():
		return violation.BusinessUnitID.String()
	default:
		return authCtx.OrganizationID.String()
	}
}

func operationForMethod(method string) permission.Operation {
	switch method {
	case http.MethodPost:
		return permission.OpCreate
	case http.MethodPut, http.MethodPatch:
		return permission.OpUpdate
	case http.MethodDelete:
		return permission.OpDelete
	default:
		return permission.OpRead
	}
}
