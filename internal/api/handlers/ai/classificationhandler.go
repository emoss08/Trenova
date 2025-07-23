// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package ai

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/ai"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type Handler struct {
	logger  *zerolog.Logger
	service services.AIClassificationService
	eh      *validator.ErrorHandler
}

type Params struct {
	fx.In

	Logger       *logger.Logger
	Service      services.AIClassificationService
	ErrorHandler *validator.ErrorHandler
}

func New(p Params) *Handler {
	log := p.Logger.With().
		Str("handler", "ai_classification").
		Logger()

	return &Handler{
		logger:  &log,
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

// RegisterRoutes registers all AI classification routes
func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/ai/classify")

	// Location classification routes
	api.Post("/location", rl.WithRateLimit(
		[]fiber.Handler{h.classifyLocation},
		middleware.PerMinute(60), // 60 requests per minute for AI calls
	)...)

	api.Post("/location/batch", rl.WithRateLimit(
		[]fiber.Handler{h.classifyLocationBatch},
		middleware.PerMinute(20), // 20 batch requests per minute
	)...)
}

func (h *Handler) classifyLocation(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req ai.ClassificationRequest
	if err := c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	req.TenantOpts = &ports.TenantOptions{
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
		UserID: reqCtx.UserID,
	}

	h.logger.Info().
		Str("locationName", req.Name).
		Msg("Classifying location")

	response, err := h.service.ClassifyLocation(c.UserContext(), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to classify location")
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *Handler) classifyLocationBatch(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req ai.BatchClassificationRequest
	if err := c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	h.logger.Info().
		Int("location_count", len(req.Locations)).
		Msg("Classifying locations in batch")

	response, err := h.service.ClassifyLocationBatch(ctx, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to classify locations in batch")
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
