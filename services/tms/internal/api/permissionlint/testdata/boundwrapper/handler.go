package boundwrapper

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	perm *middleware.PermissionMiddleware
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/roles")
	require := h.requireRolePermission

	api.GET("/", require(permission.OpRead), h.list)
	api.DELETE("/assignments/:assignmentID", h.requireUserPermission(permission.OpUnassign), h.list)
	api.POST("/:roleID/assignments", h.requireUserPermission(permission.OpAssign), h.list)
}

func (h *Handler) requireRolePermission(operation permission.Operation) gin.HandlerFunc {
	return h.perm.RequirePermission(permission.ResourceRole.String(), operation)
}

func (h *Handler) requireUserPermission(operation permission.Operation) gin.HandlerFunc {
	return h.perm.RequirePermission(permission.ResourceUser.String(), operation)
}

func (h *Handler) list(*gin.Context) {}
