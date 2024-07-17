// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
