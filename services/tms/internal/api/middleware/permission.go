package middleware

import (
	"net/http"

	authctx "github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type PermissionMiddleware struct {
	engine ports.PermissionEngine
}

func NewPermissionMiddleware(engine ports.PermissionEngine) *PermissionMiddleware {
	return &PermissionMiddleware{
		engine: engine,
	}
}

// RequirePermission is a middleware that checks if the user has the required permission
//
// Usage:
//
//	router.POST("/shipments",
//		permMiddleware.RequirePermission("shipment", "create"),
//		handler.CreateShipment,
//	)
func (pm *PermissionMiddleware) RequirePermission(
	resource permission.Resource,
	action string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := authctx.GetUserID(c)
		if !ok {
			pm.handlePermissionError(c, http.StatusUnauthorized, "User not authenticated")
			return
		}

		orgID, ok := authctx.GetOrganizationID(c)
		if !ok {
			pm.handlePermissionError(c, http.StatusUnauthorized, "Organization not found")
			return
		}

		result, err := pm.engine.Check(c.Request.Context(), &ports.PermissionCheckRequest{
			UserID:         userID,
			OrganizationID: orgID,
			ResourceType:   string(resource),
			Action:         action,
		})
		if err != nil {
			pm.handlePermissionError(
				c,
				http.StatusInternalServerError,
				"Failed to check permission",
			)
			return
		}

		if !result.Allowed {
			pm.handlePermissionError(c, http.StatusForbidden, "Insufficient permissions")
			return
		}

		c.Next()
	}
}

// RequireAnyPermission checks if the user has at least one of the specified permissions
//
// Usage:
//
//	router.GET("/shipments/:id",
//		permMiddleware.RequireAnyPermission("shipment", []string{"read", "update"}),
//		handler.GetShipment,
//	)
func (pm *PermissionMiddleware) RequireAnyPermission(
	resource string,
	actions []string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := authctx.GetUserID(c)
		if !ok {
			pm.handlePermissionError(c, http.StatusUnauthorized, "User not authenticated")
			return
		}

		orgID, ok := authctx.GetOrganizationID(c)
		if !ok {
			pm.handlePermissionError(c, http.StatusUnauthorized, "Organization not found")
			return
		}

		for _, action := range actions {
			result, err := pm.engine.Check(c.Request.Context(), &ports.PermissionCheckRequest{
				UserID:         userID,
				OrganizationID: orgID,
				ResourceType:   resource,
				Action:         action,
			})
			if err != nil {
				pm.handlePermissionError(
					c,
					http.StatusInternalServerError,
					"Failed to check permission",
				)
				return
			}

			if result.Allowed {
				c.Next()
				return
			}
		}

		pm.handlePermissionError(c, http.StatusForbidden, "Insufficient permissions")
	}
}

// RequireAllPermissions checks if the user has all of the specified permissions
//
// Usage:
//
//	router.POST("/shipments/:id/approve",
//		permMiddleware.RequireAllPermissions("shipment", []string{"read", "approve"}),
//		handler.ApproveShipment,
//	)
func (pm *PermissionMiddleware) RequireAllPermissions(
	resource string,
	actions []string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := authctx.GetUserID(c)
		if !ok {
			pm.handlePermissionError(c, http.StatusUnauthorized, "User not authenticated")
			return
		}

		orgID, ok := authctx.GetOrganizationID(c)
		if !ok {
			pm.handlePermissionError(c, http.StatusUnauthorized, "Organization not found")
			return
		}

		for _, action := range actions {
			result, err := pm.engine.Check(c.Request.Context(), &ports.PermissionCheckRequest{
				UserID:         userID,
				OrganizationID: orgID,
				ResourceType:   resource,
				Action:         action,
			})
			if err != nil {
				pm.handlePermissionError(
					c,
					http.StatusInternalServerError,
					"Failed to check permission",
				)
				return
			}

			if !result.Allowed {
				// User doesn't have one of the required permissions
				pm.handlePermissionError(c, http.StatusForbidden, "Insufficient permissions")
				return
			}
		}

		// User has all required permissions, continue
		c.Next()
	}
}

// RequireResourcePermission checks permission for a specific resource action
// This is useful for routes with dynamic resource IDs
//
// Usage:
//
//	router.PUT("/shipments/:id",
//		permMiddleware.RequireResourcePermission("shipment", "update"),
//		handler.UpdateShipment,
//	)
func (pm *PermissionMiddleware) RequireResourcePermission(
	resource permission.Resource,
	action string,
) gin.HandlerFunc {
	return pm.RequirePermission(resource, action)
}

// CheckPermissionInHandler is a helper function that can be called within handlers
// for more complex permission logic
//
// Usage within a handler:
//
//	if !pm.CheckPermissionInHandler(c, "shipment", "create") {
//		return // Response already sent
//	}
func (pm *PermissionMiddleware) CheckPermissionInHandler(
	c *gin.Context,
	resource permission.Resource,
	action string,
) bool {
	userID, ok := authctx.GetUserID(c)
	if !ok {
		pm.handlePermissionError(c, http.StatusUnauthorized, "User not authenticated")
		return false
	}

	orgID, ok := authctx.GetOrganizationID(c)
	if !ok {
		pm.handlePermissionError(c, http.StatusUnauthorized, "Organization not found")
		return false
	}

	result, err := pm.engine.Check(c.Request.Context(), &ports.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		ResourceType:   string(resource),
		Action:         action,
	})
	if err != nil {
		pm.handlePermissionError(c, http.StatusInternalServerError, "Failed to check permission")
		return false
	}

	if !result.Allowed {
		pm.handlePermissionError(c, http.StatusForbidden, "Insufficient permissions")
		return false
	}

	return true
}

// OptionalPermission checks permission but doesn't abort if missing
// Instead, it sets a context value that can be checked in the handler
//
// Usage:
//
//	router.GET("/shipments",
//		permMiddleware.OptionalPermission("shipment", "export"),
//		handler.ListShipments, // Handler can check c.GetBool("canExport")
//	)
func (pm *PermissionMiddleware) OptionalPermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := authctx.GetUserID(c)
		if !ok {
			c.Set("has_"+resource+"_"+action, false)
			c.Next()
			return
		}

		orgID, ok := authctx.GetOrganizationID(c)
		if !ok {
			c.Set("has_"+resource+"_"+action, false)
			c.Next()
			return
		}

		result, err := pm.engine.Check(c.Request.Context(), &ports.PermissionCheckRequest{
			UserID:         userID,
			OrganizationID: orgID,
			ResourceType:   resource,
			Action:         action,
		})
		if err != nil || !result.Allowed {
			c.Set("has_"+resource+"_"+action, false)
		} else {
			c.Set("has_"+resource+"_"+action, true)
		}

		c.Next()
	}
}

// handlePermissionError sends a standardized permission error response
func (pm *PermissionMiddleware) handlePermissionError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": message,
		"code":  status,
	})
	c.Abort()
}
