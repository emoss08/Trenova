package orderhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/orderservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *orderservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *orderservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/orders")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceOrder.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:orderID",
		h.pm.RequirePermission(permission.ResourceOrder.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceOrder.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:orderID/",
		h.pm.RequirePermission(permission.ResourceOrder.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:orderID/",
		h.pm.RequirePermission(permission.ResourceOrder.String(), permission.OpUpdate),
		h.patch,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:orderID/", h.getOption)
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*order.Order], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListOrdersRequest{
					Filter: req,
				},
			)
		},
	)
}

func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	orderID, err := pulid.MustParse(c.Param("orderID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetOrderByIDRequest{
		ID: orderID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*order.Order], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.OrderSelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	orderID, err := pulid.MustParse(c.Param("orderID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetOrderByIDRequest{
			ID: orderID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			IncludeShipment: true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(order.Order)
	authctx.AddContextToRequest(authCtx, entity)

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	created, err := h.service.Create(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	orderID, err := pulid.MustParse(c.Param("orderID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetOrderByIDRequest{
			ID: orderID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = c.ShouldBindJSON(existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updatedEntity, err := h.service.Update(c.Request.Context(), existing, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	orderID, err := pulid.MustParse(c.Param("orderID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(order.Order)
	entity.ID = orderID
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}
