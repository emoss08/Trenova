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
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type UserFavoriteHandler struct {
	logger  *zerolog.Logger
	service *services.UserFavoriteService
}

func NewUserFavoriteHandler(s *server.Server) *UserFavoriteHandler {
	return &UserFavoriteHandler{
		logger:  s.Logger,
		service: services.NewUserFavoriteService(s),
	}
}

func (ufh UserFavoriteHandler) RegisterRoutes(r fiber.Router) {
	ufAPI := r.Group("/user-favorites")
	ufAPI.Get("", ufh.getUserFavorites())
	ufAPI.Post("", ufh.addUserFavorite())
	ufAPI.Delete("", ufh.deleteUserFavorite())
}

func (ufh UserFavoriteHandler) getUserFavorites() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals(utils.CTXUserID).(uuid.UUID)

		if !ok {
			ufh.logger.Error().Msg("UserFavoriteHandler: User ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "User ID not found in context",
			})
		}

		entities, cnt, err := ufh.service.GetUserFavorites(c.UserContext(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.UserFavorite]{
			Results: entities,
			Count:   cnt,
			Next:    "",
			Prev:    "",
		})
	}
}

func (ufh UserFavoriteHandler) addUserFavorite() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userFav := new(models.UserFavorite)

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !buOK {
			ufh.logger.Error().Msg("UserFavoriteHandler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := utils.ParseBodyAndValidate(c, userFav); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		userFav.OrganizationID = orgID
		userFav.BusinessUnitID = buID

		if err := ufh.service.AddUserFavorite(c.UserContext(), userFav); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusCreated)
	}
}

func (ufh UserFavoriteHandler) deleteUserFavorite() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userFav := new(models.UserFavorite)

		if err := utils.ParseBodyAndValidate(c, userFav); err != nil {
			return err
		}

		if err := ufh.service.DeleteUserFavorite(c.UserContext(), userFav); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}
