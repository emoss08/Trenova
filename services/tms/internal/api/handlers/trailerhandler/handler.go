package trailerhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
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

	Service              *trailerservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *trailerservice.Service
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
	api := rg.Group("/trailers")
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceTrailer.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:trailerID/",
		h.pm.RequirePermission(permission.ResourceTrailer.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:trailerID/",
		h.pm.RequirePermission(permission.ResourceTrailer.String(), permission.OpUpdate),
		h.patch,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(permission.ResourceTrailer.String(), permission.OpUpdate),
		h.bulkUpdateStatus,
	)
	api.POST(
		"/:trailerID/locate/",
		h.pm.RequirePermission(permission.ResourceTrailer.String(), permission.OpUpdate),
		h.locate,
	)
}

// @Summary Create a trailer
// @ID createTrailer
// @Tags Trailers
// @Accept json
// @Produce json
// @Param request body trailer.Trailer true "Trailer payload"
// @Success 201 {object} trailer.Trailer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /trailers/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(trailer.Trailer)
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

// @Summary Update a trailer
// @ID updateTrailer
// @Tags Trailers
// @Accept json
// @Produce json
// @Param trailerID path string true "Trailer ID"
// @Param request body trailer.Trailer true "Trailer payload"
// @Success 200 {object} trailer.Trailer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /trailers/{trailerID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	trailerID, err := pulid.MustParse(c.Param("trailerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(trailer.Trailer)
	entity.ID = trailerID
	if err = authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updated)
}

// @Summary Patch a trailer
// @ID patchTrailer
// @Tags Trailers
// @Accept json
// @Produce json
// @Param trailerID path string true "Trailer ID"
// @Param request body trailer.Trailer true "Trailer payload"
// @Success 200 {object} trailer.Trailer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /trailers/{trailerID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	trailerID, err := pulid.MustParse(c.Param("trailerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(c.Request.Context(), trailerports.GetTrailerByIDRequest{
		ID:         trailerID,
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
			IncludeLastKnownLocation: true,
			IncludeCustomFields:      true,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = authctx.BindJSON(c, authCtx, existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updatedEntity, err := h.service.Update(c.Request.Context(), existing, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updatedEntity)
}

// @Summary Bulk update trailer statuses
// @ID bulkUpdateTrailerStatus
// @Tags Trailers
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateTrailerStatusRequest true "Bulk status update request"
// @Success 200 {array} trailer.Trailer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /trailers/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.IsAPIKey() {
		h.eh.HandleError(
			c,
			errortypes.NewAuthorizationError("API keys cannot bulk update trailers"),
		)
		return
	}

	req := new(trailerports.BulkUpdateTrailerStatusRequest)
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

func (h *Handler) locate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	trailerID, err := pulid.MustParse(c.Param("trailerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(trailerports.LocateTrailerRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TrailerID = trailerID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	result, err := h.service.Locate(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}
