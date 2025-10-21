package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicetypeservice "github.com/emoss08/trenova/internal/core/services/servicetype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type ServiceTypeHandlerParams struct {
	fx.In

	Service      *servicetypeservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type ServiceTypeHandler struct {
	service      *servicetypeservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewServiceTypeHandler(p ServiceTypeHandlerParams) *ServiceTypeHandler {
	return &ServiceTypeHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *ServiceTypeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/service-types/")
	api.GET("", h.pm.RequirePermission(permission.ResourceServiceType, "read"), h.list)
	api.POST("", h.pm.RequirePermission(permission.ResourceServiceType, "create"), h.create)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceServiceType, "read"), h.get)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceServiceType, "update"), h.update)
}

func (h *ServiceTypeHandler) list(c *gin.Context) {
	pagination.Handle[*servicetype.ServiceType](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*servicetype.ServiceType], error) {
			return h.service.List(c.Request.Context(), &repositories.ListServiceTypeRequest{
				Filter: opts,
			})
		})
}

func (h *ServiceTypeHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetServiceTypeByIDOptions{
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

func (h *ServiceTypeHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(servicetype.ServiceType)
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

func (h *ServiceTypeHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(servicetype.ServiceType)
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
