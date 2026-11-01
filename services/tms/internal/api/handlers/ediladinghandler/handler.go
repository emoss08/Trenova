package ediladinghandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Projects             *edilading.Projects
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	projects *edilading.Projects
	eh       *helpers.ErrorHandler
	pm       *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		projects: p.Projects,
		eh:       p.ErrorHandler,
		pm:       p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	h.registerLadingRoutes(rg.Group("/edi/lading"))
}
