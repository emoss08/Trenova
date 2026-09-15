package tenderpublichandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const tokenParam = "token"

type Params struct {
	fx.In

	Service      *tenderservice.Service
	ErrorHandler *helpers.ErrorHandler
	RateLimiter  *middleware.RateLimiter
}

type Handler struct {
	service     *tenderservice.Service
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
}

func (h *Handler) preview(c *gin.Context) {
	view, err := h.service.PreviewByToken(c.Request.Context(), c.Param(tokenParam))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, view)
}

type declineRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) accept(c *gin.Context) {
	if err := h.service.RespondByToken(
		c.Request.Context(), c.Param(tokenParam), tender.ResponseActionAccept, "",
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

func (h *Handler) decline(c *gin.Context) {
	var req declineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = ""
	}

	if err := h.service.RespondByToken(
		c.Request.Context(), c.Param(tokenParam), tender.ResponseActionDecline, req.Reason,
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}
