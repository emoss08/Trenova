package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	equipmentmanufactureservice "github.com/emoss08/trenova/internal/core/services/equipmentmanufacturer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type EquipmentManufacturerHandlerParams struct {
	fx.In

	Service      *equipmentmanufactureservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type EquipmentManufacturerHandler struct {
	service      *equipmentmanufactureservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewEquipmentManufacturerHandler(
	p EquipmentManufacturerHandlerParams,
) *EquipmentManufacturerHandler {
	return &EquipmentManufacturerHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *EquipmentManufacturerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/equipment-manufacturers/")
	api.GET("", h.pm.RequirePermission(permission.ResourceEquipmentManufacturer, "read"), h.list)
	api.POST(
		"",
		h.pm.RequirePermission(permission.ResourceEquipmentManufacturer, "create"),
		h.create,
	)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceEquipmentManufacturer, "read"), h.get)
	api.PUT(
		":id/",
		h.pm.RequirePermission(permission.ResourceEquipmentManufacturer, "update"),
		h.update,
	)
}

func (h *EquipmentManufacturerHandler) list(c *gin.Context) {
	pagination.Handle[*equipmentmanufacturer.EquipmentManufacturer](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListEquipmentManufacturerRequest{
					Filter: opts,
				},
			)
		})
}

func (h *EquipmentManufacturerHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetEquipmentManufacturerByIDRequest{
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

func (h *EquipmentManufacturerHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(equipmentmanufacturer.EquipmentManufacturer)
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

func (h *EquipmentManufacturerHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(equipmentmanufacturer.EquipmentManufacturer)
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
