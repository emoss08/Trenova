package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmentcontrol"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type ShipmentControlHandlerParams struct {
	fx.In

	Service      *shipmentcontrol.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type ShipmentControlHandler struct {
	service *shipmentcontrol.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func NewShipmentControlHandler(p ShipmentControlHandlerParams) *ShipmentControlHandler {
	return &ShipmentControlHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PM,
	}
}

func (h *ShipmentControlHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/shipment-controls/")
	api.GET("", h.pm.RequirePermission(permission.ResourceShipmentControl, "read"), h.get)
	api.PUT("", h.pm.RequirePermission(permission.ResourceShipmentControl, "update"), h.update)
}

func (h *ShipmentControlHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req repositories.GetShipmentControlRequest
	context.AddContextToRequest(authCtx, &req)

	entity, err := h.service.GetByID(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *ShipmentControlHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(tenant.ShipmentControl)
	entity.BusinessUnitID = authCtx.BusinessUnitID
	entity.OrganizationID = authCtx.OrganizationID

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
