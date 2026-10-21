package permissionhandler

import (
	"context"
	"net/http"
	"sort"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	PermissionEngine permissionports.PermissionEngine
	Registry         *permission.Registry
	Scopes           permissionports.ScopePolicy
	ErrorHandler     *helpers.ErrorHandler
}

type Handler struct {
	permEngine permissionports.PermissionEngine
	registry   *permission.Registry
	scopes     permissionports.ScopePolicy
	eh         *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		permEngine: p.PermissionEngine,
		registry:   p.Registry,
		scopes:     p.Scopes,
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
	resources.GET("/bundles", h.getBundles)
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

	helpers.JSON(c, http.StatusOK, manifest)
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

	helpers.JSON(c, http.StatusOK, gin.H{
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
		helpers.JSON(c, http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}

	helpers.JSON(c, http.StatusOK, detail)
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

	checks := make([]permissionports.ResourceOperationCheck, len(req.Checks))
	for i, check := range req.Checks {
		checks[i] = permissionports.ResourceOperationCheck{
			Resource:  check.Resource,
			Operation: check.Operation,
		}
	}

	result, err := h.permEngine.CheckBatch(
		c.Request.Context(),
		&permissionports.BatchPermissionCheckRequest{
			PrincipalType:  kernelports.PrincipalType(authCtx.PrincipalType),
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

	helpers.JSON(c, http.StatusOK, result)
}

type resourceCategoryResponse struct {
	Category  string             `json:"category"`
	Resources []resourceResponse `json:"resources"`
}

type resourceResponse struct {
	*permission.ResourceDefinition
	EnforceableScopes []permission.DataScope `json:"enforceableScopes"`
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
	ctx := c.Request.Context()
	categories := h.registry.GetCategories()

	response := make([]resourceCategoryResponse, 0, len(categories))
	for _, cat := range categories {
		resources := h.registry.GetByCategory(cat)
		entries := make([]resourceResponse, 0, len(resources))
		for _, def := range resources {
			entries = append(entries, resourceResponse{
				ResourceDefinition: translatedDefinition(ctx, def),
				EnforceableScopes:  h.scopes.EnforceableScopes(def.Resource),
			})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].DisplayName < entries[j].DisplayName
		})
		response = append(response, resourceCategoryResponse{
			Category:  i18n.T(ctx, cat),
			Resources: entries,
		})
	}

	helpers.JSON(c, http.StatusOK, response)
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
	helpers.JSON(c, http.StatusOK, permission.GetAllOperations())
}

type bundleGrantResponse struct {
	Resource    string                 `json:"resource"`
	DisplayName string                 `json:"displayName"`
	Operations  []permission.Operation `json:"operations"`
	DataScope   permission.DataScope   `json:"dataScope"`
}

type bundleResponse struct {
	ID          string                `json:"id"`
	Module      string                `json:"module"`
	Label       string                `json:"label"`
	Description string                `json:"description"`
	Grants      []bundleGrantResponse `json:"grants"`
}

type jobTemplateResponse struct {
	ID             string                      `json:"id"`
	Label          string                      `json:"label"`
	Description    string                      `json:"description"`
	Bundles        []string                    `json:"bundles"`
	MaxSensitivity permission.FieldSensitivity `json:"maxSensitivity"`
	Grants         []bundleGrantResponse       `json:"grants"`
}

type bundlesResponse struct {
	Bundles   []bundleResponse      `json:"bundles"`
	Templates []jobTemplateResponse `json:"templates"`
}

// @Summary List capability bundles and job templates
// @Description Returns the capability bundles roles are composed from and the job templates
// @Description built from them, each with the grants it applies in this deployment.
// @ID getPermissionBundles
// @Tags Permissions
// @Produce json
// @Success 200 {object} bundlesResponse
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /permissions/bundles [get]
func (h *Handler) getBundles(c *gin.Context) {
	ctx := c.Request.Context()
	bundles := permission.Bundles()
	templates := permission.JobTemplates()

	resp := bundlesResponse{
		Bundles:   make([]bundleResponse, 0, len(bundles)),
		Templates: make([]jobTemplateResponse, 0, len(templates)),
	}
	for _, bundle := range bundles {
		grants := make([]bundleGrantResponse, 0, len(bundle.Grants))
		for _, grant := range bundle.Grants {
			grants = append(grants, h.bundleGrant(
				ctx,
				grant.Resource.String(),
				grant.Operations,
				grant.Scope,
			))
		}
		resp.Bundles = append(resp.Bundles, bundleResponse{
			ID:          bundle.ID,
			Module:      bundle.Module,
			Label:       i18n.T(ctx, bundle.Label),
			Description: i18n.T(ctx, bundle.Description),
			Grants:      grants,
		})
	}
	for _, template := range templates {
		composed := h.registry.TemplateGrants(template)
		grants := make([]bundleGrantResponse, 0, len(composed))
		for _, rp := range composed {
			grants = append(grants, h.bundleGrant(ctx, rp.Resource, rp.Operations, rp.DataScope))
		}
		resp.Templates = append(resp.Templates, jobTemplateResponse{
			ID:             template.ID,
			Label:          i18n.T(ctx, template.Label),
			Description:    i18n.T(ctx, template.Description),
			Bundles:        template.Bundles,
			MaxSensitivity: template.MaxSensitivity,
			Grants:         grants,
		})
	}

	helpers.JSON(c, http.StatusOK, resp)
}

func (h *Handler) bundleGrant(
	ctx context.Context,
	resource string,
	operations []permission.Operation,
	scope permission.DataScope,
) bundleGrantResponse {
	if scope == "" || !h.scopes.Enforced(resource, scope) {
		scope = permission.DataScopeOrganization
	}
	name := resource
	if def, ok := h.registry.Get(resource); ok {
		name = i18n.T(ctx, def.DisplayName)
	}
	return bundleGrantResponse{
		Resource:    resource,
		DisplayName: name,
		Operations:  operations,
		DataScope:   scope,
	}
}

func translatedDefinition(
	ctx context.Context,
	def *permission.ResourceDefinition,
) *permission.ResourceDefinition {
	translated := *def
	translated.DisplayName = i18n.T(ctx, def.DisplayName)
	translated.Description = i18n.T(ctx, def.Description)
	translated.Category = i18n.T(ctx, def.Category)
	translated.Operations = make([]permission.OperationDefinition, len(def.Operations))
	for i, op := range def.Operations {
		op.DisplayName = i18n.T(ctx, op.DisplayName)
		op.Description = i18n.T(ctx, op.Description)
		translated.Operations[i] = op
	}
	return &translated
}
