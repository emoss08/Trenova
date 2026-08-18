package ratezonehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/ratezone"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ratezoneservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *ratezoneservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *ratezoneservice.Service
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
	resource := permission.ResourceRateZone.String()

	api := rg.Group("/rate-zones")
	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/:rateZoneID/", h.pm.RequirePermission(resource, permission.OpRead), h.get)
	api.POST("/", h.pm.RequirePermission(resource, permission.OpCreate), h.create)
	api.PUT("/:rateZoneID/", h.pm.RequirePermission(resource, permission.OpUpdate), h.update)
	api.DELETE("/:rateZoneID/", h.pm.RequirePermission(resource, permission.OpDelete), h.delete)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
}

// @Summary List rate zones
// @ID listRateZones
// @Tags Rate Zones
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratezone.RateZone]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-zones/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratezone.RateZone], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListRateZoneRequest{Filter: req},
			)
		},
	)
}

// @Summary Get a rate zone
// @ID getRateZone
// @Tags Rate Zones
// @Produce json
// @Param rateZoneID path string true "Rate zone ID"
// @Success 200 {object} ratezone.RateZone
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-zones/{rateZoneID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	zoneID, err := h.zoneID(c)
	if err != nil {
		return
	}

	entity, err := h.service.GetByID(c.Request.Context(), &repositories.GetRateZoneByIDRequest{
		RateZoneID:     zoneID,
		TenantInfo:     tenantOf(authCtx),
		IncludeMembers: true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a rate zone
// @ID createRateZone
// @Tags Rate Zones
// @Accept json
// @Produce json
// @Param request body ratezone.RateZone true "Rate zone payload"
// @Success 201 {object} ratezone.RateZone
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-zones/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(ratezone.RateZone)
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

// @Summary Update a rate zone
// @ID updateRateZone
// @Tags Rate Zones
// @Accept json
// @Produce json
// @Param rateZoneID path string true "Rate zone ID"
// @Param request body ratezone.RateZone true "Rate zone payload"
// @Success 200 {object} ratezone.RateZone
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-zones/{rateZoneID} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	zoneID, err := h.zoneID(c)
	if err != nil {
		return
	}

	entity := new(ratezone.RateZone)
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity.ID = zoneID

	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Delete a rate zone
// @ID deleteRateZone
// @Tags Rate Zones
// @Produce json
// @Param rateZoneID path string true "Rate zone ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-zones/{rateZoneID} [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	zoneID, err := h.zoneID(c)
	if err != nil {
		return
	}

	if err = h.service.Delete(c.Request.Context(), &repositories.GetRateZoneByIDRequest{
		RateZoneID: zoneID,
		TenantInfo: tenantOf(authCtx),
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary List rate zone options
// @ID listRateZoneOptions
// @Tags Rate Zones
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratezone.RateZone]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-zones/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratezone.RateZone], error) {
			return h.service.SelectOptions(c.Request.Context(), req)
		},
	)
}

func (h *Handler) zoneID(c *gin.Context) (pulid.ID, error) {
	zoneID, err := pulid.MustParse(c.Param("rateZoneID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pulid.Nil, err
	}

	return zoneID, nil
}

func tenantOf(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
