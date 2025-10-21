package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	distanceoverrideservice "github.com/emoss08/trenova/internal/core/services/distanceoverride"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type DistanceOverrideHandlerParams struct {
	fx.In

	Service      *distanceoverrideservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type DistanceOverrideHandler struct {
	service      *distanceoverrideservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewDistanceOverrideHandler(p DistanceOverrideHandlerParams) *DistanceOverrideHandler {
	return &DistanceOverrideHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *DistanceOverrideHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/distance-overrides/")
	api.GET("", h.pm.RequirePermission(permission.ResourceDistanceOverride, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceDistanceOverride, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceDistanceOverride, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceDistanceOverride, "update"), h.update)
	api.DELETE(
		":id/",
		h.pm.RequirePermission(permission.ResourceDistanceOverride, "delete"),
		h.delete,
	)
}

func (h *DistanceOverrideHandler) list(c *gin.Context) {
	pagination.Handle[*distanceoverride.Override](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*distanceoverride.Override], error) {
			return h.service.List(c.Request.Context(), &repositories.ListDistanceOverrideRequest{
				Filter:        opts,
				ExpandDetails: helpers.QueryBool(c, "expandDetails", false),
			})
		})
}

func (h *DistanceOverrideHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetDistanceOverrideRequest{
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

func (h *DistanceOverrideHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(distanceoverride.Override)
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

func (h *DistanceOverrideHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(distanceoverride.Override)
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

func (h *DistanceOverrideHandler) delete(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.Delete(c.Request.Context(), &repositories.DeleteDistanceOverrideRequest{
		ID:     id,
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
