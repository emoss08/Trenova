package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	customerservice "github.com/emoss08/trenova/internal/core/services/customer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type CustomerHandlerParams struct {
	fx.In

	Service      *customerservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type CustomerHandler struct {
	service      *customerservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewCustomerHandler(p CustomerHandlerParams) *CustomerHandler {
	return &CustomerHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *CustomerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/customers/")
	api.GET("", h.pm.RequirePermission(permission.ResourceCustomer, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceCustomer, "read"), h.get)
	api.GET(
		":id/document-requirements",
		h.pm.RequirePermission(permission.ResourceCustomer, "read"),
		h.getDocumentRequirements,
	)
	api.POST("", h.pm.RequirePermission(permission.ResourceCustomer, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceCustomer, "update"), h.update)
}

func (h *CustomerHandler) list(c *gin.Context) {
	pagination.Handle[*customer.Customer](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*customer.Customer], error) {
			return h.service.List(c.Request.Context(), &repositories.ListCustomerRequest{
				Filter: opts,
				CustomerFilterOptions: repositories.CustomerFilterOptions{
					IncludeState:          helpers.QueryBool(c, "includeState"),
					IncludeBillingProfile: helpers.QueryBool(c, "includeBillingProfile"),
					IncludeEmailProfile:   helpers.QueryBool(c, "includeEmailProfile"),
				},
			})
		})
}

func (h *CustomerHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetCustomerByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
			CustomerFilterOptions: repositories.CustomerFilterOptions{
				IncludeState:          helpers.QueryBool(c, "includeState"),
				IncludeBillingProfile: helpers.QueryBool(c, "includeBillingProfile"),
				IncludeEmailProfile:   helpers.QueryBool(c, "includeEmailProfile"),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *CustomerHandler) getDocumentRequirements(c *gin.Context) {
	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entities, err := h.service.GetDocumentRequirements(c.Request.Context(), id)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entities)
}

func (h *CustomerHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(customer.Customer)
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

func (h *CustomerHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(customer.Customer)
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
