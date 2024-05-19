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

type ShipmentControlHandler struct {
	Service           *services.ShipmentControlService
	PermissionService *services.PermissionService
}

func NewShipmentControlHandler(s *api.Server) *ShipmentControlHandler {
	return &ShipmentControlHandler{
		Service:           services.NewShipmentControlService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the ShipmentControlHandler.
func (h *ShipmentControlHandler) RegisterRoutes(r fiber.Router) {
	shipmentControlAPI := r.Group("/shipment-control")
	shipmentControlAPI.Get("/", h.GetShipmentControl())
	shipmentControlAPI.Put("/:shipmentControlID", h.UpdateShipmentControlByID())
}

// GetShipmentControl is a handler that returns the shipment control for an organization.
//
// GET /shipment-control
func (h *ShipmentControlHandler) GetShipmentControl() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !buOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID or Business Unit ID not found in the request context",
						Attr:   "orgID, buID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "shipmentcontrol.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		entity, err := h.Service.GetShipmentControl(c.UserContext(), orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

// UpdateShipmentControlByID is a handler that updates the shipment control for an organization.
//
// PUT /shipment-control/:shipmentControlID
func (h *ShipmentControlHandler) UpdateShipmentControlByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		shipmentControlID := c.Params("shipmentControlID")
		if shipmentControlID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Shipment Control ID is required",
						Attr:   "shipmentControlID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "shipmentcontrol.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		data := new(ent.ShipmentControl)

		if err := util.ParseBodyAndValidate(c, data); err != nil {
			return err
		}

		data.ID = uuid.MustParse(shipmentControlID)

		updatedEntity, err := h.Service.UpdateShipmentControl(c.UserContext(), data)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedEntity)
	}
}
