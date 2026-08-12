package routingguidehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/routingguideservice"
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
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceRoutingGuide.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/match/",
		h.pm.RequirePermission(permission.ResourceRoutingGuide.String(), permission.OpRead),
		h.match,
	)
	api.GET(
		"/:guideID/",
		h.pm.RequirePermission(permission.ResourceRoutingGuide.String(), permission.OpRead),
		h.get,
	)
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

// @Summary List routing guides
// @ID listRoutingGuides
// @Tags Routing Guides
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param includeEntries query bool false "Include ranked carrier entries"
// @Success 200 {object} pagination.Response[[]tender.RoutingGuide]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /routing-guides/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*tender.RoutingGuide], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListRoutingGuideRequest{
					Filter: req,
					RoutingGuideFilterOptions: repositories.RoutingGuideFilterOptions{
						IncludeEntries: helpers.QueryBool(c, "includeEntries"),
					},
				},
			)
		},
	)
}

// @Summary Match a lane to its routing guide
// @ID matchRoutingGuide
// @Tags Routing Guides
// @Produce json
// @Param originLocationId query string false "Origin location ID"
// @Param destinationLocationId query string false "Destination location ID"
// @Param originCity query string false "Origin city"
// @Param originState query string false "Origin state abbreviation"
// @Param destinationCity query string false "Destination city"
// @Param destinationState query string false "Destination state abbreviation"
// @Success 200 {object} tender.RoutingGuide
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /routing-guides/match/ [get]
func (h *Handler) match(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := &repositories.MatchLaneRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		OriginCity:       c.Query("originCity"),
		OriginState:      c.Query("originState"),
		DestinationCity:  c.Query("destinationCity"),
		DestinationState: c.Query("destinationState"),
	}
	if raw := c.Query("originLocationId"); raw != "" {
		id, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		req.OriginLocationID = id
	}
	if raw := c.Query("destinationLocationId"); raw != "" {
		id, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		req.DestinationLocationID = id
	}

	guide, err := h.service.MatchLane(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, guide)
}

// @Summary Get a routing guide
// @ID getRoutingGuide
// @Tags Routing Guides
// @Produce json
// @Param guideID path string true "Routing guide ID"
// @Success 200 {object} tender.RoutingGuide
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /routing-guides/{guideID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	guideID, err := pulid.MustParse(c.Param("guideID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetRoutingGuideByIDRequest{
		ID: guideID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		RoutingGuideFilterOptions: repositories.RoutingGuideFilterOptions{
			IncludeEntries: true,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
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
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
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

	c.JSON(http.StatusOK, updated)
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
	if err = h.service.Delete(c.Request.Context(), repositories.DeleteRoutingGuideRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		ID: guideID,
	}, actor); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
