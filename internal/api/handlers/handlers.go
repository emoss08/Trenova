// CODE GENERATED. DO NOT EDIT!

package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/gofiber/fiber/v2"
)

// AttachAllRoutes attaches all the routes to the Fiber instance.
func AttachAllRoutes(s *api.Server, api fiber.Router) {
	// Register the handlers for the accessorial charges.
	accessorialChargesAPI := api.Group("/accessorial-charges")
	accessorialChargesAPI.Get("/", GetAccessorialCharges(s))
	accessorialChargesAPI.Post("/", CreateAccessorialCharge(s))
	accessorialChargesAPI.Put("/:accessorialChargeID", UpdateAccessorialCharge(s))

	// Register the handlers for the organization
	organizationsAPI := api.Group("/organizations")
	organizationsAPI.Get("/me", GetUserOrganization(s))

	// Register the handlers for the accounting control.
	accountingControlAPI := api.Group("/accounting-control")
	accountingControlAPI.Get("/", GetAccountingControl(s))
	accountingControlAPI.Put("/:accountingControlID", UpdateAccountingControlByID(s))

	// Register the handlers for the dispatch control
	dispatchControlAPI := api.Group("/dispatch-control")
	dispatchControlAPI.Get("/", GetDispatchControl(s))
	dispatchControlAPI.Put("/:dispatchControlID", UpdateDispatchControlByID(s))

	// Register the handlers for the shipment control.
	shipmentControlAPI := api.Group("/shipment-control")
	shipmentControlAPI.Get("/", GetShipmentControl(s))
	shipmentControlAPI.Put("/:shipmentControlID", UpdateShipmentControlByID(s))

	// Register the handlers for the billing control.
	billingControlAPI := api.Group("/billing-control")
	billingControlAPI.Get("/", GetBillingControl(s))
	billingControlAPI.Put("/:billingControlID", UpdateBillingControl(s))

	// Register the handlers for the invoice control.
	invoiceControlAPI := api.Group("/invoice-control")
	invoiceControlAPI.Get("/", GetInvoiceControl(s))
	invoiceControlAPI.Put("/:invoiceControlID", UpdateInvoiceControlByID(s))

	// Register the handlers for the route control.
	routeControlAPI := api.Group("/route-control")
	routeControlAPI.Get("/", GetRouteControl(s))
	routeControlAPI.Put("/:routeControlID", UpdateRouteControlByID(s))

	// Register the handlers for the feasibility tool control.
	feasibilityToolControlAPI := api.Group("/feasibility-tool-control")
	feasibilityToolControlAPI.Get("/", GetFeasibilityToolControl(s))
	feasibilityToolControlAPI.Put("/:feasibilityToolControlID", UpdateFeasibilityToolControl(s))

	// Register the handlers the email control.
	emailControlAPI := api.Group("/email-control")
	emailControlAPI.Get("/", GetEmailControl(s))
	emailControlAPI.Put("/:emailControlID", UpdateEmailControl(s))

	// Register the handlers for the user.
	usersAPI := api.Group("/users")
	usersAPI.Get("/me", GetAuthenticatedUser(s))

	// Register the handlers for the user favorites.
	userFavoritesAPI := api.Group("/user-favorites")
	userFavoritesAPI.Get("/", GetUserFavorites(s))
	userFavoritesAPI.Post("/", AddUserFavorite(s))
	userFavoritesAPI.Delete("/", RemoveUserFavorite(s))

	// Register the handlers for the hazardous material segregation rules.
	hazardousMaterialSegregationsAPI := api.Group("/hazardous-material-segregations")
	hazardousMaterialSegregationsAPI.Get("/", GetHazmatSegregationRules(s))
	hazardousMaterialSegregationsAPI.Post("/", CreateHazmatSegregationRule(s))
	hazardousMaterialSegregationsAPI.Put("/:hazmatSegRuleID", UpdateHazmatSegregationRules(s))

	// Register the handlers for the email profiles.
	emailProfilesAPI := api.Group("/email-profiles")
	emailProfilesAPI.Get("/", GetEmailProfiles(s))
	emailProfilesAPI.Post("/", CreateEmailProfile(s))
	emailProfilesAPI.Put("/:emailProfileID", UpdateEmailProfile(s))

	// Register the handlers for the table change alerts.
	tableChangeAlertsAPI := api.Group("/table-change-alerts")
	tableChangeAlertsAPI.Get("/", GetTableChangeAlerts(s))
	tableChangeAlertsAPI.Post("/", CreateTableChangeAlert(s))
	tableChangeAlertsAPI.Put("/:tableChangeAlertID", UpdateTableChangeAlert(s))
	tableChangeAlertsAPI.Get("/table-names", GetTableNames(s))
	tableChangeAlertsAPI.Get("/topic-names", GetTopicNames(s))

	// Register the handlers for the google api.
	googleAPI := api.Group("/google-api")
	googleAPI.Get("/", GetGoogleAPI(s))
	googleAPI.Put("/:googleAPIID", UpdateGoogleAPI(s))

	// Register the handlers for the revenue code.
	revenueCodesAPI := api.Group("/revenue-codes")
	revenueCodesAPI.Get("/", GetRevenueCodes(s))
	revenueCodesAPI.Post("/", CreateRevenueCode(s))
	revenueCodesAPI.Put("/:revenueCodeID", UpdateRevenueCode(s))

	// Register the handlers for the worker.
	workersAPI := api.Group("/workers")
	workersAPI.Get("/", GetWorkers(s))
	workersAPI.Post("/", CreateWorker(s))
	workersAPI.Put("/:workerID", UpdateWorker(s))

	// Register the handlers for the charge type.
	chargeTypesAPI := api.Group("/charge-types")
	chargeTypesAPI.Get("/", GetChargeTypes(s))
	chargeTypesAPI.Post("/", CreateChargeType(s))
	chargeTypesAPI.Put("/:chargeTypeID", UpdateChargeType(s))

	// Register the handlers for the comment type.
	commentTypesAPI := api.Group("/comment-types")
	commentTypesAPI.Get("/", GetCommentTypes(s))
	commentTypesAPI.Post("/", CreateCommentType(s))
	commentTypesAPI.Put("/:commentTypeID", UpdateCommentType(s))

	// Register the handlers for the commodity.
	commoditiesAPI := api.Group("/commodities")
	commoditiesAPI.Get("/", GetCommodities(s))
	commoditiesAPI.Post("/", CreateCommodity(s))
	commoditiesAPI.Put("/:commodityID", UpdateCommodity(s))

	// Register the handlers for the customer.
	customersAPI := api.Group("/customers")
	customersAPI.Get("/", GetCustomers(s))
	customersAPI.Post("/", CreateCustomer(s))
	customersAPI.Put("/:customerID", UpdateCustomer(s))

	// Register the handlers for the delay code.
	delayCodesAPI := api.Group("/delay-codes")
	delayCodesAPI.Get("/", GetDelayCodes(s))
	delayCodesAPI.Post("/", CreateDelayCode(s))
	delayCodesAPI.Put("/:delayCodeID", UpdateDelayCode(s))

	// Register the handlers for the division code.
	divisionCodesAPI := api.Group("/division-codes")
	divisionCodesAPI.Get("/", GetDivisionCodes(s))
	divisionCodesAPI.Post("/", CreateDivisionCode(s))
	divisionCodesAPI.Put("/:divisionCodeID", UpdateDivisionCode(s))

	// Register the handlers for the document classification.
	documentClassificationsAPI := api.Group("/document-classifications")
	documentClassificationsAPI.Get("/", GetDocumentClassifications(s))
	documentClassificationsAPI.Post("/", CreateDocumentClassification(s))
	documentClassificationsAPI.Put("/:documentClassID", UpdateDocumentClassification(s))

	// Register the handlers for the equipment manufacturer.
	equipmentManufacturersAPI := api.Group("/equipment-manufacturers")
	equipmentManufacturersAPI.Get("/", GetEquipmentManufacturers(s))
	equipmentManufacturersAPI.Post("/", CreateEquipmentManufacturer(s))
	equipmentManufacturersAPI.Put("/:equipmentManuID", UpdateEquipmentManufacturer(s))

	// Register the handlers for the equipment type.
	equipmentTypesAPI := api.Group("/equipment-types")
	equipmentTypesAPI.Get("/", GetEquipmentTypes(s))
	equipmentTypesAPI.Post("/", CreateEquipmentType(s))
	equipmentTypesAPI.Put("/:equipmentTypeID", UpdateEquipmentType(s))

	// Register the handlers for the fleet code.
	fleetCodesAPI := api.Group("/fleet-codes")
	fleetCodesAPI.Get("/", GetFleetCodes(s))
	fleetCodesAPI.Post("/", CreateFleetCode(s))
	fleetCodesAPI.Put("/:fleetCodeID", UpdateFleetCode(s))

	// Register the handlers for the general ledger account.
	generalLedgerAccountsAPI := api.Group("/general-ledger-accounts")
	generalLedgerAccountsAPI.Get("/", GetGeneralLedgerAccounts(s))
	generalLedgerAccountsAPI.Post("/", CreateGeneralLedgerAccount(s))
	generalLedgerAccountsAPI.Put("/:glAccountID", UpdateGeneralLedgerAccount(s))

	// Register the handlers for the tag.
	tagsAPI := api.Group("/tags")
	tagsAPI.Get("/", GetTags(s))
	tagsAPI.Post("/", CreateTag(s))
	tagsAPI.Put("/:tagID", UpdateTag(s))

	// Register the handlers for the hazardous material.
	hazardousMaterialsAPI := api.Group("/hazardous-materials")
	hazardousMaterialsAPI.Get("/", GetHazardousMaterials(s))
	hazardousMaterialsAPI.Post("/", CreateHazardousMaterial(s))
	hazardousMaterialsAPI.Put("/:hazmatID", UpdateHazardousMaterial(s))

	// Register the handlers for the location.
	locationsAPI := api.Group("/locations")
	locationsAPI.Get("/", GetLocations(s))
	locationsAPI.Post("/", CreateLocation(s))
	locationsAPI.Put("/:locationID", UpdateLocation(s))

	// Register the handlers for the location categories.
	locationCategoriesAPI := api.Group("/location-categories")
	locationCategoriesAPI.Get("/", GetLocationCategories(s))
	locationCategoriesAPI.Post("/", CreateLocationCategory(s))
	locationCategoriesAPI.Put("/:locationCategoryID", UpdateLocationCategory(s))

	// Register the handlers for the feature flags.
	featureFlagsAPI := api.Group("/feature-flags")
	featureFlagsAPI.Get("/", GetFeatureFlags(s))

	// Register the handlers for the qualifier codes.
	qualifierCodesAPI := api.Group("/qualifier-codes")
	qualifierCodesAPI.Get("/", GetQualifierCodes(s))
	qualifierCodesAPI.Post("/", CreateQualifierCode(s))
	qualifierCodesAPI.Put("/:qualifierCodeID", UpdateQualifierCode(s))

	// Register the handlers for the reason codes.
	reasonCodesAPI := api.Group("/reason-codes")
	reasonCodesAPI.Get("/", GetReasonCodes(s))
	reasonCodesAPI.Post("/", CreateReasonCode(s))
	reasonCodesAPI.Put("/:reasonCodeID", UpdateReasonCode(s))

	// Register the handlers for the service types.
	serviceTypesAPI := api.Group("/service-types")
	serviceTypesAPI.Get("/", GetServiceTypes(s))
	serviceTypesAPI.Post("/", CreateServiceType(s))
	serviceTypesAPI.Put("/:serviceTypeID", UpdateServiceType(s))

	// Register the handlers for the shipment types.
	shipmentTypesAPI := api.Group("/shipment-types")
	shipmentTypesAPI.Get("/", GetShipmentTypes(s))
	shipmentTypesAPI.Post("/", CreateShipmentType(s))
	shipmentTypesAPI.Put("/:shipmentTypeID", UpdateShipmentType(s))

	// Register the handlers for the us states.
	usStatesAPI := api.Group("/us-states")
	usStatesAPI.Get("/", GetUSStates(s))
}
