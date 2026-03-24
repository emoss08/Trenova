package permissionhandler

import (
	"net/http"
	"sort"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	PermissionEngine services.PermissionEngine
	Registry         *permission.Registry
	ErrorHandler     *helpers.ErrorHandler
}

type Handler struct {
	permEngine services.PermissionEngine
	registry   *permission.Registry
	eh         *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		permEngine: p.PermissionEngine,
		registry:   p.Registry,
		eh:         p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/me/permissions")
	api.GET("", h.getManifest)
	api.GET("/version", h.getVersion)
	api.GET("/:resource", h.getResourcePermissions)
	api.POST("/check", h.checkBatch)

	resources := rg.Group("/permissions")
	resources.GET("/resources", h.getAvailableResources)
	resources.GET("/operations", h.getAvailableOperations)
}

// @Summary Get permission manifest
// @Description Returns the lightweight permission manifest for the authenticated actor.
// @ID getPermissionManifest
// @Tags Permissions
// @Produce json
// @Success 200 {object} services.LightPermissionManifest
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /me/permissions [get]
func (h *Handler) getManifest(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	manifest, err := h.permEngine.GetLightManifest(
		c.Request.Context(),
		authCtx.UserID,
		authCtx.OrganizationID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, manifest)
}

// @Summary Get permission manifest version
// @Description Returns the manifest checksum and expiration timestamp for cache validation.
// @ID getPermissionManifestVersion
// @Tags Permissions
// @Produce json
// @Success 200 {object} gin.H
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /me/permissions/version [get]
func (h *Handler) getVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	manifest, err := h.permEngine.GetLightManifest(
		c.Request.Context(),
		authCtx.UserID,
		authCtx.OrganizationID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"checksum":  manifest.Checksum,
		"expiresAt": manifest.ExpiresAt,
	})
}

// @Summary Get resource permissions
// @Description Returns the effective permissions for a single resource.
// @ID getResourcePermissions
// @Tags Permissions
// @Produce json
// @Param resource path string true "Resource name"
// @Success 200 {object} services.ResourcePermissionDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} gin.H
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /me/permissions/{resource} [get]
func (h *Handler) getResourcePermissions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	resource := c.Param("resource")

	detail, err := h.permEngine.GetResourcePermissions(
		c.Request.Context(),
		authCtx.UserID,
		authCtx.OrganizationID,
		resource,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

type batchCheckRequest struct {
	Checks []struct {
		Resource  string               `json:"resource"`
		Operation permission.Operation `json:"operation"`
	} `json:"checks"`
}

// @Summary Check permissions in batch
// @Description Checks multiple resource-operation pairs for the authenticated actor.
// @ID checkPermissionsBatch
// @Tags Permissions
// @Accept json
// @Produce json
// @Param request body batchCheckRequest true "Batch permission check request"
// @Success 200 {object} services.BatchPermissionCheckResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /me/permissions/check [post]
func (h *Handler) checkBatch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req batchCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	checks := make([]services.ResourceOperationCheck, len(req.Checks))
	for i, check := range req.Checks {
		checks[i] = services.ResourceOperationCheck{
			Resource:  check.Resource,
			Operation: check.Operation,
		}
	}

	result, err := h.permEngine.CheckBatch(
		c.Request.Context(),
		&services.BatchPermissionCheckRequest{
			PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
			PrincipalID:    authCtx.PrincipalID,
			UserID:         authCtx.UserID,
			APIKeyID:       authCtx.APIKeyID,
			BusinessUnitID: authCtx.BusinessUnitID,
			OrganizationID: authCtx.OrganizationID,
			Checks:         checks,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

type resourceCategoryResponse struct {
	Category  string                           `json:"category"`
	Resources []*permission.ResourceDefinition `json:"resources"`
}

// @Summary List available permission resources
// @Description Returns all registered permission resources grouped by category.
// @ID getAvailablePermissionResources
// @Tags Permissions
// @Produce json
// @Success 200 {array} resourceCategoryResponse
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /permissions/resources [get]
func (h *Handler) getAvailableResources(c *gin.Context) {
	categories := h.registry.GetCategories()
	sort.Strings(categories)

	response := make([]resourceCategoryResponse, 0, len(categories))
	for _, cat := range categories {
		resources := h.registry.GetByCategory(cat)
		sort.Slice(resources, func(i, j int) bool {
			return resources[i].DisplayName < resources[j].DisplayName
		})
		response = append(response, resourceCategoryResponse{
			Category:  cat,
			Resources: resources,
		})
	}

	c.JSON(http.StatusOK, response)
}

// @Summary List available permission operations
// @Description Returns all registered permission operations.
// @ID getAvailablePermissionOperations
// @Tags Permissions
// @Produce json
// @Success 200 {array} permission.Operation
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /permissions/operations [get]
func (h *Handler) getAvailableOperations(c *gin.Context) {
	c.JSON(http.StatusOK, permission.GetAllOperations())
}
