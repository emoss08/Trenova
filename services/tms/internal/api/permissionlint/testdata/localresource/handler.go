package localresource

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	pm *middleware.PermissionMiddleware
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/organizations/:id/iam/")
	h.registerIdentityProviderRoutes(api)
	h.registerSCIMRoutes(api)
	api.GET(
		"auth-events",
		h.pm.RequirePermission(permission.ResourceAuthEvent.String(), permission.OpRead),
		h.handle,
	)
}

func (h *Handler) registerIdentityProviderRoutes(api *gin.RouterGroup) {
	resource := permission.ResourceIdentityProvider.String()
	api.DELETE(
		"identity-providers/:providerId",
		h.pm.RequirePermission(resource, permission.OpDelete),
		h.handle,
	)
}

func (h *Handler) registerSCIMRoutes(api *gin.RouterGroup) {
	resource := permission.ResourceSCIMDirectory.String()
	update := h.pm.RequirePermission(resource, permission.OpUpdate)

	api.POST("scim/tokens/:tokenId/revoke", update, h.handle)
}

func (h *Handler) handle(*gin.Context) {}
