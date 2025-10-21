package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dataretention"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type DataRetentionHandlerParams struct {
	fx.In

	Service      *dataretention.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type DataRetentionHandler struct {
	service *dataretention.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func NewDataRetentionHandler(p DataRetentionHandlerParams) *DataRetentionHandler {
	return &DataRetentionHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PM,
	}
}

func (h *DataRetentionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/data-retention/")
	api.GET("", h.pm.RequirePermission(permission.ResourceDataRetention, "read"), h.get)
	api.PUT("", h.pm.RequirePermission(permission.ResourceDataRetention, "update"), h.update)
}

func (h *DataRetentionHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req repositories.GetDataRetentionRequest
	context.AddContextToRequest(authCtx, &req)

	entity, err := h.service.Get(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *DataRetentionHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(tenant.DataRetention)

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
