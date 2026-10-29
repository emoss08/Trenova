package ediladinghandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Outbound             *edioutbound.Service
	Projects             *edilading.Projects
	LadingRuntime        *edilading.Runtime
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	outbound      *edioutbound.Service
	projects      *edilading.Projects
	ladingRuntime *edilading.Runtime
	eh            *helpers.ErrorHandler
	pm            *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		outbound:      p.Outbound,
		projects:      p.Projects,
		ladingRuntime: p.LadingRuntime,
		eh:            p.ErrorHandler,
		pm:            p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/edi")

	h.registerLadingRoutes(api.Group("/lading"))
	h.registerFlowRoutes(api.Group("/flows"))
}
