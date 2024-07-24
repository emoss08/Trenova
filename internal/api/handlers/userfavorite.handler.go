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
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserFavoriteHandler struct {
	logger  *config.ServerLogger
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
