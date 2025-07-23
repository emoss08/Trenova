// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package consolidationsetting

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/consolidationsetting"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	ConsolidationSettingService *consolidationsetting.Service
	ErrorHandler                *validator.ErrorHandler
}

type Handler struct {
	cs *consolidationsetting.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{cs: p.ConsolidationSettingService, eh: p.ErrorHandler}
}
func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/consolidation-settings")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(120), // * 120 reads per minute
	)...)

	api.Put("/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(120), // * 120 writes per minute
	)...)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	cs, err := h.cs.Get(c.UserContext(), &repositories.GetConsolidationSettingRequest{
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(cs)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	cs := new(consolidation.ConsolidationSettings)
	if err = c.BodyParser(cs); err != nil {
		return h.eh.HandleError(c, err)
	}

	cs.OrganizationID = reqCtx.OrgID
	cs.BusinessUnitID = reqCtx.BuID

	entity, err := h.cs.Update(c.UserContext(), cs, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}
