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
	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type HazardousMaterialHandler struct {
	logger            *config.ServerLogger
	service           *services.HazardousMaterialService
	permissionService *services.PermissionService
	auditService      *audit.Service
}

func NewHazardousMaterialHandler(s *server.Server) *HazardousMaterialHandler {
	return &HazardousMaterialHandler{
		logger:            s.Logger,
		service:           services.NewHazardousMaterialService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
		auditService:      s.AuditService,
	}
}

func (h HazardousMaterialHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/hazardous-materials")
	api.Get("/", h.Get())
	api.Get("/:hazmatID", h.GetByID())
	api.Post("/", h.Create())
	api.Put("/:hazmatID", h.Update())
}

func (h HazardousMaterialHandler) Get() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		offset, limit, err := utils.PaginationParams(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ProblemDetail{
				Type:     "invalid",
				Title:    "Invalid Request",
				Status:   fiber.StatusBadRequest,
				Detail:   err.Error(),
				Instance: fmt.Sprintf("%s/probs/validation-error", c.BaseURL()),
				InvalidParams: []types.InvalidParam{
					{
						Name:   constants.FieldLimit,
						Reason: constants.ReasonMustBePositiveInteger,
					},
					{
						Name:   constants.FieldOffset,
						Reason: constants.ReasonMustBePositiveInteger,
					},
				},
			})
		}

		if err = h.permissionService.CheckUserPermission(c, constants.EntityHazardousMaterial, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		filter := &services.HazardousMaterialQueryFilter{
			Query:          c.Query("search", ""),
			OrganizationID: ids.OrganizationID,
			BusinessUnitID: ids.BusinessUnitID,
			Limit:          limit,
			Offset:         offset,
		}

		entities, cnt, err := h.service.GetAll(c.UserContext(), filter)
		if err != nil {
			h.logger.Error().Err(err).Msg("Error getting hazardous materials")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.HazardousMaterial]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}

func (h HazardousMaterialHandler) GetByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		hazmatID := c.Params("hazmatID")
		if hazmatID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "HazardousMaterial ID is required",
			})
		}

		if err = h.permissionService.CheckUserPermission(c, constants.EntityHazardousMaterial, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		entity, err := h.service.Get(c.UserContext(), uuid.MustParse(hazmatID), ids.OrganizationID, ids.BusinessUnitID)
		if err != nil {
			h.logger.Error().Str("hazmatID", hazmatID).Err(err).Msg("Error getting hazardous material by ID")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h HazardousMaterialHandler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		createdEntity := new(models.HazardousMaterial)

		if err = h.permissionService.CheckUserPermission(c, constants.EntityHazardousMaterial, constants.ActionCreate); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		createdEntity.BusinessUnitID = ids.BusinessUnitID
		createdEntity.OrganizationID = ids.OrganizationID

		if err = utils.ParseBodyAndValidate(c, createdEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		attemptID := h.auditService.LogAttempt(c.Context(), constants.TableHazardousMaterial, "", property.AuditLogActionCreate, createdEntity, ids.UserID, ids.OrganizationID, ids.BusinessUnitID)

		entity, err := h.service.Create(c.UserContext(), createdEntity)
		if err != nil {
			h.logger.Error().Interface("entity", createdEntity).Err(err).Msg("Failed to create HazardousMaterial")
			resp := utils.CreateServiceError(c, err)

			h.auditService.LogError(c.Context(), property.AuditLogActionCreate, attemptID, ids.OrganizationID, ids.BusinessUnitID, ids.UserID, err.Error())

			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		h.auditService.LogAction(c.Context(), constants.TableHazardousMaterial, entity.ID.String(), property.AuditLogActionCreate, entity, ids.UserID, ids.OrganizationID, ids.BusinessUnitID)

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

func (h HazardousMaterialHandler) Update() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		hazmatID := c.Params("hazmatID")
		if hazmatID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "HazardousMaterial ID is required",
			})
		}

		if err = h.permissionService.CheckUserPermission(c, constants.EntityHazardousMaterial, constants.ActionUpdate); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		updatedEntity := new(models.HazardousMaterial)

		if err = utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		updatedEntity.ID = uuid.MustParse(hazmatID)

		attemptID := h.auditService.LogAttempt(c.Context(), constants.TableHazardousMaterial, hazmatID, property.AuditLogActionUpdate, updatedEntity, ids.UserID, ids.OrganizationID, ids.BusinessUnitID)

		entity, err := h.service.UpdateOne(c.UserContext(), updatedEntity)
		if err != nil {
			h.logger.Error().Interface("entity", updatedEntity).Err(err).Msg("Failed to update HazardousMaterial")
			resp := utils.CreateServiceError(c, err)

			h.auditService.LogError(c.Context(), property.AuditLogActionUpdate, attemptID, ids.OrganizationID, ids.BusinessUnitID, ids.UserID, err.Error())

			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		h.auditService.LogAction(c.Context(), constants.TableHazardousMaterial, entity.ID.String(), property.AuditLogActionUpdate, entity, ids.UserID, ids.OrganizationID, ids.BusinessUnitID)

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
