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
	"fmt"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserHandler struct {
	logger            *config.ServerLogger
	service           *services.UserService
	permissionService *services.PermissionService
}

func NewUserHandler(s *server.Server) *UserHandler {
	return &UserHandler{
		logger:            s.Logger,
		service:           services.NewUserService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
	}
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (cpr ChangePasswordRequest) Validate() error {
	return validation.ValidateStruct(&cpr,
		validation.Field(&cpr.OldPassword, validation.Required),
		validation.Field(&cpr.NewPassword, validation.Required),
	)
}

func (uh UserHandler) RegisterRoutes(r fiber.Router) {
	userAPI := r.Group("/users")
	userAPI.Get("/me", uh.getAuthenticatedUser())
	userAPI.Post("/upload-profile-picture", uh.uploadProfilePicture())
	userAPI.Post("/change-password", uh.changePassword())
	userAPI.Put("/:userID", uh.updateUser())
	userAPI.Post("/clear-profile-pic", uh.clearProfilePic())
}

func (uh UserHandler) getAuthenticatedUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		entity, err := uh.service.GetAuthenticatedUser(c.UserContext(), ids.UserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (uh UserHandler) uploadProfilePicture() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		pic, err := c.FormFile("profilePicture")
		if err != nil {
			uh.logger.Error().Err(err).Msg("OrganizationHandler: Failed to get profile picture")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Failed to get profile picture",
			})
		}

		user, err := uh.service.UploadProfilePicture(c.UserContext(), pic, ids.UserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(user)
	}
}

func (uh UserHandler) changePassword() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		req := new(ChangePasswordRequest)

		if err = utils.ParseBodyAndValidate(c, req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		if err = uh.service.ChangePassword(c.UserContext(), ids.UserID, req.OldPassword, req.NewPassword); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ProblemDetail{
				Type:     "invalid",
				Title:    "Invalid Request",
				Status:   fiber.StatusBadRequest,
				Detail:   "Old password is incorrect",
				Instance: fmt.Sprintf("%s/probs/validation-error", c.BaseURL()),
				InvalidParams: []types.InvalidParam{
					{
						Name:   "oldPassword",
						Reason: "Old password is incorrect",
					},
				},
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

func (uh UserHandler) updateUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Params("userID")
		if userID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "User id is required",
			})
		}

		if err := uh.permissionService.CheckOwnershipPermission(c, constants.EntityUser, constants.ActionUpdate, userID); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: err.Error(),
			})
		}

		updatedEntity := new(models.User)

		if err := utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		updatedEntity.ID = uuid.MustParse(userID)

		entity, err := uh.service.UpdateUser(c.UserContext(), updatedEntity)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (uh UserHandler) clearProfilePic() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}
		if err = uh.service.ClearProfilePic(c.UserContext(), ids.UserID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}
