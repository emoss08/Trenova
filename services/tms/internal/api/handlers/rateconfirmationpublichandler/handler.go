package rateconfirmationpublichandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/services/rateconfirmationservice"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const tokenParam = "token"

type Params struct {
	fx.In

	Service      *rateconfirmationservice.Service
	ErrorHandler *helpers.ErrorHandler
	RateLimiter  *middleware.RateLimiter
}

type Handler struct {
	service     *rateconfirmationservice.Service
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

// RegisterPublicRoutes lives under /rate-confirmation-links rather than
// /rate-confirmations: the authed routes already bind
// /rate-confirmations/:rateConfirmationID, and gin rejects a second wildcard
// name in the same segment position.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/rate-confirmation-links")
	api.Use(h.rateLimiter.ByPublicToken(tokenParam))
	api.GET("/:token/", h.preview)
	api.POST("/:token/confirm/", h.confirm)
}

func (h *Handler) preview(c *gin.Context) {
	view, err := h.service.PreviewByToken(c.Request.Context(), c.Param(tokenParam))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, view)
}

type confirmRequest struct {
	SignerName  string `json:"signerName"`
	SignerTitle string `json:"signerTitle"`
}

func (h *Handler) confirm(c *gin.Context) {
	var req confirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = confirmRequest{}
	}

	if err := h.service.ConfirmByToken(
		c.Request.Context(), c.Param(tokenParam), req.SignerName, req.SignerTitle,
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}
