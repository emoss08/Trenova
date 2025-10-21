package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/organization"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type OrganizationHandlerParams struct {
	fx.In

	Service      *organization.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type OrganizationHandler struct {
	service *organization.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func NewOrganizationHandler(p OrganizationHandlerParams) *OrganizationHandler {
	return &OrganizationHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PM,
	}
}

func (h *OrganizationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/organizations/")
	api.GET(
		"me/",
		h.getUserOrganizations,
	)
	api.GET(":id/", h.get)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceOrganization, "update"), h.update)
}

func (h *OrganizationHandler) getUserOrganizations(c *gin.Context) {
	pagination.Handle[*tenant.Organization](c, context.GetAuthContext(c)).
		WithErrorHandler(h.eh).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*tenant.Organization], error) {
			return h.service.GetUserOrganizations(c.Request.Context(), opts)
		})
}

func (h *OrganizationHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	orgID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	org, err := h.service.GetByID(c.Request.Context(), repositories.GetOrganizationByIDRequest{
		OrgID:        orgID,
		BuID:         authCtx.BusinessUnitID,
		IncludeState: c.Query("includeState") == "true",
		IncludeBu:    c.Query("includeBu") == "true",
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, org)
}

func (h *OrganizationHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

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

	org, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, org)
}
