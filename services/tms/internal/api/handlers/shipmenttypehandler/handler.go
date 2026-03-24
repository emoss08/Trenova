package shipmenttypehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmenttypeservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *shipmenttypeservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *shipmenttypeservice.Service
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
	api := rg.Group("/shipment-types")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceShipmentType.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:shipmentTypeID",
		h.pm.RequirePermission(permission.ResourceShipmentType.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceShipmentType.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:shipmentTypeID/",
		h.pm.RequirePermission(permission.ResourceShipmentType.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:shipmentTypeID/",
		h.pm.RequirePermission(permission.ResourceShipmentType.String(), permission.OpUpdate),
		h.patch,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(permission.ResourceShipmentType.String(), permission.OpUpdate),
		h.bulkUpdateStatus,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:shipmentTypeID/", h.getOption)
}

// @Summary List shipment types
// @ID listShipmentTypes
// @Tags Shipment Types
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]shipmenttype.ShipmentType]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-types/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*shipmenttype.ShipmentType], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListShipmentTypesRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Bulk update shipment type statuses
// @ID bulkUpdateShipmentTypeStatus
// @Tags Shipment Types
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateShipmentTypeStatusRequest true "Bulk status update request"
// @Success 200 {array} shipmenttype.ShipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-types/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.IsAPIKey() {
		h.eh.HandleError(
			c,
			errortypes.NewAuthorizationError("API keys cannot bulk update shipment types"),
		)
		return
	}

	req := new(repositories.BulkUpdateShipmentTypeStatusRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	results, err := h.service.BulkUpdateStatus(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// @Summary Get a shipment type option
// @ID getShipmentTypeOption
// @Tags Shipment Types
// @Produce json
// @Param shipmentTypeID path string true "Shipment type ID"
// @Success 200 {object} shipmenttype.ShipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-types/select-options/{shipmentTypeID}/ [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentTypeID, err := pulid.MustParse(c.Param("shipmentTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetShipmentTypeByIDRequest{
		ID: shipmentTypeID,
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

// @Summary List shipment type options
// @ID listShipmentTypeOptions
// @Tags Shipment Types
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]shipmenttype.ShipmentType]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-types/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*shipmenttype.ShipmentType], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.ShipmentTypeSelectOptionsRequest{
					SelectQueryRequest: req,
					Classes:            helpers.QuerySlice(c, "classes", []string{}),
				},
			)
		},
	)
}

// @Summary Get a shipment type
// @ID getShipmentType
// @Tags Shipment Types
// @Produce json
// @Param shipmentTypeID path string true "Shipment type ID"
// @Success 200 {object} shipmenttype.ShipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-types/{shipmentTypeID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentTypeID, err := pulid.MustParse(c.Param("shipmentTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetShipmentTypeByIDRequest{
			ID: shipmentTypeID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a shipment type
// @ID createShipmentType
// @Tags Shipment Types
// @Accept json
// @Produce json
// @Param request body shipmenttype.ShipmentType true "Shipment type payload"
// @Success 201 {object} shipmenttype.ShipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-types/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(shipmenttype.ShipmentType)
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

// @Summary Patch a shipment type
// @ID patchShipmentType
// @Tags Shipment Types
// @Accept json
// @Produce json
// @Param shipmentTypeID path string true "Shipment type ID"
// @Param request body shipmenttype.ShipmentType true "Partial shipment type payload"
// @Success 200 {object} shipmenttype.ShipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-types/{shipmentTypeID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentTypeID, err := pulid.MustParse(c.Param("shipmentTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetShipmentTypeByIDRequest{
			ID: shipmentTypeID,
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

// @Summary Update a shipment type
// @ID updateShipmentType
// @Tags Shipment Types
// @Accept json
// @Produce json
// @Param shipmentTypeID path string true "Shipment type ID"
// @Param request body shipmenttype.ShipmentType true "Shipment type payload"
// @Success 200 {object} shipmenttype.ShipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-types/{shipmentTypeID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentTypeID, err := pulid.MustParse(c.Param("shipmentTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(shipmenttype.ShipmentType)
	entity.ID = shipmentTypeID
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
