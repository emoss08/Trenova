package publicconfighandler

import (
	"context"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const (
	ContributorsGroup = "public_config_contributors"
	cacheControl      = "public, max-age=60"
)

type Params struct {
	fx.In

	Config       *config.Config
	ErrorHandler *helpers.ErrorHandler
	Contributors []kernelports.PublicConfigContributor `group:"public_config_contributors"`
}

type Handler struct {
	cfg          *config.Config
	eh           *helpers.ErrorHandler
	contributors []kernelports.PublicConfigContributor
}

func New(p Params) *Handler {
	return &Handler{
		cfg:          p.Config,
		eh:           p.ErrorHandler,
		contributors: p.Contributors,
	}
}

func AsContributor(constructor any) any {
	return fx.Annotate(
		constructor,
		fx.As(new(kernelports.PublicConfigContributor)),
		fx.ResultTags(`group:"`+ContributorsGroup+`"`),
	)
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
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Build(ctx context.Context) (*kernelports.PublicConfig, error) {
	resp := &kernelports.PublicConfig{
		PlatformMode: string(h.cfg.Platform.GetMode()),
		FreePlan:     kernelports.PublicFreePlanSummary{Limits: map[string]int64{}},
		Session:      publicSessionPolicy(&h.cfg.Security.Session),
	}

	for _, contributor := range h.contributors {
		if contributor == nil {
			continue
		}
		if err := contributor.ContributePublicConfig(ctx, resp); err != nil {
			return nil, err
		}
	}

	if resp.FreePlan.Limits == nil {
		resp.FreePlan.Limits = map[string]int64{}
	}

	return resp, nil
}

func publicSessionPolicy(cfg *config.SessionConfig) kernelports.PublicSessionPolicy {
	if !cfg.IdleTimeoutEnabled() {
		return kernelports.PublicSessionPolicy{}
	}

	return kernelports.PublicSessionPolicy{
		IdleTimeoutSeconds: int64(cfg.IdleTimeout / time.Second),
		IdleWarningSeconds: int64(cfg.IdleWarning / time.Second),
		KeepAliveSeconds:   int64(session.KeepAliveInterval / time.Second),
	}
}
