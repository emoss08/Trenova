package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	accounttypeservice "github.com/emoss08/trenova/internal/core/services/accounttype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type AccountTypeHandlerParams struct {
	fx.In

	Service      *accounttypeservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type AccountTypeHandler struct {
	service      *accounttypeservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewAccountTypeHandler(p AccountTypeHandlerParams) *AccountTypeHandler {
	return &AccountTypeHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *AccountTypeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/account-types/")
	api.GET("", h.pm.RequirePermission(permission.ResourceAccountType, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceAccountType, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceAccountType, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceAccountType, "update"), h.update)
}

func (h *AccountTypeHandler) list(c *gin.Context) {
	pagination.Handle[*accounting.AccountType](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*accounting.AccountType], error) {
			return h.service.List(c.Request.Context(), &repositories.ListAccountTypeRequest{
				Filter: opts,
			})
		})
}

func (h *AccountTypeHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetAccountTypeByIDRequest{
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

func (h *AccountTypeHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(accounting.AccountType)
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

func (h *AccountTypeHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(accounting.AccountType)
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
