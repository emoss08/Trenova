package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	dedicatedlaneservice "github.com/emoss08/trenova/internal/core/services/dedicatedlane"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type DedicatedLaneHandlerParams struct {
	fx.In

	Service      *dedicatedlaneservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type DedicatedLaneHandler struct {
	service      *dedicatedlaneservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewDedicatedLaneHandler(p DedicatedLaneHandlerParams) *DedicatedLaneHandler {
	return &DedicatedLaneHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *DedicatedLaneHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/dedicated-lanes/")
	api.GET("", h.pm.RequirePermission(permission.ResourceDedicatedLane, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceDedicatedLane, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceDedicatedLane, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceDedicatedLane, "update"), h.update)
}

func (h *DedicatedLaneHandler) list(c *gin.Context) {
	pagination.Handle[*dedicatedlane.DedicatedLane](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*dedicatedlane.DedicatedLane], error) {
			return h.service.List(c.Request.Context(), &repositories.ListDedicatedLaneRequest{
				Filter: opts,
				FilterOptions: repositories.DedicatedLaneFilterOptions{
					ExpandDetails: helpers.QueryBool(c, "expandDetails"),
				},
			})
		})
}

func (h *DedicatedLaneHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetDedicatedLaneByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *DedicatedLaneHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(dedicatedlane.DedicatedLane)
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

func (h *DedicatedLaneHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(dedicatedlane.DedicatedLane)
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
