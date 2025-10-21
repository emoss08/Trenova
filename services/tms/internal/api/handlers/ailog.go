package handlers

import (
	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	ailogservice "github.com/emoss08/trenova/internal/core/services/ailog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type AILogHandlerParams struct {
	fx.In

	Service      *ailogservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type AILogHandler struct {
	service      *ailogservice.Service
	errorHandler *helpers.ErrorHandler
}

func NewAILogHandler(p AILogHandlerParams) *AILogHandler {
	return &AILogHandler{
		service:      p.Service,
		errorHandler: p.ErrorHandler,
	}
}

func (h *AILogHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/ai-logs/")
	api.GET("", h.list)
}

func (h *AILogHandler) list(c *gin.Context) {
	pagination.Handle[*ailog.AILog](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*ailog.AILog], error) {
			return h.service.List(c.Request.Context(), &repositories.ListAILogRequest{
				Filter:      opts,
				IncludeUser: helpers.QueryBool(c, "includeUser"),
			})
		})
}
