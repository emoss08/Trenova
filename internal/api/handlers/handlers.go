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

	// Register the handlers for the accounting control.
	accountingControl := api.Group("/accounting-control")
	accountingControl.Get("/", GetAccountingControl(s))
	accountingControl.Put("/:accountingControlID", UpdateAccountingControlByID(s))

	// Register the handlers for the dispatch control
	dispatchControl := api.Group("/dispatch-control")
	dispatchControl.Get("/", GetDispatchControl(s))
	dispatchControl.Put("/:dispatchControlID", UpdateDispatchControlByID(s))

	// Register the handlers for the shipment control.
	shipmentControl := api.Group("/shipment-control")
	shipmentControl.Get("/", GetShipmentControl(s))
	shipmentControl.Put("/:shipmentControlID", UpdateShipmentControlByID(s))

	// Register the handlers for the billing control.
	billingControl := api.Group("/billing-control")
	billingControl.Get("/", GetBillingControl(s))
	billingControl.Put("/:billingControlID", UpdateBillingControl(s))

	// Register the handlers for the invoice control.
	invoiceControl := api.Group("/invoice-control")
	invoiceControl.Get("/", GetInvoiceControl(s))
	invoiceControl.Put("/:invoiceControlID", UpdateInvoiceControlByID(s))

	// Register the handlers for the route control.
	routeControl := api.Group("/route-control")
	routeControl.Get("/", GetRouteControl(s))
	routeControl.Put("/:routeControlID", UpdateRouteControlByID(s))

	// Register the handlers for the feasibility tool control.
	feasibilityToolControl := api.Group("/feasibility-tool-control")
	feasibilityToolControl.Get("/", GetFeasibilityToolControl(s))
	feasibilityToolControl.Put("/:feasibilityToolControlID", UpdateFeasibilityToolControl(s))

	// Register the handlers the email control.
	emailControl := api.Group("/email-control")
	emailControl.Get("/", GetEmailControl(s))
	emailControl.Put("/:emailControlID", UpdateEmailControl(s))

	// Register the handlers for the user.
	users := api.Group("/users")
	users.Get("/me", GetAuthenticatedUser(s))

	// Register the handlers for the user favorites.
	userFavorites := api.Group("/user-favorites")
	userFavorites.Get("/", GetUserFavorites(s))
	userFavorites.Post("/", AddUserFavorite(s))
	userFavorites.Delete("/", RemoveUserFavorite(s))

	// Register the handlers for the hazardous material segregation rules.
	hazardousMaterialSegregations := api.Group("/hazardous-material-segregations")
	hazardousMaterialSegregations.Get("/", GetHazmatSegregationRules(s))
	hazardousMaterialSegregations.Post("/", CreateHazmatSegregationRule(s))
	hazardousMaterialSegregations.Put("/:hazmatSegRuleID", UpdateHazmatSegregationRules(s))
}
