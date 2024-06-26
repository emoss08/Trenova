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

type EquipmentManufacturerHandler struct {
	Service           *services.EquipmentManufacturerService
	PermissionService *services.PermissionService
}

func NewEquipmentManufacturerHandler(s *api.Server) *EquipmentManufacturerHandler {
	return &EquipmentManufacturerHandler{
		Service:           services.NewEquipmentManufacturerService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the EquipmentManufacturerHandler.
func (h *EquipmentManufacturerHandler) RegisterRoutes(r fiber.Router) {
	equipmentManufacturerAPI := r.Group("/equipment-manufacturers")
	equipmentManufacturerAPI.Get("/", h.GetEquipmentManufacturers())
	equipmentManufacturerAPI.Post("/", h.CreateEquipmentManufacturer())
	equipmentManufacturerAPI.Put("/:equipmentManuID", h.UpdateEquipmentManufacturer())
}

// GetEquipmentManufacturers is a handler that returns a list of equipment manufacturers.
//
// GET /equipment-manufacturers
func (h *EquipmentManufacturerHandler) GetEquipmentManufacturers() fiber.Handler {
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
		err := h.PermissionService.CheckUserPermission(c, "equipmentmanufacturer.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		offset, limit, err := util.PaginationParams(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "offset, limit",
					},
				},
			})
		}

		entities, count, err := h.Service.GetEquipmentManufacturers(c.UserContext(), limit, offset, orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		nextURL := util.GetNextPageURL(c, limit, offset, count)
		prevURL := util.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results:  entities,
			Count:    count,
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

// CreateEquipmentManufacturer is a handler that creates a new equipment manufacturer.
//
// POST /equipment-manufacturers
func (h *EquipmentManufacturerHandler) CreateEquipmentManufacturer() fiber.Handler {
	return func(c *fiber.Ctx) error {
		newEntity := new(ent.EquipmentManufactuer)

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
		err := h.PermissionService.CheckUserPermission(c, "equipmentmanufacturer.add")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		newEntity.BusinessUnitID = buID
		newEntity.OrganizationID = orgID

		if err := util.ParseBodyAndValidate(c, newEntity); err != nil {
			return err
		}

		entity, err := h.Service.CreateEquipmentManufacturer(c.UserContext(), newEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

// UpdateEquipmentManufacturer is a handler that updates an equipment manufacturer.
//
// PUT /equipment-manufacturers/:equipmentManuID
func (h *EquipmentManufacturerHandler) UpdateEquipmentManufacturer() fiber.Handler {
	return func(c *fiber.Ctx) error {
		equipmentManuID := c.Params("equipmentManuID")
		if equipmentManuID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Equipment Manufacturer ID is required",
						Attr:   "equipmentManuID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "equipmentmanufacturer.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		updatedEntity := new(ent.EquipmentManufactuer)

		if err := util.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return err
		}

		updatedEntity.ID = uuid.MustParse(equipmentManuID)

		entity, err := h.Service.UpdateEquipmentManufacturer(c.UserContext(), updatedEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
