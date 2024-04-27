package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/gofiber/fiber/v2"
)

// AttachAllRoutes attaches all the routes to the Fiber instance.
func AttachAllRoutes(s *api.Server, api fiber.Router) { //nolint:funlen // This function is responsible for setting up all the routes.
	// Register the handlers for the accessorial charges.
	accessorialChargesAPI := api.Group("/accessorial-charges")
	accessorialChargesAPI.Get("/", NewAccessorialChargeHandler(s).GetAccessorialCharges())
	accessorialChargesAPI.Post("/", NewAccessorialChargeHandler(s).CreateAccessorialCharge())
	accessorialChargesAPI.Put("/:accessorialChargeID", NewAccessorialChargeHandler(s).UpdateAccessorialCharge())

	// Register the handlers for the organization
	organizationsAPI := api.Group("/organizations")
	organizationsAPI.Get("/me", NewOrganizationHandler(s).GetUserOrganization())

	// Register the handlers for the accounting control.
	accountingControlAPI := api.Group("/accounting-control")
	accountingControlAPI.Get("/", NewAccountingControlHandler(s).GetAccountingControl())
	accountingControlAPI.Put("/:accountingControlID", NewAccountingControlHandler(s).UpdateAccountingControlByID())

	// Register the handlers for the dispatch control
	dispatchControlAPI := api.Group("/dispatch-control")
	dispatchControlAPI.Get("/", NewDispatchControlHandler(s).GetDispatchControl())
	dispatchControlAPI.Put("/:dispatchControlID", NewDispatchControlHandler(s).UpdateDispatchControlByID())

	// Register the handlers for the shipment control.
	shipmentControlAPI := api.Group("/shipment-control")
	shipmentControlAPI.Get("/", NewShipmentControlHandler(s).GetShipmentControl())
	shipmentControlAPI.Put("/:shipmentControlID", NewShipmentControlHandler(s).UpdateShipmentControlByID())

	// Register the handlers for the billing control.
	billingControlAPI := api.Group("/billing-control")
	billingControlAPI.Get("/", NewBillingControlHandler(s).GetBillingControl())
	billingControlAPI.Put("/:billingControlID", NewBillingControlHandler(s).UpdateBillingControl())

	// Register the handlers for the invoice control.
	invoiceControlAPI := api.Group("/invoice-control")
	invoiceControlAPI.Get("/", NewInvoiceControlHandler(s).GetInvoiceControl())
	invoiceControlAPI.Put("/:invoiceControlID", NewInvoiceControlHandler(s).UpdateInvoiceControlByID())

	// Register the handlers for the route control.
	routeControlAPI := api.Group("/route-control")
	routeControlAPI.Get("/", NewRouteControlHandler(s).GetRouteControl())
	routeControlAPI.Put("/:routeControlID", NewRouteControlHandler(s).UpdateRouteControlByID())

	// Register the handlers for the feasibility tool control.
	feasibilityToolControlAPI := api.Group("/feasibility-tool-control")
	feasibilityToolControlAPI.Get("/", NewFeasibilityToolControlHandler(s).GetFeasibilityToolControl())
	feasibilityToolControlAPI.Put("/:feasibilityToolControlID", NewFeasibilityToolControlHandler(s).UpdateFeasibilityToolControl())

	// Register the handlers the email control.
	emailControlAPI := api.Group("/email-control")
	emailControlAPI.Get("/", NewEmailControlHandler(s).GetEmailControl())
	emailControlAPI.Put("/:emailControlID", NewEmailControlHandler(s).UpdateEmailControl())

	// Register the handlers for the user.
	usersAPI := api.Group("/users")
	usersAPI.Get("/me", NewUserHandler(s).GetAuthenticatedUser())

	// Register the handlers for the user favorites.
	userFavoritesAPI := api.Group("/user-favorites")
	userFavoritesAPI.Get("/", NewUserFavoriteHandler(s).GetUserFavorites())
	userFavoritesAPI.Post("/", NewUserFavoriteHandler(s).AddUserFavorite())
	userFavoritesAPI.Delete("/", NewUserFavoriteHandler(s).RemoveUserFavorite())

	// Register the handlers for the hazardous material segregation rules.
	hazardousMaterialSegregationsAPI := api.Group("/hazardous-material-segregations")
	hazardousMaterialSegregationsAPI.Get("/", NewHazardousMaterialSegregationHandler(s).GetHazmatSegregationRules())
	hazardousMaterialSegregationsAPI.Post("/", NewHazardousMaterialSegregationHandler(s).CreateHazmatSegregationRule())
	hazardousMaterialSegregationsAPI.Put("/:hazmatSegRuleID", NewHazardousMaterialSegregationHandler(s).UpdateHazmatSegregationRules())

	// Register the handlers for the email profiles.
	emailProfilesAPI := api.Group("/email-profiles")
	emailProfilesAPI.Get("/", NewEmailProfileHandler(s).GetEmailProfiles())
	emailProfilesAPI.Post("/", NewEmailProfileHandler(s).CreateEmailProfile())
	emailProfilesAPI.Put("/:emailProfileID", NewEmailProfileHandler(s).UpdateEmailProfile())

	// Register the handlers for the table change alerts.
	tableChangeAlertsAPI := api.Group("/table-change-alerts")
	tableChangeAlertsAPI.Get("/", NewTableChangeAlertHandler(s).GetTableChangeAlerts())
	tableChangeAlertsAPI.Post("/", NewTableChangeAlertHandler(s).CreateTableChangeAlert())
	tableChangeAlertsAPI.Put("/:tableChangeAlertID", NewTableChangeAlertHandler(s).UpdateTableChangeAlert())
	tableChangeAlertsAPI.Get("/table-names", NewTableChangeAlertHandler(s).GetTableNames())
	tableChangeAlertsAPI.Get("/topic-names", NewTableChangeAlertHandler(s).GetTopicNames())

	// Register the handlers for the Google api.
	googleAPI := api.Group("/google-api")
	googleAPI.Get("/", NewGoogleAPIHandler(s).GetGoogleAPI())
	googleAPI.Put("/:googleAPIID", NewGoogleAPIHandler(s).UpdateGoogleAPI())

	// Register the handlers for the revenue code.
	revenueCodesAPI := api.Group("/revenue-codes")
	revenueCodesAPI.Get("/", NewRevenueCodeHandler(s).GetRevenueCodes())
	revenueCodesAPI.Post("/", NewRevenueCodeHandler(s).CreateRevenueCode())
	revenueCodesAPI.Put("/:revenueCodeID", NewRevenueCodeHandler(s).UpdateRevenueCode())

	// Register the handlers for the worker.
	workersAPI := api.Group("/workers")
	workersAPI.Get("/", NewWorkerHandler(s).GetWorkers())
	workersAPI.Post("/", NewWorkerHandler(s).CreateWorker())
	workersAPI.Put("/:workerID", NewWorkerHandler(s).UpdateWorker())

	// Register the handlers for the charge type.
	chargeTypesAPI := api.Group("/charge-types")
	chargeTypesAPI.Get("/", NewChargeTypeHandler(s).GetChargeTypes())
	chargeTypesAPI.Post("/", NewChargeTypeHandler(s).CreateChargeType())
	chargeTypesAPI.Put("/:chargeTypeID", NewChargeTypeHandler(s).UpdateChargeType())

	// Register the handlers for the comment type.
	commentTypesAPI := api.Group("/comment-types")
	commentTypesAPI.Get("/", NewCommentTypeService(s).GetCommentTypes())
	commentTypesAPI.Post("/", NewCommentTypeService(s).CreateCommentType())
	commentTypesAPI.Put("/:commentTypeID", NewCommentTypeService(s).UpdateCommentType())

	// Register the handlers for the commodity.
	commoditiesAPI := api.Group("/commodities")
	commoditiesAPI.Get("/", NewCommodityHandler(s).GetCommodities())
	commoditiesAPI.Post("/", NewCommodityHandler(s).CreateCommodity())
	commoditiesAPI.Put("/:commodityID", NewCommodityHandler(s).UpdateCommodity())

	// Register the handlers for the customer.
	customersAPI := api.Group("/customers")
	customersAPI.Get("/", NewCustomerHandler(s).GetCustomers())
	customersAPI.Post("/", NewCustomerHandler(s).CreateCustomer())
	customersAPI.Put("/:customerID", NewCustomerHandler(s).UpdateCustomer())

	// Register the handlers for the delay code.
	delayCodesAPI := api.Group("/delay-codes")
	delayCodesAPI.Get("/", NewDelayCodeHandler(s).GetDelayCodes())
	delayCodesAPI.Post("/", NewDelayCodeHandler(s).CreateDelayCode())
	delayCodesAPI.Put("/:delayCodeID", NewDelayCodeHandler(s).UpdateDelayCode())

	// Register the handlers for the division code.
	divisionCodesAPI := api.Group("/division-codes")
	divisionCodesAPI.Get("/", NewDivisionCodeHandler(s).GetDivisionCodes())
	divisionCodesAPI.Post("/", NewDivisionCodeHandler(s).CreateDivisionCode())
	divisionCodesAPI.Put("/:divisionCodeID", NewDivisionCodeHandler(s).UpdateDivisionCode())

	// Register the handlers for the document classification.
	documentClassificationsAPI := api.Group("/document-classifications")
	documentClassificationsAPI.Get("/", NewDocumentClassificationHandler(s).GetDocumentClassifications())
	documentClassificationsAPI.Post("/", NewDocumentClassificationHandler(s).CreateDocumentClassification())
	documentClassificationsAPI.Put("/:documentClassID", NewDocumentClassificationHandler(s).UpdateDocumentClassification())

	// Register the handlers for the equipment manufacturer.
	equipmentManufacturersAPI := api.Group("/equipment-manufacturers")
	equipmentManufacturersAPI.Get("/", NewEquipmentManufacturerHandler(s).GetEquipmentManufacturers())
	equipmentManufacturersAPI.Post("/", NewEquipmentManufacturerHandler(s).CreateEquipmentManufacturer())
	equipmentManufacturersAPI.Put("/:equipmentManuID", NewEquipmentManufacturerHandler(s).UpdateEquipmentManufacturer())

	// Register the handlers for the equipment type.
	equipmentTypesAPI := api.Group("/equipment-types")
	equipmentTypesAPI.Get("/", NewEquipmentTypeHandler(s).GetEquipmentTypes())
	equipmentTypesAPI.Post("/", NewEquipmentTypeHandler(s).CreateEquipmentType())
	equipmentTypesAPI.Put("/:equipmentTypeID", NewEquipmentTypeHandler(s).UpdateEquipmentType())

	// Register the handlers for the fleet code.
	fleetCodesAPI := api.Group("/fleet-codes")
	fleetCodesAPI.Get("/", NewFleetCodeHandler(s).GetFleetCodes())
	fleetCodesAPI.Post("/", NewFleetCodeHandler(s).CreateFleetCode())
	fleetCodesAPI.Put("/:fleetCodeID", NewFleetCodeHandler(s).UpdateFleetCode())

	// Register the handlers for the general ledger account.
	generalLedgerAccountsAPI := api.Group("/general-ledger-accounts")
	generalLedgerAccountsAPI.Get("/", NewGeneralLedgerAccountHandler(s).GetGeneralLedgerAccounts())
	generalLedgerAccountsAPI.Post("/", NewGeneralLedgerAccountHandler(s).CreateGeneralLedgerAccount())
	generalLedgerAccountsAPI.Put("/:glAccountID", NewGeneralLedgerAccountHandler(s).UpdateGeneralLedgerAccount())

	// Register the handlers for the tag.
	tagsAPI := api.Group("/tags")
	tagsAPI.Get("/", NewTagHandler(s).GetTags())
	tagsAPI.Post("/", NewTagHandler(s).CreateTag())
	tagsAPI.Put("/:tagID", NewTagHandler(s).UpdateTag())

	// Register the handlers for the hazardous material.
	hazardousMaterialsAPI := api.Group("/hazardous-materials")
	hazardousMaterialsAPI.Get("/", NewHazardousMaterialHandler(s).GetHazardousMaterials())
	hazardousMaterialsAPI.Post("/", NewHazardousMaterialHandler(s).CreateHazardousMaterial())
	hazardousMaterialsAPI.Put("/:hazmatID", NewHazardousMaterialHandler(s).UpdateHazardousMaterial())

	// Register the handlers for the location.
	locationsAPI := api.Group("/locations")
	locationsAPI.Get("/", NewLocationHandler(s).GetLocations())
	locationsAPI.Post("/", NewLocationHandler(s).CreateLocation())
	locationsAPI.Put("/:locationID", NewLocationHandler(s).UpdateLocation())

	// Register the handlers for the location categories.
	locationCategoriesAPI := api.Group("/location-categories")
	locationCategoriesAPI.Get("/", NewLocationCategoryHandler(s).GetLocationCategories())
	locationCategoriesAPI.Post("/", NewLocationCategoryHandler(s).CreateLocationCategory())
	locationCategoriesAPI.Put("/:locationCategoryID", NewLocationCategoryHandler(s).UpdateLocationCategory())

	// Register the handlers for the feature flags.
	featureFlagsAPI := api.Group("/feature-flags")
	featureFlagsAPI.Get("/", NewFeatureFlagHandler(s).GetFeatureFlags())

	// Register the handlers for the qualifier codes.
	qualifierCodesAPI := api.Group("/qualifier-codes")
	qualifierCodesAPI.Get("/", NewQualifierCodeHandler(s).GetQualifierCodes())
	qualifierCodesAPI.Post("/", NewQualifierCodeHandler(s).CreateQualifierCode())
	qualifierCodesAPI.Put("/:qualifierCodeID", NewQualifierCodeHandler(s).UpdateQualifierCode())

	// Register the handlers for the reason codes.
	reasonCodesAPI := api.Group("/reason-codes")
	reasonCodesAPI.Get("/", NewReasonCodeHandler(s).GetReasonCodes())
	reasonCodesAPI.Post("/", NewReasonCodeHandler(s).CreateReasonCode())
	reasonCodesAPI.Put("/:reasonCodeID", NewReasonCodeHandler(s).UpdateReasonCode())

	// Register the handlers for the service types.
	serviceTypesAPI := api.Group("/service-types")
	serviceTypesAPI.Get("/", NewServiceTypeHandler(s).GetServiceTypes())
	serviceTypesAPI.Post("/", NewServiceTypeHandler(s).CreateServiceType())
	serviceTypesAPI.Put("/:serviceTypeID", NewServiceTypeHandler(s).UpdateServiceType())

	// Register the handlers for the shipment types.
	shipmentTypesAPI := api.Group("/shipment-types")
	shipmentTypesAPI.Get("/", NewShipmentTypeHandler(s).GetShipmentTypes())
	shipmentTypesAPI.Post("/", NewShipmentTypeHandler(s).CreateShipmentType())
	shipmentTypesAPI.Put("/:shipmentTypeID", NewShipmentTypeHandler(s).UpdateShipmentType())

	// Register the handlers for the US states.
	usStatesAPI := api.Group("/us-states")
	usStatesAPI.Get("/", NewUSStateHandler(s).GetUSStates())

	// Register the handlers for the tractors.
	tractorsAPI := api.Group("/tractors")
	tractorsAPI.Get("/", NewTractorHandler(s).GetTractors())
	tractorsAPI.Post("/", NewTractorHandler(s).CreateTractor())
	tractorsAPI.Put("/:tractorID", NewTractorHandler(s).UpdateTractor())

	// Register the handlers for the trailers.
	trailersAPI := api.Group("/trailers")
	trailersAPI.Get("/", NewTrailerHandler(s).GetTrailers())
	trailersAPI.Post("/", NewTrailerHandler(s).CreateTrailer())
	trailersAPI.Put("/:trailerID", NewTrailerHandler(s).UpdateTrailer())

	// Register the handlers for the reports.
	reportsAPI := api.Group("/reports")
	reportsAPI.Get("/column-names", NewReportHandler(s).GetColumnNames())
	reportsAPI.Post("/generate", NewReportHandler(s).GenerateReport())

	// Register the handlers for the user notifications.
	userNotificationsAPI := api.Group("/user-notifications")
	userNotificationsAPI.Get("/", NewUserNotificationHandler(s).GetUserNotifications())

	// Register the handlers the permissions.
	permissionsAPI := api.Group("/permissions")
	permissionsAPI.Get("/", NewPermissionHandler(s).GetPermissions())

	// Register the handlers for the roles.
	rolesAPI := api.Group("/roles")
	rolesAPI.Get("/", NewRoleHandler(s).GetRoles())
	rolesAPI.Post("/", NewRoleHandler(s).CreateRole())
	rolesAPI.Put("/:roleID", NewRoleHandler(s).UpdateRole())
}
