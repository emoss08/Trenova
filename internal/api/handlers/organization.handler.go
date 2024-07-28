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
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

const (
	// actionChangeLogo is the permission action for changing the logo of an organization.
	actionChangeLogo = "change_logo"
)

type OrganizationHandler struct {
	logger            *config.ServerLogger
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
	orgAPI.Get("/", oh.getOrganizationDetails())
	orgAPI.Put("/", oh.updateOrganization())
	orgAPI.Post("/upload-logo", oh.uploadOrganizationLogo())
	orgAPI.Post("/clear-logo", oh.clearOrganizationLogo())
}

func (oh OrganizationHandler) getOrganizationDetails() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		if err = oh.permissionService.CheckUserPermission(c, constants.EntityOrganization, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		entity, err := oh.service.GetOrganization(c.UserContext(), ids.BusinessUnitID, ids.OrganizationID)
		if err != nil {
			oh.logger.Error().Str("organizationID", ids.OrganizationID.String()).Err(err).Msg("Error getting organization details")
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
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		if err = oh.permissionService.CheckUserPermission(c, constants.EntityOrganization, constants.ActionUpdate); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		updatedEntity := new(models.Organization)
		updatedEntity.BusinessUnitID = ids.BusinessUnitID
		updatedEntity.ID = ids.OrganizationID

		if err = utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		entity, err := oh.service.UpdateOrganization(c.UserContext(), updatedEntity)
		if err != nil {
			oh.logger.Error().Interface("entity", updatedEntity).Err(err).Msg("Failed to update Organization")
			resp := utils.CreateServiceError(c, err)

			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (oh OrganizationHandler) uploadOrganizationLogo() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		if err = oh.permissionService.CheckUserPermission(c, constants.EntityOrganization, actionChangeLogo); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		logo, err := c.FormFile("logo")
		if err != nil {
			oh.logger.Error().Err(err).Msg("OrganizationHandler: Failed to get logo file")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		// Return back the entire organization.
		entity, err := oh.service.UploadLogo(c.UserContext(), logo, ids.OrganizationID)
		if err != nil {
			oh.logger.Error().Str("organizationID", ids.OrganizationID.String()).Err(err).Msg("Failed to upload logo for organization")

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (oh OrganizationHandler) clearOrganizationLogo() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		if err = oh.permissionService.CheckUserPermission(c, constants.EntityOrganization, actionChangeLogo); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		entity, err := oh.service.ClearLogo(c.UserContext(), ids.OrganizationID)
		if err != nil {
			oh.logger.Error().Str("organizationID", ids.OrganizationID.String()).Err(err).Msg("Failed to clear logo for organization")
			resp := utils.CreateServiceError(c, err)

			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
