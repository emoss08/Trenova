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
		permissionService: services.NewPermissionService(s),
	}
}

func (oh *OrganizationHandler) RegisterRoutes(r fiber.Router) {
	orgAPI := r.Group("/organizations")
	orgAPI.Get("/me", oh.getUserOrganization())
	orgAPI.Get("/details", oh.getOrganizationDetails())
	orgAPI.Put("/:orgID", oh.updateOrganization())
	orgAPI.Post("/upload-logo", oh.uploadOrganizationLogo())
	orgAPI.Post("/clear-logo", oh.clearOrganizationLogo())
}

// getUserOrganization godoc
// @Summary Fetch the organization for the current user
// @Tags organizations
// @Accept json
// @Produce json
// @Success 200 {object} UserOrganizationResponse
// @Router /organizations/me [get]
func (oh *OrganizationHandler) getUserOrganization() fiber.Handler {
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

// getOrganizationDetails godoc
// @Summary Fetch a single organization
// @Tags organizations
// @Accept json
// @Produce json
// @Success 200 {object} OrganizationResponse
// @Router /organizations/details [get]
func (oh *OrganizationHandler) getOrganizationDetails() fiber.Handler {
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

		if err := oh.permissionService.CheckUserPermission(c, models.PermissionOrganizationView.String()); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
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

// updateOrganization godoc
// @Summary Update a single organization
// @Tags organizations
// @Accept json
// @Produce json
// @Param orgID path string true "Organization ID"
// @Param body body UpdateOrganizationRequest true "Organization object"
// @Success 200 {object} OrganizationResponse
// @Router /organizations/{orgID} [put]
func (oh *OrganizationHandler) updateOrganization() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID := c.Params("orgID")
		if orgID == "" {
			oh.logger.Error().Msg("OrganizationHandler: orgID is required")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "orgID is required",
			})
		}

		if err := oh.permissionService.CheckUserPermission(c, models.PermissionOrganizationAdd.String()); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
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

// uploadOrganizationLogo godoc
// @Summary Upload a logo for the organization
// @Tags organizations
// @Accept multipart/form-data
// @Produce json
// @Param logo formData file true "Logo file"
// @Success 204
// @Router /organizations/upload-logo [post]
func (oh *OrganizationHandler) uploadOrganizationLogo() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		if !ok {
			oh.logger.Error().Msg("OrganizationHandler: Organization ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization ID not found in context",
			})
		}

		if err := oh.permissionService.CheckUserPermission(c, models.PermissionOrganizationChangeLogo.String()); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
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

// clearOrganizationLogo godoc
// @Summary Clear the logo for the organization
// @Tags organizations
// @Success 204
// @Router /organizations/clear-logo [post]
func (oh *OrganizationHandler) clearOrganizationLogo() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		if !ok {
			oh.logger.Error().Msg("OrganizationHandler: Organization ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization ID not found in context",
			})
		}

		if err := oh.permissionService.CheckUserPermission(c, models.PermissionOrganizationChangeLogo.String()); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
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
