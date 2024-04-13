package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/gofiber/fiber/v2"
)

// AttachAllRoutes attaches all the routes to the Fiber instance.
func AttachAllRoutes(s *api.Server, api fiber.Router) {
	// Accessorial Charge Routers
	accessorialCharges := api.Group("/accessorial-charges")
	accessorialCharges.Get("/", GetAccessorialCharges(s))
	accessorialCharges.Post("/", CreateAccessorialCharge(s))

	// Register the handlers for the organization
	organization := api.Group("/organization")
	organization.Get("/me", GetUserOrganization(s))
}
