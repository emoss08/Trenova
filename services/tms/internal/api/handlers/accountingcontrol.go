package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrol"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type AccountingControlHandlerParams struct {
	fx.In

	Service      *accountingcontrol.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type AccountingControlHandler struct {
	service *accountingcontrol.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func NewAccountingControlHandler(p AccountingControlHandlerParams) *AccountingControlHandler {
	return &AccountingControlHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PM,
	}
}

func (h *AccountingControlHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting-controls/")
	api.GET("", h.pm.RequirePermission(permission.ResourceAccountingControl, "read"), h.get)
	api.PUT("", h.pm.RequirePermission(permission.ResourceAccountingControl, "update"), h.update)
}

func (h *AccountingControlHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req repositories.GetAccountingControlRequest
	context.AddContextToRequest(authCtx, &req)

	entity, err := h.service.GetByID(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *AccountingControlHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(accounting.AccountingControl)
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
