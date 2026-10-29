package tenderpublichandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderresponse"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

func responseRecorded() gin.H {
	return gin.H{"status": "recorded"}
}

const tokenParam = "token"

type Params struct {
	fx.In

	Service      *tenderresponse.Service
	ErrorHandler *helpers.ErrorHandler
	RateLimiter  *middleware.RateLimiter
}

type Handler struct {
	service     *tenderresponse.Service
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
	api := rg.Group("/tender-offers")
	api.Use(h.rateLimiter.ByPublicToken(tokenParam))
	api.GET("/:token/", h.preview)
	api.POST("/:token/accept/", h.accept)
	api.POST("/:token/decline/", h.decline)
	api.POST("/:token/counter/", h.counter)
}

func (h *Handler) preview(c *gin.Context) {
	view, err := h.service.PreviewByToken(c.Request.Context(), c.Param(tokenParam))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, view)
}

type declineRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) accept(c *gin.Context) {
	if err := h.service.RespondByToken(
		c.Request.Context(), c.Param(tokenParam),
		tenderresponse.PublicResponse{Action: tender.ResponseActionAccept},
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, responseRecorded())
}

func (h *Handler) decline(c *gin.Context) {
	var req declineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = ""
	}

	if err := h.service.RespondByToken(
		c.Request.Context(),
		c.Param(tokenParam),
		tenderresponse.PublicResponse{
			Action:        tender.ResponseActionDecline,
			DeclineReason: req.Reason,
		},
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, responseRecorded())
}

type counterRequest struct {
	Rate decimal.Decimal `json:"rate"`
	Note string          `json:"note"`
}

func (h *Handler) counter(c *gin.Context) {
	var req counterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"rate", errortypes.ErrInvalid, "Enter the rate you would run the load for",
		))
		return
	}

	if err := h.service.RespondByToken(
		c.Request.Context(), c.Param(tokenParam),
		tenderresponse.PublicResponse{
			Action:      tender.ResponseActionCounter,
			CounterRate: req.Rate,
			CounterNote: req.Note,
		},
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, responseRecorded())
}
