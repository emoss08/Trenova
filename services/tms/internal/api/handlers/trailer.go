package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	trailerservice "github.com/emoss08/trenova/internal/core/services/trailer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type TrailerHandlerParams struct {
	fx.In

	Service      *trailerservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type TrailerHandler struct {
	service      *trailerservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewTrailerHandler(p TrailerHandlerParams) *TrailerHandler {
	return &TrailerHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *TrailerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/trailers/")
	api.GET("", h.pm.RequirePermission(permission.ResourceTrailer, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceTrailer, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceTrailer, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceTrailer, "update"), h.update)
}

func (h *TrailerHandler) list(c *gin.Context) {
	pagination.Handle[*trailer.Trailer](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*trailer.Trailer], error) {
			return h.service.List(c.Request.Context(), &repositories.ListTrailerRequest{
				Filter: opts,
				FilterOptions: repositories.TrailerFilterOptions{
					IncludeEquipmentDetails: helpers.QueryBool(c, "includeEquipmentDetails"),
					IncludeFleetDetails:     helpers.QueryBool(c, "includeFleetDetails"),
					Status:                  c.Query("status"),
				},
			})
		})
}

func (h *TrailerHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetTrailerByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
			FilterOptions: repositories.TrailerFilterOptions{
				IncludeEquipmentDetails: helpers.QueryBool(c, "includeEquipmentDetails"),
				IncludeFleetDetails:     helpers.QueryBool(c, "includeFleetDetails"),
				Status:                  c.Query("status"),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *TrailerHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(trailer.Trailer)
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

func (h *TrailerHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(trailer.Trailer)
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
