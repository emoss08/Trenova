package handlers

import (
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type OrganizationHandler struct {
	logger            *zerolog.Logger
	service           *services.OrganizationService
	permissionService *services.PermissionService
}

func NewOrganizationHandler(s *server.Server) *OrganizationHandler {
	return &OrganizationHandler{
		logger:            s.Logger,
		service:           services.NewOrganizationService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
	}
}

func (oh OrganizationHandler) RegisterRoutes(r fiber.Router) {
	orgAPI := r.Group("/organizations")
	orgAPI.Get("/me", oh.getUserOrganization())
	orgAPI.Get("/details", oh.getOrganizationDetails())
	orgAPI.Put("/:orgID", oh.updateOrganization())
	orgAPI.Post("/upload-logo", oh.uploadOrganizationLogo())
	orgAPI.Post("/clear-logo", oh.clearOrganizationLogo())
}

func (oh OrganizationHandler) getUserOrganization() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			oh.logger.Error().Msg("OrganizationHandler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		entity, err := oh.service.GetUserOrganization(c.UserContext(), buID, orgID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(err)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (oh OrganizationHandler) getOrganizationDetails() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			oh.logger.Error().Msg("OrganizationHandler: Organization ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization ID not found in context",
			})
		}

		if err := oh.permissionService.CheckUserPermission(c, "organization", "view"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		entity, err := oh.service.GetOrganization(c.UserContext(), buID, orgID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (oh OrganizationHandler) updateOrganization() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID := c.Params("orgID")
		if orgID == "" {
			oh.logger.Error().Msg("OrganizationHandler: orgID is required")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "orgID is required",
			})
		}

		if err := oh.permissionService.CheckUserPermission(c, "organization", "update"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		updatedEntity := new(models.Organization)

		if err := utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		entity, err := oh.service.UpdateOrganization(c.UserContext(), updatedEntity)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (oh OrganizationHandler) uploadOrganizationLogo() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		if !ok {
			oh.logger.Error().Msg("OrganizationHandler: Organization ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization ID not found in context",
			})
		}

		if err := oh.permissionService.CheckUserPermission(c, "organization", "change_logo"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		logo, err := c.FormFile("logo")
		if err != nil {
			oh.logger.Error().Err(err).Msg("OrganizationHandler: Failed to get logo file")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get logo file",
			})
		}

		if err = oh.service.UploadLogo(c.UserContext(), logo, orgID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

func (oh OrganizationHandler) clearOrganizationLogo() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		if !ok {
			oh.logger.Error().Msg("OrganizationHandler: Organization ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization ID not found in context",
			})
		}

		if err := oh.permissionService.CheckUserPermission(c, "organization", "change_logo"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		if err := oh.service.ClearLogo(c.UserContext(), orgID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}
