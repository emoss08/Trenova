package tractorhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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
	entity.IFTAQualified = true
	if err := authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	created, err := h.service.Create(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, created)
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
	if err = authctx.BindJSON(c, authCtx, entity); err != nil {
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

	helpers.JSON(c, http.StatusOK, updated)
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
		tractorports.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = authctx.BindJSON(c, authCtx, existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), existing, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updated)
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

	req := new(tractorports.BulkUpdateTractorStatusRequest)
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

	helpers.JSON(c, http.StatusOK, results)
}
