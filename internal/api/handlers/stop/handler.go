// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package stop

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/stop"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	ss *stop.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	StopService  *stop.Service
	ErrorHandler *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ss: p.StopService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/stops")

	api.Get("/:stopID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(300), // 300 reads per minute
	)...)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	stopID, err := pulid.MustParse(c.Params("stopID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	stp, err := h.ss.Get(c.UserContext(), repositories.GetStopByIDRequest{
		StopID:            stopID,
		BuID:              reqCtx.BuID,
		OrgID:             reqCtx.OrgID,
		UserID:            reqCtx.UserID,
		ExpandStopDetails: c.QueryBool("expandStopDetails"),
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(stp)
}
