package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/billingcontrol"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type BillingControlHandlerParams struct {
	fx.In

	Service      *billingcontrol.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type BillingControlHandler struct {
	service *billingcontrol.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func NewBillingControlHandler(p BillingControlHandlerParams) *BillingControlHandler {
	return &BillingControlHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PM,
	}
}

func (h *BillingControlHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/billing-controls/")
	api.GET("", h.pm.RequirePermission(permission.ResourceBillingControl, "read"), h.get)
	api.PUT(
		"",
		h.pm.RequirePermission(permission.ResourceBillingControl, "update"),
		h.update,
	)
}

func (h *BillingControlHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req repositories.GetBillingControlRequest
	context.AddContextToRequest(authCtx, &req)

	entity, err := h.service.GetByID(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *BillingControlHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(tenant.BillingControl)
	entity.BusinessUnitID = authCtx.BusinessUnitID
	entity.OrganizationID = authCtx.OrganizationID

	if err := c.ShouldBindJSON(entity); err != nil {
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
