package carriertrackingpublichandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const tokenParam = "token"

type Params struct {
	fx.In

	Service      checkcallports.CheckCallService
	ErrorHandler *helpers.ErrorHandler
	RateLimiter  *middleware.RateLimiter
}

type Handler struct {
	service     checkcallports.CheckCallService
	eh          *helpers.ErrorHandler
	rateLimiter *middleware.RateLimiter
}

func New(p Params) *Handler {
	return &Handler{
		service:     p.Service,
		eh:          p.ErrorHandler,
		rateLimiter: p.RateLimiter,
	}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/carrier-tracking-links")
	api.Use(h.rateLimiter.ByPublicToken(tokenParam))
	api.GET("/:token/", h.preview)
	api.POST("/:token/check-calls/", h.post)
}

func (h *Handler) preview(c *gin.Context) {
	view, err := h.service.PreviewByToken(c.Request.Context(), c.Param(tokenParam))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, view)
}

type postRequest struct {
	Kind             checkcall.Kind `json:"kind"`
	StopID           pulid.ID       `json:"stopId"`
	ReportedAt       *int64         `json:"reportedAt"`
	Latitude         *float64       `json:"latitude"`
	Longitude        *float64       `json:"longitude"`
	City             string         `json:"city"`
	StateCode        string         `json:"stateCode"`
	Delayed          bool           `json:"delayed"`
	DelayReason      string         `json:"delayReason"`
	EstimatedArrival *int64         `json:"estimatedArrival"`
	Note             string         `json:"note"`
	ContactName      string         `json:"contactName"`
	ClientEventID    string         `json:"clientEventId"`
}

func (h *Handler) post(c *gin.Context) {
	var req postRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.PostByToken(
		c.Request.Context(),
		c.Param(tokenParam),
		&checkcallports.PublicCheckCallInput{
			Details: checkcallports.CheckCallDetails{
				Kind:             req.Kind,
				StopID:           req.StopID,
				ReportedAt:       req.ReportedAt,
				Latitude:         req.Latitude,
				Longitude:        req.Longitude,
				City:             req.City,
				StateCode:        req.StateCode,
				Delayed:          req.Delayed,
				DelayReason:      req.DelayReason,
				EstimatedArrival: req.EstimatedArrival,
				Note:             req.Note,
				ContactName:      req.ContactName,
			},
			ClientEventID: req.ClientEventID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}
