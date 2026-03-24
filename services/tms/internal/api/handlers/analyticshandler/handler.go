package analyticshandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
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

	if req.Page == "" {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError("page", errortypes.ErrInvalid, "").Internal,
		)
		return
	}

	opts := &services.AnalyticsRequestOptions{
		OrgID:    authCtx.OrganizationID,
		BuID:     authCtx.BusinessUnitID,
		UserID:   authCtx.UserID,
		Page:     req.Page,
		Limit:    req.Limit,
		Timezone: req.Timezone,
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
