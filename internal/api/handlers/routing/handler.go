package routing

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/routing"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	RoutingService *routing.Service
	ErrorHandler   *validator.ErrorHandler
}

type Handler struct {
	rs *routing.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		rs: p.RoutingService,
		eh: p.ErrorHandler,
	}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/routing")

	api.Get("/single-search", rl.WithRateLimit(
		[]fiber.Handler{h.singleSearch},
		middleware.PerMinute(120),
	)...)
}

func (h Handler) singleSearch(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := routing.SingleSearchParams{
		Query: c.Query("query"),
		ConfigOpts: repositories.GetPCMilerConfigurationOptions{
			OrgID: reqCtx.OrgID,
			BuID:  reqCtx.BuID,
		},
	}

	resp, err := h.rs.SingleSearch(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(resp)
}
