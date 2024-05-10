package handlers

import (
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AnalyticHandler struct {
	Server  *api.Server
	Service *services.AnalyticService
}

func NewAnalyticHandler(s *api.Server) *AnalyticHandler {
	return &AnalyticHandler{
		Server:  s,
		Service: services.NewAnalyticService(s),
	}
}

func (h *AnalyticHandler) RegisterRoutes(r fiber.Router) {
	analyticAPI := r.Group("/analytics")
	analyticAPI.Get("/daily-shipment-count", h.getDailyShipmentCounts())
}

func (h *AnalyticHandler) getDailyShipmentCounts() fiber.Handler {
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

		startDateStr := c.Query("start_date")
		endDateStr := c.Query("end_date")

		// Convert the start and end dates to time.Time objects
		startDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid start date format",
			})
		}

		endDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid end date format",
			})
		}

		results, count, err := h.Service.GetDailyShipmentCounts(c.Context(), startDate, endDate, orgID, buID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error getting new shipment count",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"count":   count,
			"results": results,
		})
	}
}
