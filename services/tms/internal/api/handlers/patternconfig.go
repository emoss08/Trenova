package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/patternconfig"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type PatternConfigHandlerParams struct {
	fx.In

	Service      *patternconfig.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type PatternConfigHandler struct {
	service *patternconfig.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func NewPatternConfigHandler(p PatternConfigHandlerParams) *PatternConfigHandler {
	return &PatternConfigHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PM,
	}
}

func (h *PatternConfigHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/pattern-config/")
	api.GET("", h.pm.RequirePermission(permission.ResourcePatternConfig, "read"), h.get)
	api.PUT("", h.pm.RequirePermission(permission.ResourcePatternConfig, "update"), h.update)
}

func (h *PatternConfigHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req repositories.GetPatternConfigRequest
	context.AddContextToRequest(authCtx, &req)

	entity, err := h.service.GetByOrgID(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *PatternConfigHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(dedicatedlane.PatternConfig)
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
