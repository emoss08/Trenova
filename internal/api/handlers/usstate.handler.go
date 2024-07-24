// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package handlers

import (
	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/gofiber/fiber/v2"
)

type USStateHandler struct {
	logger  *config.ServerLogger
	service *services.USStateService
}

func NewUSStateHandler(s *server.Server) *USStateHandler {
	return &USStateHandler{
		logger:  s.Logger,
		service: services.NewUSStateService(s),
	}
}

func (h USStateHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/us-states")
	api.Get("/", h.Get())
}

func (h USStateHandler) Get() fiber.Handler {
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
