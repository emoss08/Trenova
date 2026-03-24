package tractorhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *tractorservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *tractorservice.Service
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
	api := rg.Group("/tractors")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceTractor.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:tractorID/",
		h.pm.RequirePermission(permission.ResourceTractor.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceTractor.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:tractorID/",
		h.pm.RequirePermission(permission.ResourceTractor.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:tractorID/",
		h.pm.RequirePermission(permission.ResourceTractor.String(), permission.OpUpdate),
		h.patch,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(permission.ResourceTractor.String(), permission.OpUpdate),
		h.bulkUpdateStatus,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:tractorID", h.getOption)
}

// @Summary List tractors
// @ID listTractors
// @Tags Tractors
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param includeEquipmentDetails query bool false "Include equipment details"
// @Param includeFleetDetails query bool false "Include fleet details"
// @Param includeWorkerDetails query bool false "Include worker details"
// @Param status query string false "Filter by status"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]tractor.Tractor]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tractors/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*tractor.Tractor], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListTractorsRequest{
					Filter:                  req,
					IncludeEquipmentDetails: helpers.QueryBool(c, "includeEquipmentDetails", false),
					IncludeFleetDetails:     helpers.QueryBool(c, "includeFleetDetails", false),
					IncludeWorkerDetails:    helpers.QueryBool(c, "includeWorkerDetails", false),
					Status:                  helpers.QueryString(c, "status", ""),
				},
			)
		},
	)
}

// @Summary Get a tractor
// @ID getTractor
// @Tags Tractors
// @Produce json
// @Param tractorID path string true "Tractor ID"
// @Success 200 {object} tractor.Tractor
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tractors/{tractorID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("tractorID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetTractorByIDRequest{
			ID: id,
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

// @Summary Get a tractor option
// @ID getTractorOption
// @Tags Tractors
// @Produce json
// @Param tractorID path string true "Tractor ID"
// @Success 200 {object} tractor.Tractor
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tractors/select-options/{tractorID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	tractorID, err := pulid.MustParse(c.Param("tractorID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetTractorByIDRequest{
		ID: tractorID,
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

// @Summary List tractor options
// @ID listTractorOptions
// @Tags Tractors
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]tractor.Tractor]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tractors/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*tractor.Tractor], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.TractorSelectOptionsRequest{
					SelectOptionsRequest: req,
				},
			)
		},
	)
}

// @Summary Create a tractor
// @ID createTractor
// @Tags Tractors
// @Accept json
// @Produce json
// @Param request body tractor.Tractor true "Tractor payload"
// @Success 201 {object} tractor.Tractor
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tractors/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(tractor.Tractor)
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

// @Summary Update a tractor
// @ID updateTractor
// @Tags Tractors
// @Accept json
// @Produce json
// @Param tractorID path string true "Tractor ID"
// @Param request body tractor.Tractor true "Tractor payload"
// @Success 200 {object} tractor.Tractor
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tractors/{tractorID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	tractorID, err := pulid.MustParse(c.Param("tractorID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(tractor.Tractor)
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity.ID = tractorID

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Patch a tractor
// @ID patchTractor
// @Tags Tractors
// @Accept json
// @Produce json
// @Param tractorID path string true "Tractor ID"
// @Param request body tractor.Tractor true "Partial tractor payload"
// @Success 200 {object} tractor.Tractor
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tractors/{tractorID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	tractorID, err := pulid.MustParse(c.Param("tractorID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetTractorByIDRequest{
			ID: tractorID,
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

	if err = c.ShouldBindJSON(existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), existing, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Bulk update tractor statuses
// @ID bulkUpdateTractorStatus
// @Tags Tractors
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateTractorStatusRequest true "Bulk status update request"
// @Success 200 {array} tractor.Tractor
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tractors/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.IsAPIKey() {
		h.eh.HandleError(
			c,
			errortypes.NewAuthorizationError("API keys cannot bulk update tractors"),
		)
		return
	}

	req := new(repositories.BulkUpdateTractorStatusRequest)
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
