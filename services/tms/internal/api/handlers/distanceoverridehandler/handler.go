package distanceoverridehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/distanceoverrideservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *distanceoverrideservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *distanceoverrideservice.Service
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
	api := rg.Group("/distance-overrides")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceDistanceOverride.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:distanceOverrideID",
		h.pm.RequirePermission(permission.ResourceDistanceOverride.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceDistanceOverride.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:distanceOverrideID/",
		h.pm.RequirePermission(permission.ResourceDistanceOverride.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:distanceOverrideID/",
		h.pm.RequirePermission(permission.ResourceDistanceOverride.String(), permission.OpUpdate),
		h.patch,
	)
	api.DELETE(
		"/:distanceOverrideID/",
		h.pm.RequirePermission(permission.ResourceDistanceOverride.String(), permission.OpDelete),
		h.delete,
	)
}

// @Summary List distance overrides
// @ID listDistanceOverrides
// @Tags Distance Overrides
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]distanceoverride.DistanceOverride]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /distance-overrides/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*distanceoverride.DistanceOverride], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListDistanceOverrideRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get a distance override
// @ID getDistanceOverride
// @Tags Distance Overrides
// @Produce json
// @Param distanceOverrideID path string true "Distance override ID"
// @Success 200 {object} distanceoverride.DistanceOverride
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /distance-overrides/{distanceOverrideID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	distanceOverrideID, err := pulid.MustParse(c.Param("distanceOverrideID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetDistanceOverrideByIDRequest{
			ID: distanceOverrideID,
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

// @Summary Create a distance override
// @ID createDistanceOverride
// @Tags Distance Overrides
// @Accept json
// @Produce json
// @Param request body distanceoverride.DistanceOverride true "Distance override payload"
// @Success 201 {object} distanceoverride.DistanceOverride
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /distance-overrides/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(distanceoverride.DistanceOverride)
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

// @Summary Update a distance override
// @ID updateDistanceOverride
// @Tags Distance Overrides
// @Accept json
// @Produce json
// @Param distanceOverrideID path string true "Distance override ID"
// @Param request body distanceoverride.DistanceOverride true "Distance override payload"
// @Success 200 {object} distanceoverride.DistanceOverride
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /distance-overrides/{distanceOverrideID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	distanceOverrideID, err := pulid.MustParse(c.Param("distanceOverrideID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(distanceoverride.DistanceOverride)
	entity.ID = distanceOverrideID
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

// @Summary Patch a distance override
// @ID patchDistanceOverride
// @Tags Distance Overrides
// @Accept json
// @Produce json
// @Param distanceOverrideID path string true "Distance override ID"
// @Param request body distanceoverride.DistanceOverride true "Distance override payload"
// @Success 200 {object} distanceoverride.DistanceOverride
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /distance-overrides/{distanceOverrideID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	distanceOverrideID, err := pulid.MustParse(c.Param("distanceOverrideID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetDistanceOverrideByIDRequest{
			ID: distanceOverrideID,
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

	updatedEntity, err := h.service.Update(c.Request.Context(), existing, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

// @Summary Delete a distance override
// @ID deleteDistanceOverride
// @Tags Distance Overrides
// @Param distanceOverrideID path string true "Distance override ID"
// @Success 204 "No Content"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /distance-overrides/{distanceOverrideID}/ [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	distanceOverrideID, err := pulid.MustParse(c.Param("distanceOverrideID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Delete(c.Request.Context(), repositories.DeleteDistanceOverrideRequest{
		ID: distanceOverrideID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
