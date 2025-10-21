package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	commodityservice "github.com/emoss08/trenova/internal/core/services/commodity"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type CommodityHandlerParams struct {
	fx.In

	Service      *commodityservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type CommodityHandler struct {
	service      *commodityservice.Service
	errorHandler *helpers.ErrorHandler
	pm           *middleware.PermissionMiddleware
}

func NewCommodityHandler(p CommodityHandlerParams) *CommodityHandler {
	return &CommodityHandler{
		service:      p.Service,
		errorHandler: p.ErrorHandler,
		pm:           p.PM,
	}
}

func (h *CommodityHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/commodities/")
	api.GET("", h.pm.RequirePermission(permission.ResourceCommodity, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceCommodity, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceCommodity, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceCommodity, "update"), h.update)
}

func (h *CommodityHandler) list(c *gin.Context) {
	pagination.Handle[*commodity.Commodity](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*commodity.Commodity], error) {
			return h.service.List(c.Request.Context(), &repositories.ListCommodityRequest{
				Filter: opts,
			})
		})
}

func (h *CommodityHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetCommodityByIDRequest{
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

func (h *CommodityHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(commodity.Commodity)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *CommodityHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(commodity.Commodity)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)

	entity, err = h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
