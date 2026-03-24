package fleetcodehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fleetcodeservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *fleetcodeservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *fleetcodeservice.Service
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
	api := rg.Group("/fleet-codes")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceFleetCode.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:fleetCodeID",
		h.pm.RequirePermission(permission.ResourceFleetCode.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceFleetCode.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:fleetCodeID",
		h.pm.RequirePermission(permission.ResourceFleetCode.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:fleetCodeID",
		h.pm.RequirePermission(permission.ResourceFleetCode.String(), permission.OpUpdate),
		h.patch,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:fleetCodeID", h.getOption)
}

// @Summary List fleet codes
// @ID listFleetCodes
// @Tags Fleet Codes
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param includeManagerDetails query bool false "Include fleet manager details"
// @Param status query string false "Filter by status"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]fleetcode.FleetCode]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fleet-codes/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*fleetcode.FleetCode], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListFleetCodesRequest{
					Filter:                req,
					IncludeManagerDetails: helpers.QueryBool(c, "includeManagerDetails", false),
				},
			)
		},
	)
}

// @Summary Get a fleet code option
// @ID getFleetCodeOption
// @Tags Fleet Codes
// @Produce json
// @Param fleetCodeID path string true "Fleet code ID"
// @Success 200 {object} fleetcode.FleetCode
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fleet-codes/select-options/{fleetCodeID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fleetCodeID, err := pulid.MustParse(c.Param("fleetCodeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetFleetCodeByIDRequest{
			ID: fleetCodeID,
			TenantInfo: &pagination.TenantInfo{
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

// @Summary List fleet code options
// @ID listFleetCodeOptions
// @Tags Fleet Codes
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]fleetcode.FleetCode]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fleet-codes/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*fleetcode.FleetCode], error) {
			return h.service.SelectOptions(c.Request.Context(), req)
		},
	)
}

// @Summary Get a fleet code
// @ID getFleetCode
// @Tags Fleet Codes
// @Produce json
// @Param fleetCodeID path string true "Fleet code ID"
// @Success 200 {object} fleetcode.FleetCode
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fleet-codes/{fleetCodeID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fleetCodeID, err := pulid.MustParse(c.Param("fleetCodeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetFleetCodeByIDRequest{
			ID: fleetCodeID,
			TenantInfo: &pagination.TenantInfo{
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

// @Summary Create a fleet code
// @ID createFleetCode
// @Tags Fleet Codes
// @Accept json
// @Produce json
// @Param request body fleetcode.FleetCode true "Fleet code payload"
// @Success 201 {object} fleetcode.FleetCode
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fleet-codes/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(fleetcode.FleetCode)
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

// @Summary Update a fleet code
// @ID updateFleetCode
// @Tags Fleet Codes
// @Accept json
// @Produce json
// @Param fleetCodeID path string true "Fleet code ID"
// @Param request body fleetcode.FleetCode true "Fleet code payload"
// @Success 200 {object} fleetcode.FleetCode
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fleet-codes/{fleetCodeID} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fleetCodeID, err := pulid.MustParse(c.Param("fleetCodeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(fleetcode.FleetCode)
	entity.ID = fleetCodeID
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

// @Summary Patch a fleet code
// @ID patchFleetCode
// @Tags Fleet Codes
// @Accept json
// @Produce json
// @Param fleetCodeID path string true "Fleet code ID"
// @Param request body fleetcode.FleetCode true "Partial fleet code payload"
// @Success 200 {object} fleetcode.FleetCode
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fleet-codes/{fleetCodeID} [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fleetCodeID, err := pulid.MustParse(c.Param("fleetCodeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(fleetcode.FleetCode)
	entity.ID = fleetCodeID
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updatedEntity, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}
