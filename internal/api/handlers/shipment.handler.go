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
	"time"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	ptypes "github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ShipmentHandler struct {
	logger            *config.ServerLogger
	service           *services.ShipmentService
	permissionService *services.PermissionService
	auditService      *audit.Service
}

func NewShipmentHandler(s *server.Server) *ShipmentHandler {
	return &ShipmentHandler{
		logger:            s.Logger,
		service:           services.NewShipmentService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
		auditService:      s.AuditService,
	}
}

func (h ShipmentHandler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/shipments")
	api.Get("/", h.Get())
	api.Post("/assign-tractor", h.AssignTractorToShipment())
	api.Get("/:shipmentID", h.GetByID())
	api.Post("/", h.Create())
	// api.Put("/:shipmentID", h.Update())
}

func (h ShipmentHandler) Get() fiber.Handler {
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

		if err = h.permissionService.CheckUserPermission(c, constants.EntityShipment, constants.ActionView); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		filter := &services.ShipmentQueryFilter{
			Query:          c.Query("search", ""),
			OrganizationID: ids.OrganizationID,
			BusinessUnitID: ids.BusinessUnitID,
			Limit:          limit,
			Offset:         offset,
		}

		// Parse and set additional filters
		if customerID := c.Query("customerId"); customerID != "" {
			if id, err := uuid.Parse(customerID); err == nil {
				filter.CustomerID = id
			}
		}

		if fromDate := c.Query(constants.FieldFromDate); fromDate != "" {
			if date, err := time.Parse(time.RFC3339, fromDate); err == nil {
				filter.FromDate = &date
			}
		}

		if toDate := c.Query(constants.FieldToDate); toDate != "" {
			if date, err := time.Parse(time.RFC3339, toDate); err == nil {
				filter.ToDate = &date
			}
		}

		if shipmentTypeID := c.Query("shipmentTypeId"); shipmentTypeID != "" {
			if id, err := uuid.Parse(shipmentTypeID); err == nil {
				filter.ShipmentTypeID = id
			}
		}

		// Parse status filter
		if status := c.Query("status"); status != "" {
			filter.Status = property.ShipmentStatus(status)
		}

		// Parse isHazardous filter
		if isHazardous := c.Query("isHazardous"); isHazardous != "" {
			filter.IsHazardous = isHazardous == constants.QueryParamTrue
		}

		entities, cnt, err := h.service.GetAll(c.UserContext(), filter)
		if err != nil {
			h.logger.Error().Err(err).Msg("Error getting shipments")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: constants.ErrInternalServer,
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.Shipment]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}

func (h ShipmentHandler) GetByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		shipmentID := c.Params("shipmentID")
		if shipmentID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Shipment ID is required",
			})
		}

		if err = h.permissionService.CheckUserPermission(c, "shipment", "view"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		entity, err := h.service.Get(c.UserContext(), uuid.MustParse(shipmentID), ids.OrganizationID, ids.BusinessUnitID)
		if err != nil {
			h.logger.Error().Str("shipmentID", shipmentID).Err(err).Msg("Error getting shipment by ID")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h ShipmentHandler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		createdEntity := new(ptypes.CreateShipmentInput)

		if err = h.permissionService.CheckUserPermission(c, "shipment", "create"); err != nil {
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

		entity, err := h.service.Create(c.UserContext(), createdEntity)
		if err != nil {
			h.logger.Error().Interface("entity", createdEntity).Err(err).Msg("Failed to create Shipment")
			resp := utils.CreateServiceError(c, err)
			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		go h.auditService.LogAction("shipments", entity.ID.String(), property.AuditLogActionCreate, entity, ids.UserID, ids.OrganizationID, ids.BusinessUnitID)

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

func (h ShipmentHandler) AssignTractorToShipment() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := utils.ExtractAndHandleContextIDs(c)
		if err != nil {
			return err
		}

		if err = h.permissionService.CheckUserPermission(c, "shipment", "assign_tractor"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		assignTractorInput := new(ptypes.AssignTractorInput)
		if err = utils.ParseBodyAndValidate(c, assignTractorInput); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		assignments, err := h.service.AssignTractorToShipment(c.UserContext(), assignTractorInput, ids.OrganizationID, ids.BusinessUnitID)
		if err != nil {
			h.logger.Error().Interface("entity", assignTractorInput).Err(err).Msg("Failed to assign tractor to shipment")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Message: err.Error(),
				Code:    fiber.StatusBadRequest,
			})
		}

		go h.auditService.LogAction("shipments", assignTractorInput.TractorID.String(), property.AuditLogActionUpdate, assignments, ids.UserID, ids.OrganizationID, ids.BusinessUnitID)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Tractor assigned to shipment successfully.",
			"data":    assignments,
		})
	}
}

// func (h *ShipmentHandler) Update() fiber.Handler {
// 	return func(c *fiber.Ctx) error {
// 		shipmentID := c.Params("shipmentID")
// 		if shipmentID == "" {
// 			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
// 				Code:    fiber.StatusBadRequest,
// 				Message: "DelayCode ID is required",
// 			})
// 		}

// 		if err := h.permissionService.CheckUserPermission(c, models.PermissionShipmentEdit.String()); err != nil {
// 			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
// 				Code:    fiber.StatusUnauthorized,
// 				Message: err.Error(),
// 			})
// 		}

// 		updatedEntity := new(models.Shipment)

// 		if err := utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
// 			return c.Status(fiber.StatusBadRequest).JSON(err)
// 		}

// 		updatedEntity.ID = uuid.MustParse(shipmentID)

// 		entity, err := h.service.UpdateOne(c.UserContext(), updatedEntity)
// 		if err != nil {
// 			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
// 				Code:    fiber.StatusInternalServerError,
// 				Message: "Failed to update DelayCode",
// 			})
// 		}

// 		return c.Status(fiber.StatusOK).JSON(entity)
// 	}
// }
