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

type UserHandler struct {
	Server            *api.Server
	Service           *services.UserService
	PermissionService *services.PermissionService
}

func NewUserHandler(s *api.Server) *UserHandler {
	return &UserHandler{
		Server:            s,
		Service:           services.NewUserService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

func (h *UserHandler) RegisterRoutes(r fiber.Router) {
	usersAPI := r.Group("/users")
	usersAPI.Get("/me", h.GetAuthenticatedUser())
	usersAPI.Post("/profile-picture", h.UploadProfilePicture())
	usersAPI.Post("/change-password", h.ChangePassword())
	usersAPI.Put("/:userID", h.UpdateUser())
}

func (h *UserHandler) GetAuthenticatedUser() fiber.Handler {
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

		user, err := h.Service.GetAuthenticatedUser(c.UserContext(), userID)
		if err != nil {
			errResp := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errResp)
		}

		return c.Status(fiber.StatusOK).JSON(user)
	}
}

// UploadProfilePicture is a handler that uploads a profile picture for the authenticated user.
//
// POST /users/profile-picture
func (h *UserHandler) UploadProfilePicture() fiber.Handler {
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

		// Handle the uploaded file
		profilePicture, err := c.FormFile("profilePicture")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to read profile picture from request")
		}

		entity, err := h.Service.UploadProfilePicture(c.UserContext(), profilePicture, userID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		// Send success response
		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h *UserHandler) ChangePassword() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			OldPassword string `json:"oldPassword" validate:"required"`
			NewPassword string `json:"newPassword" validate:"required,min=8"`
		}

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

		// Parse the request body
		if err := util.ParseBodyAndValidate(c, &req); err != nil {
			return err
		}

		err := h.Service.ChangePassword(c.UserContext(), userID, req.OldPassword, req.NewPassword)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).SendString("Password changed successfully")
	}
}

func (h *UserHandler) UpdateUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Params("userID")
		if userID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "User ID is required",
						Attr:   "userID",
					},
				},
			})
		}

		// Check if the user has the required permissions or if the user is updating their own profile
		err := h.PermissionService.CheckOwnershipPermission(c, "user.update", userID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		updatedEntity := new(ent.User)

		if err := util.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return err
		}

		updatedEntity.ID = uuid.MustParse(userID)

		entity, err := h.Service.UpdateUser(c.UserContext(), updatedEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
