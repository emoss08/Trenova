package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchcontrol"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type DispatchControlHandlerParams struct {
	fx.In

	Service      *dispatchcontrol.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type DispatchControlHandler struct {
	service *dispatchcontrol.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func NewDispatchControlHandler(p DispatchControlHandlerParams) *DispatchControlHandler {
	return &DispatchControlHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PM,
	}
}

func (h *DispatchControlHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/dispatch-controls/")
	api.GET("", h.pm.RequirePermission(permission.ResourceDispatchControl, "read"), h.get)
	api.PUT("", h.pm.RequirePermission(permission.ResourceDispatchControl, "update"), h.update)
}

func (h *DispatchControlHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req repositories.GetDispatchControlRequest
	context.AddContextToRequest(authCtx, &req)

	entity, err := h.service.GetByID(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *DispatchControlHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(tenant.DispatchControl)
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
