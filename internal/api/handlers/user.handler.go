package handlers

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required"`
}

func (cpr ChangePasswordRequest) Validate() error {
	return validation.ValidateStruct(&cpr,
		validation.Field(&cpr.OldPassword, validation.Required),
		validation.Field(&cpr.NewPassword, validation.Required),
	)
}

type UserHandler struct {
	logger            *zerolog.Logger
	service           *services.UserService
	permissionService *services.PermissionService
}

func NewUserHandler(s *server.Server) *UserHandler {
	return &UserHandler{
		logger:            s.Logger,
		service:           services.NewUserService(s),
		permissionService: services.NewPermissionService(s),
	}
}

func (uh *UserHandler) RegisterRoutes(r fiber.Router) {
	userAPI := r.Group("/users")
	userAPI.Get("/me", uh.getAuthenticatedUser())
	userAPI.Post("/upload-profile-picture", uh.uploadProfilePicture())
	userAPI.Post("/change-password", uh.changePassword())
	userAPI.Put("/:userID", uh.updateUser())
	userAPI.Post("/clear-profile-pic", uh.clearProfilePic())
}

func (uh *UserHandler) getAuthenticatedUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals(utils.CTXUserID).(uuid.UUID)

		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "User id not found in context",
			})
		}

		entity, err := uh.service.GetAuthenticatedUser(c.UserContext(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (uh *UserHandler) uploadProfilePicture() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals(utils.CTXUserID).(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "User id not found in context",
			})
		}

		pic, err := c.FormFile("profilePicture")
		if err != nil {
			uh.logger.Error().Err(err).Msg("OrganizationHandler: Failed to get profile picture")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Failed to get profile picture",
			})
		}

		user, err := uh.service.UploadProfilePicture(c.UserContext(), pic, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(user)
	}
}

func (uh *UserHandler) changePassword() fiber.Handler {
	return func(c *fiber.Ctx) error {
		req := new(ChangePasswordRequest)

		userID, ok := c.Locals(utils.CTXUserID).(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "User id not found in context",
			})
		}

		if err := utils.ParseBodyAndValidate(c, req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		if err := uh.service.ChangePassword(c.UserContext(), userID, req.OldPassword, req.NewPassword); err != nil {
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

func (uh *UserHandler) updateUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Params("userID")
		if userID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "User id is required",
			})
		}

		if err := uh.permissionService.CheckOwnershipPermission(c, models.PermissionUserAdd.String(), userID); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "You do not have permission to perform this action.",
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

func (uh *UserHandler) clearProfilePic() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals(utils.CTXUserID).(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "User id not found in context",
			})
		}

		if err := uh.service.ClearProfilePic(c.UserContext(), userID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}
