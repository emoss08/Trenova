/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package usstate

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	usstate "github.com/emoss08/trenova/internal/core/services/usstate"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	os *usstate.Service
	eh *validator.ErrorHandler
}

func NewHandler(os *usstate.Service, eh *validator.ErrorHandler) *Handler {
	return &Handler{os: os, eh: eh}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/us-states")

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)
}

func (h *Handler) selectOptions(c *fiber.Ctx) error {
	_, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	options, err := h.os.SelectOptions(c.UserContext())
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"results": options})
}
