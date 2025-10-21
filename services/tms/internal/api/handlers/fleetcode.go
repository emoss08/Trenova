package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	fleetcodeservice "github.com/emoss08/trenova/internal/core/services/fleetcode"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type FleetCodeHandlerParams struct {
	fx.In

	Service      *fleetcodeservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type FleetCodeHandler struct {
	service      *fleetcodeservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewFleetCodeHandler(p FleetCodeHandlerParams) *FleetCodeHandler {
	return &FleetCodeHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *FleetCodeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/fleet-codes/")
	api.GET("", h.pm.RequirePermission(permission.ResourceFleetCode, "read"), h.list)
	api.POST("", h.pm.RequirePermission(permission.ResourceFleetCode, "create"), h.create)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceFleetCode, "read"), h.get)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceFleetCode, "update"), h.update)
}

func (h *FleetCodeHandler) list(c *gin.Context) {
	pagination.Handle[*fleetcode.FleetCode](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*fleetcode.FleetCode], error) {
			return h.service.List(c.Request.Context(), &repositories.ListFleetCodeRequest{
				Filter:                opts,
				IncludeManagerDetails: helpers.QueryBool(c, "includeManagerDetails"),
			})
		})
}

func (h *FleetCodeHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetFleetCodeByIDRequest{
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

func (h *FleetCodeHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(fleetcode.FleetCode)
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

func (h *FleetCodeHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(fleetcode.FleetCode)
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
