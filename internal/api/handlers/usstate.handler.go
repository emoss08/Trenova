package handlers

import (
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

type USStateHandler struct {
	logger  *zerolog.Logger
	service *services.USStateService
}

func NewUSStateHandler(s *server.Server) *USStateHandler {
	return &USStateHandler{
		logger:  s.Logger,
		service: services.NewUSStateService(s),
	}
}

func (h *USStateHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/us-states")
	api.Get("/", h.Get())
}

func (h *USStateHandler) Get() fiber.Handler {
	return func(c *fiber.Ctx) error {
		entities, cnt, err := h.service.GetUSStates(c.UserContext())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get trailers",
			})
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.UsState]{
			Results: entities,
			Count:   cnt,
		})
	}
}
