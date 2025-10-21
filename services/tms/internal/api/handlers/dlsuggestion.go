package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	dlsuggestionservice "github.com/emoss08/trenova/internal/core/services/dlsuggestion"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type DedicatedLaneSuggestionHandlerParams struct {
	fx.In

	Service      *dlsuggestionservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type DedicatedLaneSuggestionHandler struct {
	service      *dlsuggestionservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewDedicatedLaneSuggestionHandler(
	p DedicatedLaneSuggestionHandlerParams,
) *DedicatedLaneSuggestionHandler {
	return &DedicatedLaneSuggestionHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *DedicatedLaneSuggestionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/dedicated-lane-suggestions/")
	api.GET("", h.pm.RequirePermission(permission.ResourceDedicatedLaneSuggestion, "read"), h.list)
	api.GET(
		":id/",
		h.pm.RequirePermission(permission.ResourceDedicatedLaneSuggestion, "read"),
		h.get,
	)
}

func (h *DedicatedLaneSuggestionHandler) list(c *gin.Context) {
	pagination.Handle[*dedicatedlane.Suggestion](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*dedicatedlane.Suggestion], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListDedicatedLaneSuggestionRequest{
					Filter: opts,
				},
			)
		})
}

func (h *DedicatedLaneSuggestionHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetDedicatedLaneSuggestionByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
