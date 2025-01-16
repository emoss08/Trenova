package usstate

import (
	"github.com/gofiber/fiber/v2"
	"github.com/trenova-app/transport/internal/api/middleware"
	usstate "github.com/trenova-app/transport/internal/core/services/usstate"
	"github.com/trenova-app/transport/internal/pkg/ctx"
	"github.com/trenova-app/transport/internal/pkg/validator"
)

type Handler struct {
	os *usstate.Service
	eh *validator.ErrorHandler
}

func NewHandler(os *usstate.Service, eh *validator.ErrorHandler) *Handler {
	return &Handler{os: os, eh: eh}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/us-states")

	api.Get("/select-options", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)
}

func (h Handler) selectOptions(c *fiber.Ctx) error {
	_, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	options, err := h.os.SelectOptions(c.UserContext())
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"results": options})
}
