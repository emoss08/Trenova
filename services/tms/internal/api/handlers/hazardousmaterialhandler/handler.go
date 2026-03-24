package hazardousmaterialhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/hazardousmaterialservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *hazardousmaterialservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *hazardousmaterialservice.Service
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
	api := rg.Group("/hazardous-materials")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceHazardousMaterial.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:hazardousMaterialID",
		h.pm.RequirePermission(permission.ResourceHazardousMaterial.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceHazardousMaterial.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:hazardousMaterialID/",
		h.pm.RequirePermission(permission.ResourceHazardousMaterial.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:hazardousMaterialID/",
		h.pm.RequirePermission(permission.ResourceHazardousMaterial.String(), permission.OpUpdate),
		h.patch,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(permission.ResourceHazardousMaterial.String(), permission.OpUpdate),
		h.bulkUpdateStatus,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:hazardousMaterialID/", h.getOption)
}

// @Summary List hazardous materials
// @ID listHazardousMaterials
// @Tags Hazardous Materials
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]hazardousmaterial.HazardousMaterial]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazardous-materials/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListHazardousMaterialsRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Bulk update hazardous material statuses
// @ID bulkUpdateHazardousMaterialStatus
// @Tags Hazardous Materials
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateHazardousMaterialStatusRequest true "Bulk status update request"
// @Success 200 {array} hazardousmaterial.HazardousMaterial
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazardous-materials/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.IsAPIKey() {
		h.eh.HandleError(
			c,
			errortypes.NewAuthorizationError("API keys cannot bulk update hazardous materials"),
		)
		return
	}

	req := new(repositories.BulkUpdateHazardousMaterialStatusRequest)
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

// @Summary Get a hazardous material option
// @ID getHazardousMaterialOption
// @Tags Hazardous Materials
// @Produce json
// @Param hazardousMaterialID path string true "Hazardous material ID"
// @Success 200 {object} hazardousmaterial.HazardousMaterial
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazardous-materials/select-options/{hazardousMaterialID}/ [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	hazardousMaterialID, err := pulid.MustParse(c.Param("hazardousMaterialID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetHazardousMaterialByIDRequest{
		ID: hazardousMaterialID,
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

// @Summary List hazardous material options
// @ID listHazardousMaterialOptions
// @Tags Hazardous Materials
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]hazardousmaterial.HazardousMaterial]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazardous-materials/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.HazardousMaterialSelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

// @Summary Get a hazardous material
// @ID getHazardousMaterial
// @Tags Hazardous Materials
// @Produce json
// @Param hazardousMaterialID path string true "Hazardous material ID"
// @Success 200 {object} hazardousmaterial.HazardousMaterial
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazardous-materials/{hazardousMaterialID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	hazardousMaterialID, err := pulid.MustParse(c.Param("hazardousMaterialID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetHazardousMaterialByIDRequest{
			ID: hazardousMaterialID,
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

// @Summary Create a hazardous material
// @ID createHazardousMaterial
// @Tags Hazardous Materials
// @Accept json
// @Produce json
// @Param request body hazardousmaterial.HazardousMaterial true "Hazardous material payload"
// @Success 201 {object} hazardousmaterial.HazardousMaterial
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazardous-materials/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(hazardousmaterial.HazardousMaterial)
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

// @Summary Patch a hazardous material
// @ID patchHazardousMaterial
// @Tags Hazardous Materials
// @Accept json
// @Produce json
// @Param hazardousMaterialID path string true "Hazardous material ID"
// @Param request body hazardousmaterial.HazardousMaterial true "Partial hazardous material payload"
// @Success 200 {object} hazardousmaterial.HazardousMaterial
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazardous-materials/{hazardousMaterialID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	hazardousMaterialID, err := pulid.MustParse(c.Param("hazardousMaterialID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetHazardousMaterialByIDRequest{
			ID: hazardousMaterialID,
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

// @Summary Update a hazardous material
// @ID updateHazardousMaterial
// @Tags Hazardous Materials
// @Accept json
// @Produce json
// @Param hazardousMaterialID path string true "Hazardous material ID"
// @Param request body hazardousmaterial.HazardousMaterial true "Hazardous material payload"
// @Success 200 {object} hazardousmaterial.HazardousMaterial
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazardous-materials/{hazardousMaterialID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	hazardousMaterialID, err := pulid.MustParse(c.Param("hazardousMaterialID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(hazardousmaterial.HazardousMaterial)
	entity.ID = hazardousMaterialID
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
