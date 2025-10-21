package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	locationservice "github.com/emoss08/trenova/internal/core/services/location"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type LocationHandlerParams struct {
	fx.In

	Service      *locationservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type LocationHandler struct {
	service      *locationservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewLocationHandler(p LocationHandlerParams) *LocationHandler {
	return &LocationHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *LocationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/locations/")
	api.GET("", h.pm.RequirePermission(permission.ResourceLocation, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceLocation, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceLocation, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceLocation, "update"), h.update)
}

func (h *LocationHandler) list(c *gin.Context) {
	pagination.Handle[*location.Location](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*location.Location], error) {
			return h.service.List(c.Request.Context(), &repositories.ListLocationRequest{
				Filter:          opts,
				IncludeCategory: helpers.QueryBool(c, "includeCategory"),
				IncludeState:    helpers.QueryBool(c, "includeState"),
			})
		})
}

func (h *LocationHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetLocationByIDRequest{
			ID:              id,
			OrgID:           authCtx.OrganizationID,
			BuID:            authCtx.BusinessUnitID,
			UserID:          authCtx.UserID,
			IncludeCategory: helpers.QueryBool(c, "includeCategory"),
			IncludeState:    helpers.QueryBool(c, "includeState"),
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *LocationHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(location.Location)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *LocationHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(location.Location)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)

	entity, err = h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
