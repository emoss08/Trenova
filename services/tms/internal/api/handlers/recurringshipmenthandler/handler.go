package recurringshipmenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/recurringshipmentservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *recurringshipmentservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *recurringshipmentservice.Service
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
	api := rg.Group("/recurring-shipments")
	api.GET(
		"/",
		h.pm.RequirePermission(
			permission.ResourceRecurringShipment.String(),
			permission.OpRead,
		),
		h.list,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(
			permission.ResourceRecurringShipment.String(),
			permission.OpCreate,
		),
		h.create,
	)
	api.POST(
		"/match/",
		h.pm.RequirePermission(
			permission.ResourceRecurringShipment.String(),
			permission.OpRead,
		),
		h.match,
	)
	api.GET(
		"/:recurringShipmentID",
		h.pm.RequirePermission(
			permission.ResourceRecurringShipment.String(),
			permission.OpRead,
		),
		h.get,
	)
	api.GET(
		"/:recurringShipmentID/",
		h.pm.RequirePermission(
			permission.ResourceRecurringShipment.String(),
			permission.OpRead,
		),
		h.get,
	)
	api.PUT(
		"/:recurringShipmentID/",
		h.pm.RequirePermission(
			permission.ResourceRecurringShipment.String(),
			permission.OpUpdate,
		),
		h.update,
	)
	api.PUT(
		"/:recurringShipmentID/status/",
		h.pm.RequirePermission(
			permission.ResourceRecurringShipment.String(),
			permission.OpUpdate,
		),
		h.updateStatus,
	)
	api.POST(
		"/:recurringShipmentID/generate/",
		h.pm.RequirePermission(
			permission.ResourceRecurringShipment.String(),
			permission.OpDuplicate,
		),
		h.generate,
	)
	api.GET(
		"/:recurringShipmentID/runs/",
		h.pm.RequirePermission(
			permission.ResourceRecurringShipment.String(),
			permission.OpRead,
		),
		h.listRuns,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:recurringShipmentID/", h.getOption)
}

// @Summary List recurring shipments
// @ID listRecurringShipments
// @Tags Recurring Shipments
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]recurringshipment.RecurringShipment]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /recurring-shipments/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListRecurringShipmentsRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get a recurring shipment
// @ID getRecurringShipment
// @Tags Recurring Shipments
// @Produce json
// @Param recurringShipmentID path string true "Recurring shipment ID"
// @Param expandDetails query bool false "Include the full source shipment"
// @Success 200 {object} recurringshipment.RecurringShipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /recurring-shipments/{recurringShipmentID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	recurringShipmentID, err := pulid.MustParse(c.Param("recurringShipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetRecurringShipmentByIDRequest{
			ID: recurringShipmentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			ExpandDetails: c.Query("expandDetails") == "true",
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a recurring shipment
// @ID createRecurringShipment
// @Tags Recurring Shipments
// @Accept json
// @Produce json
// @Param request body recurringshipment.RecurringShipment true "Recurring shipment payload"
// @Success 201 {object} recurringshipment.RecurringShipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /recurring-shipments/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(recurringshipment.RecurringShipment)
	authctx.AddContextToRequest(authCtx, entity)

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update a recurring shipment
// @ID updateRecurringShipment
// @Tags Recurring Shipments
// @Accept json
// @Produce json
// @Param recurringShipmentID path string true "Recurring shipment ID"
// @Param request body recurringshipment.RecurringShipment true "Recurring shipment payload"
// @Success 200 {object} recurringshipment.RecurringShipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /recurring-shipments/{recurringShipmentID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	recurringShipmentID, err := pulid.MustParse(c.Param("recurringShipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(recurringshipment.RecurringShipment)
	entity.ID = recurringShipmentID
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

type updateStatusRequest struct {
	Status  recurringshipment.Status `json:"status"`
	Version int64                    `json:"version"`
}

// @Summary Update a recurring shipment's status
// @ID updateRecurringShipmentStatus
// @Tags Recurring Shipments
// @Accept json
// @Produce json
// @Param recurringShipmentID path string true "Recurring shipment ID"
// @Param request body updateStatusRequest true "Status payload"
// @Success 200 {object} recurringshipment.RecurringShipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /recurring-shipments/{recurringShipmentID}/status/ [put]
func (h *Handler) updateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	recurringShipmentID, err := pulid.MustParse(c.Param("recurringShipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(updateStatusRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.UpdateStatus(
		c.Request.Context(),
		&repositories.UpdateRecurringShipmentStatusRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			RecurringShipmentID: recurringShipmentID,
			Status:              req.Status,
			Version:             req.Version,
		},
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Match recurring shipments for a lane
// @ID matchRecurringShipments
// @Tags Recurring Shipments
// @Accept json
// @Produce json
// @Param request body repositories.MatchRecurringShipmentsRequest true "Lane match payload"
// @Success 200 {object} repositories.MatchRecurringShipmentsResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /recurring-shipments/match/ [post]
func (h *Handler) match(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(repositories.MatchRecurringShipmentsRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	response, err := h.service.Match(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

type generateRequest struct {
	OccurrenceAt *int64 `json:"occurrenceAt"`
}

// @Summary Generate a shipment from a recurring series
// @ID generateRecurringShipment
// @Tags Recurring Shipments
// @Accept json
// @Produce json
// @Param recurringShipmentID path string true "Recurring shipment ID"
// @Param request body generateRequest false "Generation payload"
// @Success 201 {object} repositories.GenerateRecurringShipmentResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /recurring-shipments/{recurringShipmentID}/generate/ [post]
func (h *Handler) generate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	recurringShipmentID, err := pulid.MustParse(c.Param("recurringShipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(generateRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}

	result, err := h.service.Generate(
		c.Request.Context(),
		&repositories.GenerateRecurringShipmentRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			RecurringShipmentID: recurringShipmentID,
			OccurrenceAt:        req.OccurrenceAt,
			Trigger:             recurringshipment.RunTriggerManual,
			RequestedBy:         authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// @Summary List generation runs for a recurring shipment
// @ID listRecurringShipmentRuns
// @Tags Recurring Shipments
// @Produce json
// @Param recurringShipmentID path string true "Recurring shipment ID"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]recurringshipment.RecurringShipmentRun]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /recurring-shipments/{recurringShipmentID}/runs/ [get]
func (h *Handler) listRuns(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	recurringShipmentID, err := pulid.MustParse(c.Param("recurringShipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*recurringshipment.RecurringShipmentRun], error) {
			return h.service.ListRuns(
				c.Request.Context(),
				&repositories.ListRecurringShipmentRunsRequest{
					TenantInfo: pagination.TenantInfo{
						OrgID:  authCtx.OrganizationID,
						BuID:   authCtx.BusinessUnitID,
						UserID: authCtx.UserID,
					},
					RecurringShipmentID: recurringShipmentID,
					Filter:              req,
				},
			)
		},
	)
}

// @Summary List recurring shipment options
// @ID listRecurringShipmentOptions
// @Tags Recurring Shipments
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]recurringshipment.RecurringShipment]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /recurring-shipments/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.RecurringShipmentSelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

// @Summary Get a recurring shipment option
// @ID getRecurringShipmentOption
// @Tags Recurring Shipments
// @Produce json
// @Param recurringShipmentID path string true "Recurring shipment ID"
// @Success 200 {object} recurringshipment.RecurringShipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /recurring-shipments/select-options/{recurringShipmentID}/ [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	recurringShipmentID, err := pulid.MustParse(c.Param("recurringShipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetRecurringShipmentByIDRequest{
			ID: recurringShipmentID,
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
