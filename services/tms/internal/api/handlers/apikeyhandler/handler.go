package apikeyhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/apikeyservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	ApiKeyService        *apikeyservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	apiKeyService *apikeyservice.Service
	pm            *middleware.PermissionMiddleware
	eh            *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		apiKeyService: p.ApiKeyService,
		pm:            p.PermissionMiddleware,
		eh:            p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/api-keys")

	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceAPIKey.String(), permission.OpRead),
		h.list,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceAPIKey.String(), permission.OpCreate),
		h.create,
	)
	api.GET(
		"/allowed-resources",
		h.pm.RequirePermission(permission.ResourceAPIKey.String(), permission.OpRead),
		h.getAllowedResources,
	)
	api.GET(
		"/:apiKeyID/",
		h.pm.RequirePermission(permission.ResourceAPIKey.String(), permission.OpRead),
		h.get,
	)
	api.PUT(
		"/:apiKeyID/",
		h.pm.RequirePermission(permission.ResourceAPIKey.String(), permission.OpUpdate),
		h.update,
	)
	api.POST(
		"/:apiKeyID/rotate/",
		h.pm.RequirePermission(permission.ResourceAPIKey.String(), permission.OpUpdate),
		h.rotate,
	)
	api.POST(
		"/:apiKeyID/revoke/",
		h.pm.RequirePermission(permission.ResourceAPIKey.String(), permission.OpUpdate),
		h.revoke,
	)
}

// @Summary List API keys
// @ID listAPIKeys
// @Tags API Keys
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]services.APIKeyResponse]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /api-keys/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if !h.isSessionPrincipal(authCtx) {
		h.eh.HandleError(c, errortypes.NewAuthorizationError("API keys cannot manage API keys"))
		return
	}
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[services.APIKeyResponse], error) {
		return h.apiKeyService.ListAPIKeys(
			c.Request.Context(),
			&repositories.ListAPIKeysRequest{
				Filter: req,
			},
		)
	})
}

// @Summary Create an API key
// @ID createAPIKey
// @Tags API Keys
// @Accept json
// @Produce json
// @Param request body services.CreateAPIKeyRequest true "API key payload"
// @Success 201 {object} services.APIKeySecretResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /api-keys/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if !h.isSessionPrincipal(authCtx) {
		h.eh.HandleError(c, errortypes.NewAuthorizationError("API keys cannot manage API keys"))
		return
	}

	req := new(services.CreateAPIKeyRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.apiKeyService.CreateAPIKey(c.Request.Context(), pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}, req, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// @Summary Get an API key
// @ID getAPIKey
// @Tags API Keys
// @Produce json
// @Param apiKeyID path string true "API key ID"
// @Success 200 {object} services.APIKeyResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /api-keys/{apiKeyID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if !h.isSessionPrincipal(authCtx) {
		h.eh.HandleError(c, errortypes.NewAuthorizationError("API keys cannot manage API keys"))
		return
	}

	apiKeyID, err := pulid.MustParse(c.Param("apiKeyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, svcErr := h.apiKeyService.GetAPIKey(c.Request.Context(), pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}, apiKeyID)
	if svcErr != nil {
		h.eh.HandleError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Update an API key
// @ID updateAPIKey
// @Tags API Keys
// @Accept json
// @Produce json
// @Param apiKeyID path string true "API key ID"
// @Param request body services.UpdateAPIKeyRequest true "API key payload"
// @Success 200 {object} services.APIKeyResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /api-keys/{apiKeyID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if !h.isSessionPrincipal(authCtx) {
		h.eh.HandleError(c, errortypes.NewAuthorizationError("API keys cannot manage API keys"))
		return
	}

	apiKeyID, err := pulid.MustParse(c.Param("apiKeyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(services.UpdateAPIKeyRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, svcErr := h.apiKeyService.UpdateAPIKey(c.Request.Context(), pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}, apiKeyID, req)
	if svcErr != nil {
		h.eh.HandleError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Rotate an API key
// @ID rotateAPIKey
// @Tags API Keys
// @Produce json
// @Param apiKeyID path string true "API key ID"
// @Success 200 {object} services.APIKeySecretResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /api-keys/{apiKeyID}/rotate/ [post]
func (h *Handler) rotate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if !h.isSessionPrincipal(authCtx) {
		h.eh.HandleError(c, errortypes.NewAuthorizationError("API keys cannot manage API keys"))
		return
	}

	apiKeyID, err := pulid.MustParse(c.Param("apiKeyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, svcErr := h.apiKeyService.RotateAPIKey(c.Request.Context(), pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}, apiKeyID)
	if svcErr != nil {
		h.eh.HandleError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Revoke an API key
// @ID revokeAPIKey
// @Tags API Keys
// @Produce json
// @Param apiKeyID path string true "API key ID"
// @Success 200 {object} services.APIKeyResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /api-keys/{apiKeyID}/revoke/ [post]
func (h *Handler) revoke(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if !h.isSessionPrincipal(authCtx) {
		h.eh.HandleError(c, errortypes.NewAuthorizationError("API keys cannot manage API keys"))
		return
	}

	apiKeyID, err := pulid.MustParse(c.Param("apiKeyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, svcErr := h.apiKeyService.RevokeAPIKey(c.Request.Context(), pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}, apiKeyID, authCtx.UserID)
	if svcErr != nil {
		h.eh.HandleError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary List allowed API key resources
// @ID listAllowedAPIKeyResources
// @Tags API Keys
// @Produce json
// @Success 200 {array} apikeyservice.AllowedResourceCategory
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /api-keys/allowed-resources [get]
func (h *Handler) getAllowedResources(c *gin.Context) {
	c.JSON(http.StatusOK, h.apiKeyService.GetAllowedResources())
}

func (h *Handler) isSessionPrincipal(authCtx *authctx.AuthContext) bool {
	return authCtx.PrincipalType == "" || authCtx.PrincipalType == authctx.PrincipalTypeUser
}
