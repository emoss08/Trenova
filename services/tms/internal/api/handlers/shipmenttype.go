package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	shipmenttypeservice "github.com/emoss08/trenova/internal/core/services/shipmenttype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type ShipmentTypeHandlerParams struct {
	fx.In

	Service      *shipmenttypeservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type ShipmentTypeHandler struct {
	service      *shipmenttypeservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewShipmentTypeHandler(p ShipmentTypeHandlerParams) *ShipmentTypeHandler {
	return &ShipmentTypeHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *ShipmentTypeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/shipment-types/")
	api.GET("", h.pm.RequirePermission(permission.ResourceShipmentType, "read"), h.list)
	api.POST("", h.pm.RequirePermission(permission.ResourceShipmentType, "create"), h.create)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceShipmentType, "read"), h.get)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceShipmentType, "update"), h.update)
}

func (h *ShipmentTypeHandler) list(c *gin.Context) {
	pagination.Handle[*shipmenttype.ShipmentType](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*shipmenttype.ShipmentType], error) {
			return h.service.List(c.Request.Context(), &repositories.ListShipmentTypeRequest{
				Filter: opts,
			})
		})
}

func (h *ShipmentTypeHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetShipmentTypeByIDOptions{
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

func (h *ShipmentTypeHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(shipmenttype.ShipmentType)
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

func (h *ShipmentTypeHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(shipmenttype.ShipmentType)
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
