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
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/models/property"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type TractorHandler struct {
	logger            *config.ServerLogger
	service           *services.TractorService
	permissionService *services.PermissionService
}

func NewTractorHandler(s *server.Server) *TractorHandler {
	return &TractorHandler{
		logger:            s.Logger,
		service:           services.NewTractorService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
	}
}

func (h TractorHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/tractors")
	api.Get("/", h.Get())
	api.Get("/:tractorID", h.GetByID())
	api.Get("/:tractorID/assignments", h.GetActiveAssignments())
	api.Post("/", h.Create())
	api.Put("/:tractorID", h.Update())
}

func (h TractorHandler) Get() fiber.Handler {
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

		if err = h.permissionService.CheckUserPermission(c, constants.EntityTractor, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		filter := &services.TractorQueryFilter{
			Query:          c.Query("search", ""),
			OrganizationID: ids.OrganizationID,
			BusinessUnitID: ids.BusinessUnitID,
			Limit:          limit,
			Offset:         offset,
		}

		// Parse the status filter
		if status := c.Query("status"); status != "" {
			filter.Status = status
		}

		// Parse the fleet code ID filter
		if fleetCodeID := c.Query("fleetCodeId"); fleetCodeID != "" {
			if id, uErr := uuid.Parse(fleetCodeID); uErr == nil {
				filter.FleetCodeID = id
			}
		}

		// Parse the expand equipment details filter
		if expandEquipDetails := c.Query("expandEquipDetails"); expandEquipDetails != "" {
			filter.ExpandEquipDetails = expandEquipDetails == constants.QueryParamTrue
		}

		// Parse the expand worker details filter
		if expandWorkerDetails := c.Query("expandWorkerDetails"); expandWorkerDetails != "" {
			filter.ExpandWorkerDetails = expandWorkerDetails == constants.QueryParamTrue
		}

		entities, cnt, err := h.service.GetAll(c.UserContext(), filter)
		if err != nil {
			h.logger.Error().Err(err).Msg("Error getting tractors")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.Tractor]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}

func (h TractorHandler) GetByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		tractorID := c.Params("tractorID")
		if tractorID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Tractor ID is required",
			})
		}

		if err = h.permissionService.CheckUserPermission(c, constants.EntityTractor, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		entity, err := h.service.Get(c.UserContext(), uuid.MustParse(tractorID), ids.OrganizationID, ids.BusinessUnitID)
		if err != nil {
			h.logger.Error().Str("tractorID", tractorID).Err(err).Msg("Error getting tractor by ID")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h TractorHandler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		createdEntity := new(models.Tractor)

		if err = h.permissionService.CheckUserPermission(c, constants.EntityTractor, constants.ActionCreate); err != nil {
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
			h.logger.Error().Interface("entity", createdEntity).Err(err).Msg("Failed to create Tractor")
			resp := utils.CreateServiceError(c, err)

			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

func (h TractorHandler) Update() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		tractorID := c.Params("tractorID")
		if tractorID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Tractor ID is required",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, constants.EntityTractor, constants.ActionUpdate); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		updatedEntity := new(models.Tractor)

		if err := utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		updatedEntity.ID = uuid.MustParse(tractorID)

		entity, err := h.service.UpdateOne(c.UserContext(), updatedEntity, ids.UserID)
		if err != nil {
			h.logger.Error().Interface("entity", updatedEntity).Err(err).Msg("Failed to update Tractor")
			resp := utils.CreateServiceError(c, err)

			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h TractorHandler) GetActiveAssignments() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		tractorID := c.Params("tractorID")
		if tractorID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Tractor ID is required",
			})
		}

		filter := &services.AssignmentQueryFilter{
			OrganizationID: ids.OrganizationID,
			BusinessUnitID: ids.BusinessUnitID,
			TractorID:      uuid.MustParse(tractorID),
		}

		// Pase and set addtional filters
		if status := c.Query("status"); status != "" {
			filter.Status = property.AssignmentStatus(status)
		}

		if expandShipmentDetails := c.Query("expandShipmentDetails"); expandShipmentDetails != "" {
			filter.ExpandShipmentDetails = expandShipmentDetails == constants.QueryParamTrue
		}

		assignments, err := h.service.GetActiveAssignments(c.UserContext(), filter)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.JSON(assignments)
	}
}
