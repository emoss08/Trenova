package handlers

import (
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	equipmenttypeservice "github.com/emoss08/trenova/internal/core/services/equipmenttype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type EquipmentTypeHandlerParams struct {
	fx.In

	Service      *equipmenttypeservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type EquipmentTypeHandler struct {
	service      *equipmenttypeservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewEquipmentTypeHandler(p EquipmentTypeHandlerParams) *EquipmentTypeHandler {
	return &EquipmentTypeHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *EquipmentTypeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/equipment-types/")
	api.GET("", h.list)
	api.POST("", h.create)
	api.GET(":id/", h.get)
	api.PUT(":id/", h.update)
}

func (h *EquipmentTypeHandler) list(c *gin.Context) {
	pagination.Handle[*equipmenttype.EquipmentType](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*equipmenttype.EquipmentType], error) {
			// ! We don't require classes to be present, so we can just pass an empty array if they're not present
			classes, _ := c.GetQuery("classes")

			return h.service.List(c.Request.Context(), &repositories.ListEquipmentTypeRequest{
				Filter:  opts,
				Classes: strings.Split(classes, ","),
			})
		})
}

func (h *EquipmentTypeHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetEquipmentTypeByIDRequest{
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

func (h *EquipmentTypeHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(equipmenttype.EquipmentType)
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

func (h *EquipmentTypeHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(equipmenttype.EquipmentType)
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
