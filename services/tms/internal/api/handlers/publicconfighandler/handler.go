package publicconfighandler

import (
	"context"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config/platformconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const cacheControl = "public, max-age=60"

type Params struct {
	fx.In

	PlatformConfig *platformconfig.Config
	SecurityConfig *securityconfig.Config
	ErrorHandler   *helpers.ErrorHandler
}

type Handler struct {
	platformConfig *platformconfig.Config
	securityConfig *securityconfig.Config
	eh             *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		platformConfig: p.PlatformConfig,
		securityConfig: p.SecurityConfig,
		eh:             p.ErrorHandler,
	}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	system := rg.Group("/system")
	system.GET("/public-config", h.publicConfig)
}

func (h *Handler) publicConfig(c *gin.Context) {
	resp, err := h.Build(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", cacheControl)
	helpers.JSON(c, http.StatusOK, resp)
}

func (h *Handler) Build(context.Context) (*kernelports.PublicConfig, error) {
	return &kernelports.PublicConfig{
		PlatformMode: string(h.platformConfig.GetMode()),
		Session:      publicSessionPolicy(&h.securityConfig.Session),
	}, nil
}

func publicSessionPolicy(cfg *securityconfig.SessionConfig) kernelports.PublicSessionPolicy {
	if !cfg.IdleTimeoutEnabled() {
		return kernelports.PublicSessionPolicy{}
	}

	return kernelports.PublicSessionPolicy{
		IdleTimeoutSeconds: int64(cfg.IdleTimeout / time.Second),
		IdleWarningSeconds: int64(cfg.IdleWarning / time.Second),
		KeepAliveSeconds:   int64(session.KeepAliveInterval / time.Second),
	}
}
