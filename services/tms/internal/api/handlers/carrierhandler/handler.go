package carrierhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/carrierservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *carrierservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *carrierservice.Service
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
	api := rg.Group("/carriers")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceCarrier.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:carrierID",
		h.pm.RequirePermission(permission.ResourceCarrier.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceCarrier.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:carrierID/",
		h.pm.RequirePermission(permission.ResourceCarrier.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:carrierID/",
		h.pm.RequirePermission(permission.ResourceCarrier.String(), permission.OpUpdate),
		h.patch,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(permission.ResourceCarrier.String(), permission.OpUpdate),
		h.bulkUpdateStatus,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:carrierID/", h.getOption)
}

// @Summary List carriers
// @ID listCarriers
// @Tags Carriers
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param fieldFilters query string false "JSON array of field filters"
// @Param filterGroups query string false "JSON array of grouped filters"
// @Param sort query string false "JSON array of sort fields"
// @Param includeState query bool false "Include state relationships"
// @Param includeContacts query bool false "Include carrier contacts"
// @Param includeInsurancePolicies query bool false "Include carrier insurance policies"
// @Success 200 {object} pagination.Response[[]carrier.Carrier]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /carriers/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*carrier.Carrier], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListCarrierRequest{
					Filter: req,
					CarrierFilterOptions: repositories.CarrierFilterOptions{
						IncludeState:             helpers.QueryBool(c, "includeState"),
						IncludeContacts:          helpers.QueryBool(c, "includeContacts"),
						IncludeInsurancePolicies: helpers.QueryBool(c, "includeInsurancePolicies"),
					},
				},
			)
		},
	)
}

// @Summary Bulk update carrier statuses
// @ID bulkUpdateCarrierStatus
// @Tags Carriers
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateCarrierStatusRequest true "Bulk status update request"
// @Success 200 {array} carrier.Carrier
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /carriers/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.IsAPIKey() {
		h.eh.HandleError(
			c,
			errortypes.NewAuthorizationError("API keys cannot bulk update carrier status"),
		)
		return
	}

	req := new(repositories.BulkUpdateCarrierStatusRequest)
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

// @Summary Get a carrier option
// @ID getCarrierOption
// @Tags Carriers
// @Produce json
// @Param carrierID path string true "Carrier ID"
// @Success 200 {object} carrier.Carrier
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /carriers/select-options/{carrierID}/ [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	carrierID, err := pulid.MustParse(c.Param("carrierID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetCarrierByIDRequest{
		ID: carrierID,
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

// @Summary List carrier options
// @ID listCarrierOptions
// @Tags Carriers
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]carrier.Carrier]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /carriers/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*carrier.Carrier], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.CarrierSelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

// @Summary Get a carrier
// @ID getCarrier
// @Tags Carriers
// @Produce json
// @Param carrierID path string true "Carrier ID"
// @Param includeState query bool false "Include state relationships"
// @Param includeContacts query bool false "Include carrier contacts"
// @Param includeInsurancePolicies query bool false "Include carrier insurance policies"
// @Success 200 {object} carrier.Carrier
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /carriers/{carrierID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	carrierID, err := pulid.MustParse(c.Param("carrierID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetCarrierByIDRequest{
			ID: carrierID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			CarrierFilterOptions: repositories.CarrierFilterOptions{
				IncludeState:             helpers.QueryBool(c, "includeState"),
				IncludeContacts:          helpers.QueryBool(c, "includeContacts"),
				IncludeInsurancePolicies: helpers.QueryBool(c, "includeInsurancePolicies"),
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a carrier
// @ID createCarrier
// @Tags Carriers
// @Accept json
// @Produce json
// @Param request body carrier.Carrier true "Carrier payload"
// @Success 201 {object} carrier.Carrier
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /carriers/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(carrier.Carrier)
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

// @Summary Patch a carrier
// @ID patchCarrier
// @Tags Carriers
// @Accept json
// @Produce json
// @Param carrierID path string true "Carrier ID"
// @Param request body carrier.Carrier true "Partial carrier payload"
// @Success 200 {object} carrier.Carrier
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /carriers/{carrierID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	carrierID, err := pulid.MustParse(c.Param("carrierID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetCarrierByIDRequest{
			ID: carrierID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			CarrierFilterOptions: repositories.CarrierFilterOptions{
				IncludeContacts:          true,
				IncludeInsurancePolicies: true,
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

// @Summary Update a carrier
// @ID updateCarrier
// @Tags Carriers
// @Accept json
// @Produce json
// @Param carrierID path string true "Carrier ID"
// @Param request body carrier.Carrier true "Carrier payload"
// @Success 200 {object} carrier.Carrier
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /carriers/{carrierID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	carrierID, err := pulid.MustParse(c.Param("carrierID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(carrier.Carrier)
	entity.ID = carrierID
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
