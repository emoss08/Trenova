package analyticshandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"go.uber.org/fx"
)

const (
	defaultAnalyticsWindowDays = 7
	minAnalyticsWindowDays     = 1
	maxAnalyticsWindowDays     = 90
	includeLaneHeatmap         = "laneHeatmap"
	includeTomorrowsPickups    = "tomorrowsPickups"
	includeSavedViewCounts     = "savedViewCounts"
)

// pagePermissions names the resource a caller must be able to read before an
// analytics page will answer. Every page must appear here: an unlisted page is
// refused rather than served, so adding a provider without deciding who may see
// it fails closed.
var pagePermissions = map[services.AnalyticsPage]permission.Resource{
	services.ShipmentAnalyticsPage:        permission.ResourceShipment,
	services.BillingClientAnalyticsPage:   permission.ResourceBillingQueue,
	services.DedicatedLaneSuggestionsPage: permission.ResourceShipment,
	services.APIKeyAnalyticsPage:          permission.ResourceAPIKey,
}

type Params struct {
	fx.In

	Service          services.AnalyticsService
	PermissionEngine services.PermissionEngine
	ErrorHandler     *helpers.ErrorHandler
}

type Handler struct {
	service     services.AnalyticsService
	permissions services.PermissionEngine
	eh          *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service:     p.Service,
		permissions: p.PermissionEngine,
		eh:          p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/analytics")

	api.GET("/", h.get)
}

// @Summary Get analytics data
// @Description Returns analytics data for the requested application page.
// @ID getAnalytics
// @Tags Analytics
// @Produce json
// @Param page query string true "Analytics page"
// @Param startDate query int false "Start date as Unix timestamp"
// @Param endDate query int false "End date as Unix timestamp"
// @Param limit query int false "Result limit"
// @Param offset query int false "Result offset"
// @Param timezone query string false "IANA timezone name"
// @Param windowDays query int false "Rolling analytics window in days"
// @Param include query string false "Optional analytics section to include"
// @Success 200 {object} services.AnalyticsData
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /analytics/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(services.AnaltyicsRequest)

	if err := c.ShouldBindQuery(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err := validateAnalyticsRequest(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	// The page is a query parameter, so the gate cannot live in route
	// middleware — it has to be applied once the request is bound.
	if err := h.authorizePage(c, req.Page); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	opts := &services.AnalyticsRequestOptions{
		OrgID:      authCtx.OrganizationID,
		BuID:       authCtx.BusinessUnitID,
		UserID:     authCtx.UserID,
		Page:       req.Page,
		Limit:      req.Limit,
		Offset:     req.Offset,
		Timezone:   req.Timezone,
		WindowDays: req.WindowDays,
		Include:    req.Include,
	}

	if req.StartDate > 0 && req.EndDate > 0 {
		opts.DateRange = &services.DateRange{
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
		}
	}

	data, err := h.service.GetAnalytics(c.Request.Context(), opts)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) authorizePage(c *gin.Context, page services.AnalyticsPage) error {
	resource, ok := pagePermissions[page]
	if !ok {
		return errortypes.NewAuthorizationError(
			"You don't have permission to view this analytics page",
		)
	}

	result, err := h.permissions.Check(
		c.Request.Context(),
		middleware.BuildPermissionCheckRequest(
			authctx.GetAuthContext(c),
			resource.String(),
			permission.OpRead,
		),
	)
	if err != nil {
		return err
	}

	if !result.Allowed {
		return errortypes.NewAuthorizationError(
			"You don't have permission to view this analytics page",
		)
	}

	return nil
}

func validateAnalyticsRequest(req *services.AnaltyicsRequest) error {
	if req.WindowDays == 0 {
		req.WindowDays = defaultAnalyticsWindowDays
	}

	me := errortypes.NewMultiError()
	err := validation.ValidateStruct(
		req,
		validation.Field(
			&req.Page,
			validation.Required.Error("Page is required"),
		),
		validation.Field(
			&req.WindowDays,
			validation.Min(minAnalyticsWindowDays).Error("Window days must be at least 1"),
			validation.Max(maxAnalyticsWindowDays).Error("Window days must be at most 90"),
		),
		validation.Field(
			&req.Include,
			validation.In("", includeLaneHeatmap, includeTomorrowsPickups, includeSavedViewCounts).
				Error("Include must be laneHeatmap, tomorrowsPickups, or savedViewCounts when provided"),
		),
		validation.Field(
			&req.Offset,
			validation.Min(0).Error("Offset must be at least 0"),
		),
	)

	me.AddOzzoError(err)
	if me.HasErrors() {
		return me
	}

	return nil
}
