package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserFavoriteHandler struct {
	Server  *api.Server
	Service *services.UserFavoriteService
}

func NewUserFavoriteHandler(s *api.Server) *UserFavoriteHandler {
	return &UserFavoriteHandler{
		Server:  s,
		Service: services.NewUserFavoriteService(s),
	}
}

// GetUserFavorites returns the user's favorite pages.
func (h *UserFavoriteHandler) GetUserFavorites() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals(util.CTXUserID).(uuid.UUID)

		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "User ID not found in the request context",
						Attr:   "userID",
					},
				},
			})
		}

		favorites, count, err := h.Service.GetUserFavorites(c.UserContext(), userID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results: favorites,
			Count:   count,
		})
	}
}

// AddUserFavorite adds a page to the user's favorites.
func (h *UserFavoriteHandler) AddUserFavorite() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userFav := new(ent.UserFavorite)

		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !buOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID or Business Unit ID not found in the request context",
						Attr:   "orgID, buID",
					},
				},
			})
		}

		if err := util.ParseBodyAndValidate(c, userFav); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "pageLink",
					},
				},
			})
		}

		userFav.OrganizationID = orgID
		userFav.BusinessUnitID = buID

		createdEntity, err := h.Service.AddUserFavorite(c.UserContext(), userFav)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusCreated).JSON(createdEntity)
	}
}

// RemoveUserFavorite deletes a user favorite.
func (h *UserFavoriteHandler) RemoveUserFavorite() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userFav := new(ent.UserFavorite)

		if err := util.ParseBodyAndValidate(c, userFav); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "body",
					},
				},
			})
		}

		if err := h.Service.RemoveUserFavorite(c.UserContext(), userFav.UserID, userFav.PageLink); err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusNoContent).JSON(nil)
	}
}
