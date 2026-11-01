package routingguidehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/routingguideservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *routingguideservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *routingguideservice.Service
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
	api := rg.Group("/routing-guides")
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceRoutingGuide.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:guideID/",
		h.pm.RequirePermission(permission.ResourceRoutingGuide.String(), permission.OpUpdate),
		h.update,
	)
	api.DELETE(
		"/:guideID/",
		h.pm.RequirePermission(permission.ResourceRoutingGuide.String(), permission.OpDelete),
		h.delete,
	)
}

// @Summary Create a routing guide
// @ID createRoutingGuide
// @Tags Routing Guides
// @Accept json
// @Produce json
// @Param request body tender.RoutingGuide true "Routing guide payload"
// @Success 201 {object} tender.RoutingGuide
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /routing-guides/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(tender.RoutingGuide)
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

// @Summary Update a routing guide
// @ID updateRoutingGuide
// @Tags Routing Guides
// @Accept json
// @Produce json
// @Param guideID path string true "Routing guide ID"
// @Param request body tender.RoutingGuide true "Routing guide payload"
// @Success 200 {object} tender.RoutingGuide
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /routing-guides/{guideID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	guideID, err := pulid.MustParse(c.Param("guideID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(tender.RoutingGuide)
	if err = authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity.ID = guideID

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updated)
}

// @Summary Delete a routing guide
// @ID deleteRoutingGuide
// @Tags Routing Guides
// @Produce json
// @Param guideID path string true "Routing guide ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /routing-guides/{guideID}/ [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	guideID, err := pulid.MustParse(c.Param("guideID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	if err = h.service.Delete(c.Request.Context(), tenderports.DeleteRoutingGuideRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		ID:         guideID,
	}, actor); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
