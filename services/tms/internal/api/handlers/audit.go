package handlers

import (
	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type AuditHandlerParams struct {
	fx.In

	Service      services.AuditService
	ErrorHandler *helpers.ErrorHandler
}

type AuditHandler struct {
	service services.AuditService
	eh      *helpers.ErrorHandler
}

func NewAuditHandler(p AuditHandlerParams) *AuditHandler {
	return &AuditHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *AuditHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/audit-entries/")
	api.GET("", h.list)
}

func (h *AuditHandler) list(c *gin.Context) {
	pagination.Handle[*audit.Entry](c, context.GetAuthContext(c)).
		WithErrorHandler(h.eh).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*audit.Entry], error) {
			return h.service.List(c.Request.Context(), opts)
		})
}
