package handlers

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	ptypes "github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type ShipmentHandler struct {
	logger            *zerolog.Logger
	service           *services.ShipmentService
	permissionService *services.PermissionService
}

func NewShipmentHandler(s *server.Server) *ShipmentHandler {
	return &ShipmentHandler{
		logger:            s.Logger,
		service:           services.NewShipmentService(s),
		permissionService: services.NewPermissionService(s.Enforcer),
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
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)
		if !ok || !orgOK {
			h.logger.Error().Msg("ShipmentHandler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
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
						Name:   "limit",
						Reason: "Limit must be a positive integer",
					},
					{
						Name:   "offset",
						Reason: "Offset must be a positive integer",
					},
				},
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "shipment", "view"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		filter := &services.ShipmentQueryFilter{
			Query:          c.Query("search", ""),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Limit:          limit,
			Offset:         offset,
		}

		// Parse and set additional filters
		if customerID := c.Query("customerId"); customerID != "" {
			if id, err := uuid.Parse(customerID); err == nil {
				filter.CustomerID = id
			}
		}

		if fromDate := c.Query("fromDate"); fromDate != "" {
			if date, err := time.Parse(time.RFC3339, fromDate); err == nil {
				filter.FromDate = &date
			}
		}

		if toDate := c.Query("toDate"); toDate != "" {
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
			if isHazardous == "true" {
				filter.IsHazardous = true
			} else {
				filter.IsHazardous = false
			}
		}

		entities, cnt, err := h.service.GetAll(c.UserContext(), filter)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
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

func (h ShipmentHandler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		createdEntity := new(ptypes.CreateShipmentInput)

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "shipment", "create"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		createdEntity.BusinessUnitID = buID
		createdEntity.OrganizationID = orgID

		if err := utils.ParseBodyAndValidate(c, createdEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		entity, err := h.service.Create(c.UserContext(), createdEntity)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

func (h ShipmentHandler) GetByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		shipmentID := c.Params("shipmentID")
		if shipmentID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "Shipment ID is required",
			})
		}

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("ShipmentHandler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "shipment", "view"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		entity, err := h.service.Get(c.UserContext(), uuid.MustParse(shipmentID), orgID, buID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h ShipmentHandler) AssignTractorToShipment() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)
		if !ok || !orgOK {
			h.logger.Error().Msg("ShipmentHandler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, "shipment", "update"); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: err.Error(),
			})
		}

		assignTractorInput := new(ptypes.AssignTractorInput)
		if err := utils.ParseBodyAndValidate(c, assignTractorInput); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		if err := h.service.AssignTractorToShipment(c.UserContext(), assignTractorInput, orgID, buID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Message: err.Error(),
				Code:    fiber.StatusBadRequest,
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Tractor assigned to shipment successfully.",
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
// 				Message: "You do not have permission to perform this action.",
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
