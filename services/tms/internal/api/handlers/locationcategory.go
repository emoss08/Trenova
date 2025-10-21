package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	locationcategoryservice "github.com/emoss08/trenova/internal/core/services/locationcategory"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type LocationCategoryHandlerParams struct {
	fx.In

	Service      *locationcategoryservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type LocationCategoryHandler struct {
	service      *locationcategoryservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewLocationCategoryHandler(p LocationCategoryHandlerParams) *LocationCategoryHandler {
	return &LocationCategoryHandler{
		service:      p.Service,
		errorHandler: p.ErrorHandler,
		pm:           p.PM,
	}
}

func (h *LocationCategoryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/location-categories/")
	api.GET("", h.pm.RequirePermission(permission.ResourceLocationCategory, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceLocationCategory, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceLocationCategory, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceLocationCategory, "update"), h.update)
}

func (h *LocationCategoryHandler) list(c *gin.Context) {
	pagination.Handle[*location.LocationCategory](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*location.LocationCategory], error) {
			return h.service.List(c.Request.Context(), &repositories.ListLocationCategoryRequest{
				Filter: opts,
			})
		})
}

func (h *LocationCategoryHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetLocationCategoryByIDRequest{
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

func (h *LocationCategoryHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(location.LocationCategory)
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

func (h *LocationCategoryHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(location.LocationCategory)
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
