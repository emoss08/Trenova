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
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ShipmentTypeHandler struct {
	logger            *config.ServerLogger
	service           *services.ShipmentTypeService
	permissionService *services.PermissionService
}

func NewShipmentTypeHandler(s *server.Server) *ShipmentTypeHandler {
	return &ShipmentTypeHandler{
		logger:            s.Logger,
		service:           services.NewShipmentTypeService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
	}
}

func (h ShipmentTypeHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/shipment-types")
	api.Get("/", h.Get())
	api.Get("/:shipmentTypeID", h.GetByID())
	api.Post("/", h.Create())
	api.Put("/:shipmentTypeID", h.Update())
}

func (h ShipmentTypeHandler) Get() fiber.Handler {
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

		if err = h.permissionService.CheckUserPermission(c, constants.EntityShipmentType, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		filter := &services.ShipmentTypeQueryFilter{
			Query:          c.Query("search", ""),
			OrganizationID: ids.OrganizationID,
			BusinessUnitID: ids.BusinessUnitID,
			Limit:          limit,
			Offset:         offset,
		}

		entities, cnt, err := h.service.GetAll(c.UserContext(), filter)
		if err != nil {
			h.logger.Error().Err(err).Msg("Error getting shipment types")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.ShipmentType]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}

func (h ShipmentTypeHandler) GetByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		shipmentTypeID := c.Params("shipmentTypeID")
		if shipmentTypeID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "ShipmentType ID is required",
			})
		}

		if err = h.permissionService.CheckUserPermission(c, constants.EntityShipmentType, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		entity, err := h.service.Get(c.UserContext(), uuid.MustParse(shipmentTypeID), ids.OrganizationID, ids.BusinessUnitID)
		if err != nil {
			h.logger.Error().Str("shipmentTypeID", shipmentTypeID).Err(err).Msg("Error getting shipment type by ID")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h ShipmentTypeHandler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		createdEntity := new(models.ShipmentType)

		if err = h.permissionService.CheckUserPermission(c, constants.EntityShipmentType, constants.ActionCreate); err != nil {
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

		entity, err := h.service.Create(c.UserContext(), createdEntity, ids.UserID)
		if err != nil {
			h.logger.Error().Interface("entity", createdEntity).Err(err).Msg("Failed to create ShipmentType")
			resp := utils.CreateServiceError(c, err)

			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

func (h ShipmentTypeHandler) Update() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		shipmentTypeID := c.Params("shipmentTypeID")
		if shipmentTypeID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "ShipmentType ID is required",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, constants.EntityShipmentType, constants.ActionUpdate); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		updatedEntity := new(models.ShipmentType)

		if err := utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		updatedEntity.ID = uuid.MustParse(shipmentTypeID)

		entity, err := h.service.UpdateOne(c.UserContext(), updatedEntity, ids.UserID)
		if err != nil {
			h.logger.Error().Interface("entity", updatedEntity).Err(err).Msg("Failed to update ShipmentType")
			resp := utils.CreateServiceError(c, err)

			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
