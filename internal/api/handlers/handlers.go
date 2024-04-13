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
	accessorialCharges.Put("/:accessorialChargeID", UpdateAccessorialCharge(s))

	// Register the handlers for the organization
	organizations := api.Group("/organizations")
	organizations.Get("/me", GetUserOrganization(s))

	// Register the handlers for the organization
	accountingControl := api.Group("/accounting-control")
	accountingControl.Get("/", GetAccountingControl(s))
	accountingControl.Put("/:accountingControlID", UpdateAccountingControlByID(s))

	// Register the handlers for the user.
	users := api.Group("/users")
	users.Get("/me", GetAuthenticatedUser(s))

	// Register the handlers for the user favorites.
	userFavorites := api.Group("/user-favorites")
	userFavorites.Get("/", GetUserFavorites(s))
	userFavorites.Post("/", AddUserFavorite(s))
	userFavorites.Delete("/", RemoveUserFavorite(s))
}
