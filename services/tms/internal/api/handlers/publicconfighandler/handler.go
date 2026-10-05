package publicconfighandler

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
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
	Contributors []services.PublicConfigContributor `group:"public_config_contributors"`
}

type Handler struct {
	cfg          *config.Config
	eh           *helpers.ErrorHandler
	contributors []services.PublicConfigContributor
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
		fx.As(new(services.PublicConfigContributor)),
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

func (h *Handler) Build(ctx context.Context) (*services.PublicConfig, error) {
	resp := &services.PublicConfig{
		PlatformMode: string(h.cfg.Platform.GetMode()),
		FreePlan:     services.PublicFreePlanSummary{Limits: map[string]int64{}},
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
