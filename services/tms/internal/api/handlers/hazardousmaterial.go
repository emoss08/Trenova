package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	hazardousmaterialservice "github.com/emoss08/trenova/internal/core/services/hazardousmaterial"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type HazardousMaterialHandlerParams struct {
	fx.In

	Service      *hazardousmaterialservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type HazardousMaterialHandler struct {
	service *hazardousmaterialservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func NewHazardousMaterialHandler(p HazardousMaterialHandlerParams) *HazardousMaterialHandler {
	return &HazardousMaterialHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PM,
	}
}

func (h *HazardousMaterialHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/hazardous-materials/")
	api.GET(
		"",
		h.pm.RequirePermission(permission.ResourceHazardousMaterial, "read"),
		h.list,
	)
	api.GET(
		":id/",
		h.pm.RequirePermission(permission.ResourceHazardousMaterial, "read"),
		h.get,
	)
	api.POST(
		"",
		h.pm.RequirePermission(permission.ResourceHazardousMaterial, "create"),
		h.create,
	)
	api.PUT(
		":id/",
		h.pm.RequirePermission(permission.ResourceHazardousMaterial, "update"),
		h.update,
	)
}

func (h *HazardousMaterialHandler) list(c *gin.Context) {
	pagination.Handle[*hazardousmaterial.HazardousMaterial](
		c,
		context.GetAuthContext(c),
	).
		WithErrorHandler(h.eh).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
			return h.service.List(c.Request.Context(), &repositories.ListHazardousMaterialRequest{
				Filter: opts,
			})
		})
}

func (h *HazardousMaterialHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	hazardousMaterial, err := h.service.Get(
		c.Request.Context(),
		repositories.GetHazardousMaterialByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, hazardousMaterial)
}

func (h *HazardousMaterialHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	hazardousMaterial := new(hazardousmaterial.HazardousMaterial)
	if err := c.ShouldBindJSON(hazardousMaterial); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, hazardousMaterial)
	hazardousMaterial, err := h.service.Create(
		c.Request.Context(),
		hazardousMaterial,
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, hazardousMaterial)
}

func (h *HazardousMaterialHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	hazardousMaterial := new(hazardousmaterial.HazardousMaterial)
	if err = c.ShouldBindJSON(hazardousMaterial); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	hazardousMaterial.ID = id
	context.AddContextToRequest(authCtx, hazardousMaterial)
	hazardousMaterial, err = h.service.Update(
		c.Request.Context(),
		hazardousMaterial,
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, hazardousMaterial)
}
