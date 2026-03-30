package organizationhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              services.OrganizationService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service services.OrganizationService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/organizations/")
	api.GET(
		"/:id",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpRead),
		h.get,
	)
	api.PUT(
		"/:id",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpUpdate),
		h.update,
	)
	api.POST(
		"/:id/logo",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpUpdate),
		h.uploadLogo,
	)
	api.GET(
		"/:id/logo",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpRead),
		h.getLogoURL,
	)
	api.DELETE(
		"/:id/logo",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpUpdate),
		h.deleteLogo,
	)
	api.GET(
		"/:id/microsoft-sso",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpRead),
		h.getMicrosoftSSOConfig,
	)
	api.PUT(
		"/:id/microsoft-sso",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpUpdate),
		h.upsertMicrosoftSSOConfig,
	)
}

// @Summary Get an organization
// @ID getOrganization
// @Tags Organizations
// @Produce json
// @Param id path string true "Organization ID"
// @Param includeState query bool false "Include state details"
// @Param includeBu query bool false "Include business unit details"
// @Success 200 {object} tenant.Organization
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /organizations/{id} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	orgID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(c.Request.Context(), repositories.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  authCtx.BusinessUnitID,
		},
		IncludeState: helpers.QueryBool(c, "includeState", false),
		IncludeBU:    helpers.QueryBool(c, "includeBu", false),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Update an organization
// @ID updateOrganization
// @Tags Organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Param request body tenant.Organization true "Organization payload"
// @Success 200 {object} tenant.Organization
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /organizations/{id} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	orgID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(tenant.Organization)
	entity.ID = orgID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updatedEntity, err := h.service.Update(c.Request.Context(), entity)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

// @Summary Upload an organization logo
// @ID uploadOrganizationLogo
// @Tags Organizations
// @Accept mpfd
// @Produce json
// @Param id path string true "Organization ID"
// @Param file formData file true "Logo file"
// @Success 200 {object} tenant.Organization
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /organizations/{id}/logo [post]
func (h *Handler) uploadLogo(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	orgID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updatedEntity, err := h.service.UploadLogo(
		c.Request.Context(),
		&services.UploadLogoRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			OrganizationID: orgID,
			File:           file,
		},
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

// @Summary Get organization logo URL
// @ID getOrganizationLogoURL
// @Tags Organizations
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} services.GetLogoURLResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /organizations/{id}/logo [get]
func (h *Handler) getLogoURL(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	orgID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.GetLogoURL(c.Request.Context(), services.GetLogoURLRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		OrganizationID: orgID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Delete an organization logo
// @ID deleteOrganizationLogo
// @Tags Organizations
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} tenant.Organization
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /organizations/{id}/logo [delete]
func (h *Handler) deleteLogo(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	orgID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updatedEntity, err := h.service.DeleteLogo(
		c.Request.Context(),
		services.DeleteLogoRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			OrganizationID: orgID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

func (h *Handler) getMicrosoftSSOConfig(c *gin.Context) {
	orgID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.GetMicrosoftSSOConfig(c.Request.Context(), orgID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) upsertMicrosoftSSOConfig(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	orgID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(services.MicrosoftSSOConfig)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.OrganizationID = orgID.String()

	resp, err := h.service.UpsertMicrosoftSSOConfig(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  orgID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		req,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
