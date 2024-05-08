package handlers

import (
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/gofiber/fiber/v2"
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
	analyticAPI.Get("/daily-shipment-count", h.GetDailyShipmentCounts())
}

func (h *AnalyticHandler) GetDailyShipmentCounts() fiber.Handler {
	return func(c *fiber.Ctx) error {
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

		results, err := h.Service.GetDailyShipmentCounts(c.Context(), startDate, endDate)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error getting new shipment count",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"results": results,
		})
	}
}
