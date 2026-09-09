package networkpulsehandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/services/networkpulseservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      *networkpulseservice.Service
	Config       *config.Config
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *networkpulseservice.Service
	cfg     *config.Config
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		cfg:     p.Config,
		eh:      p.ErrorHandler,
	}
}

// RegisterPublicRoutes mounts the pulse alongside /system/version, the other endpoint
// the sign-in screen reads before a session exists. The route is registered whether or
// not the feature is on; when it is off the handler answers 404, so a deployment that
// has not opted in does not advertise that the figures exist.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/system")
	api.GET("/network-pulse", h.getNetworkPulse)
}

func (h *Handler) getNetworkPulse(c *gin.Context) {
	pulse, err := h.service.GetNetworkPulse(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	// The snapshot is shared and already TTL-cached in the service; letting a proxy hold
	// it for the same window keeps repeated anonymous page loads off the process too.
	ttlSeconds := int(h.cfg.System.NetworkPulse.GetCacheTTL().Seconds())
	c.Header("Cache-Control", "public, max-age="+strconv.Itoa(ttlSeconds))
	c.JSON(http.StatusOK, pulse)
}
