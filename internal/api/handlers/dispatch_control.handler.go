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

type DispatchControlHandler struct {
	Server  *api.Server
	Service *services.DispatchControlService
}

func NewDispatchControlHandler(s *api.Server) *DispatchControlHandler {
	return &DispatchControlHandler{
		Server:  s,
		Service: services.NewDispatchControlService(s),
	}
}

// GetDispatchControl is a handler that returns the dispatch control for an organization.
//
// GET /dispatch-control
func (h *DispatchControlHandler) GetDispatchControl() fiber.Handler {
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

		entity, err := h.Service.GetDispatchControl(c.UserContext(), orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

// UpdateDispatchControlByID is a handler that updates the dispatch control for an organization.
//
// PUT /dispatch-control/:dispatchControlID
func (h *DispatchControlHandler) UpdateDispatchControlByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		dispatchControlID := c.Params("dispatchControlID")
		if dispatchControlID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Dispatch Control ID is required",
						Attr:   "dispatchControlID",
					},
				},
			})
		}

		data := new(ent.DispatchControl)

		if err := util.ParseBodyAndValidate(c, data); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "request body",
					},
				},
			})
		}

		data.ID = uuid.MustParse(dispatchControlID)

		updatedEntity, err := h.Service.UpdateDispatchControl(c.UserContext(), data)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(updatedEntity)
	}
}
