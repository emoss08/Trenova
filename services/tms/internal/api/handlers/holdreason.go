package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	holdreasonservice "github.com/emoss08/trenova/internal/core/services/holdreason"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type HoldReasonHandlerParams struct {
	fx.In

	Service      *holdreasonservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type HoldReasonHandler struct {
	service      *holdreasonservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewHoldReasonHandler(p HoldReasonHandlerParams) *HoldReasonHandler {
	return &HoldReasonHandler{
		service:      p.Service,
		errorHandler: p.ErrorHandler,
		pm:           p.PM,
	}
}

func (h *HoldReasonHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/hold-reasons/")
	api.GET("", h.pm.RequirePermission(permission.ResourceHoldReason, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceHoldReason, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceHoldReason, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceHoldReason, "update"), h.update)
}

func (h *HoldReasonHandler) list(c *gin.Context) {
	pagination.Handle[*holdreason.HoldReason](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*holdreason.HoldReason], error) {
			return h.service.List(c.Request.Context(), &repositories.ListHoldReasonRequest{
				Filter: opts,
			})
		})
}

func (h *HoldReasonHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	holdReason, err := h.service.Get(c.Request.Context(), repositories.GetHoldReasonByIDRequest{
		ID:     id,
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, holdReason)
}

func (h *HoldReasonHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	holdReason := new(holdreason.HoldReason)
	if err := c.ShouldBindJSON(holdReason); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, holdReason)
	holdReason, err := h.service.Create(c.Request.Context(), holdReason, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, holdReason)
}

func (h *HoldReasonHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	holdReason := new(holdreason.HoldReason)
	if err = c.ShouldBindJSON(holdReason); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	holdReason.ID = id
	context.AddContextToRequest(authCtx, holdReason)
	holdReason, err = h.service.Update(c.Request.Context(), holdReason, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, holdReason)
}
