package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	accessorialchargeservice "github.com/emoss08/trenova/internal/core/services/accessorialcharge"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type AccessorialChargeHandlerParams struct {
	fx.In

	Service      *accessorialchargeservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type AccessorialChargeHandler struct {
	service      *accessorialchargeservice.Service
	errorHandler *helpers.ErrorHandler
	pm           *middleware.PermissionMiddleware
}

func NewAccessorialChargeHandler(p AccessorialChargeHandlerParams) *AccessorialChargeHandler {
	return &AccessorialChargeHandler{
		service:      p.Service,
		errorHandler: p.ErrorHandler,
		pm:           p.PM,
	}
}

func (h *AccessorialChargeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accessorial-charges/")
	api.GET("", h.pm.RequirePermission(permission.ResourceAccessorialCharge, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceAccessorialCharge, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceAccessorialCharge, "create"), h.create)
	api.PUT(
		":id/",
		h.pm.RequirePermission(permission.ResourceAccessorialCharge, "update"),
		h.update,
	)
}

func (h *AccessorialChargeHandler) list(c *gin.Context) {
	pagination.Handle[*accessorialcharge.AccessorialCharge](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
			return h.service.List(c.Request.Context(), &repositories.ListAccessorialChargeRequest{
				Filter: opts,
			})
		})
}

func (h *AccessorialChargeHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetAccessorialChargeByIDRequest{
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

func (h *AccessorialChargeHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(accessorialcharge.AccessorialCharge)
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

func (h *AccessorialChargeHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(accessorialcharge.AccessorialCharge)
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
