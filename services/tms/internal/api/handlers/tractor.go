package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	tractorservice "github.com/emoss08/trenova/internal/core/services/tractor"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type TractorHandlerParams struct {
	fx.In

	Service      *tractorservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type TractorHandler struct {
	service      *tractorservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewTractorHandler(p TractorHandlerParams) *TractorHandler {
	return &TractorHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *TractorHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/tractors/")
	api.GET("", h.pm.RequirePermission(permission.ResourceTractor, "read"), h.list)
	api.POST("", h.pm.RequirePermission(permission.ResourceTractor, "create"), h.create)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceTractor, "read"), h.get)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceTractor, "update"), h.update)
	api.GET(
		":id/assignment/",
		h.pm.RequirePermission(permission.ResourceTractor, "assign"),
		h.assignment,
	)
}

func (h *TractorHandler) list(c *gin.Context) {
	pagination.Handle[*tractor.Tractor](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*tractor.Tractor], error) {
			return h.service.List(c.Request.Context(), &repositories.ListTractorRequest{
				Filter: opts,
				FilterOptions: repositories.TractorFilterOptions{
					IncludeWorkerDetails:    helpers.QueryBool(c, "includeWorkerDetails"),
					IncludeEquipmentDetails: helpers.QueryBool(c, "includeEquipmentDetails"),
					IncludeFleetDetails:     helpers.QueryBool(c, "includeFleetDetails"),
					Status:                  c.Query("status"),
				},
			})
		})
}

func (h *TractorHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetTractorByIDRequest{
			TractorID: id,
			OrgID:     authCtx.OrganizationID,
			BuID:      authCtx.BusinessUnitID,
			UserID:    authCtx.UserID,
			FilterOptions: repositories.TractorFilterOptions{
				IncludeWorkerDetails:    helpers.QueryBool(c, "includeWorkerDetails"),
				IncludeEquipmentDetails: helpers.QueryBool(c, "includeEquipmentDetails"),
				IncludeFleetDetails:     helpers.QueryBool(c, "includeFleetDetails"),
				Status:                  c.Query("status"),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *TractorHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(tractor.Tractor)
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

func (h *TractorHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(tractor.Tractor)
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

func (h *TractorHandler) assignment(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	tractorID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	assignment, err := h.service.Assignment(
		c.Request.Context(),
		repositories.TractorAssignmentRequest{
			TractorID: tractorID,
			OrgID:     authCtx.OrganizationID,
			BuID:      authCtx.BusinessUnitID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, assignment)
}
