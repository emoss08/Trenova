package middleware

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type PermissionMiddlewareParams struct {
	fx.In

	PermissionEngine services.PermissionEngine
	ErrorHandler     *helpers.ErrorHandler
}

type PermissionMiddleware struct {
	permEngine   services.PermissionEngine
	errorHandler *helpers.ErrorHandler
}

func NewPermissionMiddleware(p PermissionMiddlewareParams) *PermissionMiddleware {
	return &PermissionMiddleware{
		permEngine:   p.PermissionEngine,
		errorHandler: p.ErrorHandler,
	}
}

func (m *PermissionMiddleware) RequirePermission(
	resource string,
	operation permission.Operation,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := authctx.GetAuthContext(c)

		result, err := m.permEngine.Check(c.Request.Context(), &services.PermissionCheckRequest{
			PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
			PrincipalID:    authCtx.PrincipalID,
			UserID:         authCtx.UserID,
			APIKeyID:       authCtx.APIKeyID,
			BusinessUnitID: authCtx.BusinessUnitID,
			OrganizationID: authCtx.OrganizationID,
			Resource:       resource,
			Operation:      operation,
		})
		if err != nil {
			m.errorHandler.HandleError(c, err)
			return
		}

		if !result.Allowed {
			m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(
				"You don't have permission to perform this action",
			))
			return
		}

		c.Set("permission_result", result)
		c.Next()
	}
}

func (m *PermissionMiddleware) RequireAnyPermission(checks ...struct {
	Resource  string
	Operation permission.Operation
},
) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := authctx.GetAuthContext(c)

		for _, check := range checks {
			result, err := m.permEngine.Check(c.Request.Context(), &services.PermissionCheckRequest{
				PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
				PrincipalID:    authCtx.PrincipalID,
				UserID:         authCtx.UserID,
				APIKeyID:       authCtx.APIKeyID,
				BusinessUnitID: authCtx.BusinessUnitID,
				OrganizationID: authCtx.OrganizationID,
				Resource:       check.Resource,
				Operation:      check.Operation,
			})
			if err != nil {
				continue
			}

			if result.Allowed {
				c.Set("permission_result", result)
				c.Next()
				return
			}
		}

		m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(
			"You don't have permission to perform this action",
		))
	}
}

func (m *PermissionMiddleware) RequireAllPermissions(checks ...struct {
	Resource  string
	Operation permission.Operation
},
) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := authctx.GetAuthContext(c)

		for _, check := range checks {
			result, err := m.permEngine.Check(c.Request.Context(), &services.PermissionCheckRequest{
				PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
				PrincipalID:    authCtx.PrincipalID,
				UserID:         authCtx.UserID,
				APIKeyID:       authCtx.APIKeyID,
				BusinessUnitID: authCtx.BusinessUnitID,
				OrganizationID: authCtx.OrganizationID,
				Resource:       check.Resource,
				Operation:      check.Operation,
			})
			if err != nil {
				m.errorHandler.HandleError(c, err)
				return
			}

			if !result.Allowed {
				m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(
					"You don't have permission to perform this action",
				))
				return
			}
		}

		c.Next()
	}
}

func (m *PermissionMiddleware) RequirePlatformAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := authctx.GetAuthContext(c)
		if authCtx.IsAPIKey() {
			m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(
				"API keys cannot perform platform admin actions",
			))
			return
		}

		manifest, err := m.permEngine.GetLightManifest(
			c.Request.Context(),
			authCtx.UserID,
			authCtx.OrganizationID,
		)
		if err != nil {
			m.errorHandler.HandleError(c, err)
			return
		}

		if !manifest.IsPlatformAdmin {
			m.errorHandler.HandleError(c, errortypes.NewAuthorizationError(
				"You don't have permission to perform this action",
			))
			return
		}

		c.Next()
	}
}

func GetPermissionResult(c *gin.Context) *services.PermissionCheckResult {
	if result, exists := c.Get("permission_result"); exists {
		if pr, ok := result.(*services.PermissionCheckResult); ok {
			return pr
		}
	}
	return nil
}
