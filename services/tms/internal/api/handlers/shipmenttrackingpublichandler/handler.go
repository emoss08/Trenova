package shipmenttrackingpublichandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const tokenParam = "token"

type Params struct {
	fx.In

	Service      customertrackingports.CustomerTrackingService
	ErrorHandler *helpers.ErrorHandler
	RateLimiter  *middleware.RateLimiter
}

// Handler serves the customer's shipment tracking page. It is public: the token
// in the path is the only credential, and the service answers every bad token
// the same way.
type Handler struct {
	service     customertrackingports.CustomerTrackingService
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
	api := rg.Group("/shipment-tracking-links")
	api.Use(h.rateLimiter.ByPublicToken(tokenParam))
	api.GET("/:token/", h.preview)
	api.GET("/:token/proof-of-delivery/", h.downloadPod)
}

func (h *Handler) preview(c *gin.Context) {
	view, err := h.service.PreviewByToken(c.Request.Context(), c.Param(tokenParam))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.JSON(http.StatusOK, view)
}

func (h *Handler) downloadPod(c *gin.Context) {
	result, err := h.service.DownloadPodByToken(c.Request.Context(), c.Param(tokenParam))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	defer func() { _ = result.Body.Close() }()
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.DataFromReader(
		http.StatusOK,
		result.ContentLength,
		result.ContentType,
		result.Body,
		map[string]string{"Content-Disposition": result.ContentDisposition},
	)
}
