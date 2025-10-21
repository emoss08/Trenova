package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/permissionregistry"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type PermissionHandlerParams struct {
	fx.In

	Registry         *permissionregistry.Registry
	PermissionEngine ports.PermissionEngine
	ErrorHandler     *helpers.ErrorHandler
}

type PermissionHandler struct {
	registry *permissionregistry.Registry
	engine   ports.PermissionEngine
	eh       *helpers.ErrorHandler
}

func NewPermissionHandler(p PermissionHandlerParams) *PermissionHandler {
	return &PermissionHandler{
		registry: p.Registry,
		engine:   p.PermissionEngine,
		eh:       p.ErrorHandler,
	}
}

func (h *PermissionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	permissions := rg.Group("/permissions/")
	permissions.GET("manifest/", h.getManifest)
	permissions.POST("verify/", h.verifyPermission)
	permissions.POST("check-batch/", h.checkBatch)
	permissions.POST("refresh/", h.refreshPermissions)
	permissions.POST("invalidate-cache/", h.invalidateCache)
	permissions.POST("switch-organization/", h.switchOrganization)
	permissions.GET("registry/", h.getRegistry)
	permissions.GET("registry/:resource/", h.getResourceMetadata)
}

type GetManifestResponse struct {
	Version          string                      `json:"version"`
	UserID           string                      `json:"userId"`
	CurrentOrgID     string                      `json:"currentOrgId"`
	AvailableOrgsIDs []string                    `json:"availableOrgsIds"`
	ComputedAt       int64                       `json:"computedAt"`
	ExpiresAt        int64                       `json:"expiresAt"`
	Resources        ports.ResourcePermissionMap `json:"resources"`
	Checksum         string                      `json:"checksum"`
}

func (h *PermissionHandler) getManifest(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	manifest, err := h.engine.GetUserPermissions(
		c.Request.Context(),
		authCtx.UserID,
		authCtx.OrganizationID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	availableOrgs := make([]string, len(manifest.AvailableOrgs))
	for i, org := range manifest.AvailableOrgs {
		availableOrgs[i] = org.String()
	}

	response := GetManifestResponse{
		Version:          manifest.Version,
		UserID:           manifest.UserID.String(),
		CurrentOrgID:     manifest.CurrentOrg.String(),
		AvailableOrgsIDs: availableOrgs,
		Resources:        manifest.Resources,
		Checksum:         manifest.Checksum,
		ComputedAt:       manifest.ComputedAt.Unix(),
		ExpiresAt:        manifest.ExpiresAt.Unix(),
	}

	c.JSON(http.StatusOK, response)
}

type VerifyPermissionRequest struct {
	ResourceType string         `json:"resourceType"         binding:"required"`
	Action       string         `json:"action"               binding:"required"`
	ResourceID   *string        `json:"resourceId,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
}

func (h *PermissionHandler) verifyPermission(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req VerifyPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError("request", errortypes.ErrInvalidFormat, err.Error()),
		)
		return
	}

	var resourceID *pulid.ID
	if req.ResourceID != nil {
		parsed, err := pulid.Parse(*req.ResourceID)
		if err != nil {
			h.eh.HandleError(
				c,
				errortypes.NewValidationError(
					"resourceId",
					errortypes.ErrInvalidFormat,
					"Invalid resource ID",
				),
			)
			return
		}
		resourceID = &parsed
	}

	checkReq := &ports.PermissionCheckRequest{
		UserID:         authCtx.UserID,
		OrganizationID: authCtx.OrganizationID,
		ResourceType:   req.ResourceType,
		Action:         req.Action,
		ResourceID:     resourceID,
		Context:        req.Context,
	}

	result, err := h.engine.Check(c.Request.Context(), checkReq)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

type CheckBatchRequest struct {
	Checks []VerifyPermissionRequest `json:"checks" binding:"required,min=1,max=100"`
}

func (h *PermissionHandler) checkBatch(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req CheckBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError("request", errortypes.ErrInvalidFormat, err.Error()),
		)
		return
	}

	checks := make([]*ports.PermissionCheckRequest, len(req.Checks))
	for i, check := range req.Checks {
		var resourceID *pulid.ID
		if check.ResourceID != nil {
			parsed, err := pulid.Parse(*check.ResourceID)
			if err != nil {
				h.eh.HandleError(
					c,
					errortypes.NewValidationError(
						"resourceId",
						errortypes.ErrInvalidFormat,
						"Invalid resource ID at index "+string(rune(i)),
					),
				)
				return
			}
			resourceID = &parsed
		}

		checks[i] = &ports.PermissionCheckRequest{
			UserID:         authCtx.UserID,
			OrganizationID: authCtx.OrganizationID,
			ResourceType:   check.ResourceType,
			Action:         check.Action,
			ResourceID:     resourceID,
			Context:        check.Context,
		}
	}

	batchReq := &ports.BatchPermissionCheckRequest{
		UserID:         authCtx.UserID,
		OrganizationID: authCtx.OrganizationID,
		Checks:         checks,
	}

	result, err := h.engine.CheckBatch(c.Request.Context(), batchReq)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PermissionHandler) refreshPermissions(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	if err := h.engine.RefreshUserPermissions(c.Request.Context(), authCtx.UserID, authCtx.OrganizationID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permissions refreshed successfully"})
}

func (h *PermissionHandler) invalidateCache(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	if err := h.engine.InvalidateCache(c.Request.Context(), authCtx.UserID, authCtx.OrganizationID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cache invalidated successfully"})
}

type SwitchOrganizationRequest struct {
	OrganizationID string `json:"organizationId" binding:"required"`
}

type SwitchOrganizationResponse struct {
	Message        string                      `json:"message"`
	OrganizationID string                      `json:"organizationId"`
	Permissions    ports.ResourcePermissionMap `json:"permissions"`
}

func (h *PermissionHandler) switchOrganization(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req SwitchOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError("request", errortypes.ErrInvalidFormat, err.Error()),
		)
		return
	}

	newOrgID, err := pulid.Parse(req.OrganizationID)
	if err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"organizationId",
				errortypes.ErrInvalidFormat,
				"Invalid organization ID",
			),
		)
		return
	}

	manifest, err := h.engine.GetUserPermissions(c.Request.Context(), authCtx.UserID, newOrgID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	context.SetOrganizationID(c, newOrgID)

	response := SwitchOrganizationResponse{
		Message:        "Organization switched successfully",
		OrganizationID: newOrgID.String(),
		Permissions:    manifest.Resources,
	}

	c.JSON(http.StatusOK, response)
}

func (h *PermissionHandler) getRegistry(c *gin.Context) {
	resources := h.registry.GetAllResources()

	response := make(map[string]any)
	for name, res := range resources {
		resourceData := map[string]any{
			"name":                        name,
			"operations":                  res.GetSupportedOperations(),
			"compositeOperations":         res.GetCompositeOperations(),
			"defaultOperation":            res.GetDefaultOperation(),
			"operationsRequiringApproval": res.GetOperationsRequiringApproval(),
		}

		response[name] = resourceData
	}

	c.JSON(http.StatusOK, response)
}

func (h *PermissionHandler) getResourceMetadata(c *gin.Context) {
	resourceName := c.Param("resource")
	if resourceName == "" {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"resource",
				errortypes.ErrInvalidFormat,
				"Resource name is required",
			),
		)
		return
	}

	res, exists := h.registry.GetResource(resourceName)
	if !exists {
		h.eh.HandleError(
			c,
			errortypes.NewNotFoundError("Resource not found in registry"),
		)
		return
	}

	response := map[string]any{
		"name":                        res.GetResourceName(),
		"operations":                  res.GetSupportedOperations(),
		"compositeOperations":         res.GetCompositeOperations(),
		"defaultOperation":            res.GetDefaultOperation(),
		"operationsRequiringApproval": res.GetOperationsRequiringApproval(),
	}

	c.JSON(http.StatusOK, response)
}
