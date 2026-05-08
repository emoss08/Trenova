package analyticshandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
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
)

type Params struct {
	fx.In

	Service      services.AnalyticsService
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service services.AnalyticsService
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
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

	opts := &services.AnalyticsRequestOptions{
		OrgID:      authCtx.OrganizationID,
		BuID:       authCtx.BusinessUnitID,
		UserID:     authCtx.UserID,
		Page:       req.Page,
		Limit:      req.Limit,
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
			validation.In("", includeLaneHeatmap).
				Error("Include must be laneHeatmap when provided"),
		),
	)

	me.AddOzzoError(err)
	if me.HasErrors() {
		return me
	}

	return nil
}
