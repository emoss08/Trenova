//revive:disable-next-line:var-naming
package api

import (
	"time"

	graphqlapi "github.com/emoss08/trenova/internal/api/graphql"
	"github.com/emoss08/trenova/internal/api/handlers/accessorialchargehandler"
	"github.com/emoss08/trenova/internal/api/handlers/accountingcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/accountsreceivablehandler"
	"github.com/emoss08/trenova/internal/api/handlers/accounttypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/analyticshandler"
	"github.com/emoss08/trenova/internal/api/handlers/apikeyhandler"
	"github.com/emoss08/trenova/internal/api/handlers/assignmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/authhandler"
	"github.com/emoss08/trenova/internal/api/handlers/bankreceiptbatchhandler"
	"github.com/emoss08/trenova/internal/api/handlers/bankreceipthandler"
	"github.com/emoss08/trenova/internal/api/handlers/bankreceiptworkitemhandler"
	"github.com/emoss08/trenova/internal/api/handlers/billingcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/billingqueuehandler"
	"github.com/emoss08/trenova/internal/api/handlers/commodityhandler"
	"github.com/emoss08/trenova/internal/api/handlers/controlplaneprovisioninghandler"
	"github.com/emoss08/trenova/internal/api/handlers/customerhandler"
	"github.com/emoss08/trenova/internal/api/handlers/customerpaymenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/customfieldhandler"
	"github.com/emoss08/trenova/internal/api/handlers/databasesessionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/dataentrycontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/dataretentionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/dispatchcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/distancecontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/distanceoverridehandler"
	"github.com/emoss08/trenova/internal/api/handlers/distanceprofilehandler"
	"github.com/emoss08/trenova/internal/api/handlers/docshandler"
	"github.com/emoss08/trenova/internal/api/handlers/documentcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/documenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/documentoperationshandler"
	"github.com/emoss08/trenova/internal/api/handlers/documentpacketrulehandler"
	"github.com/emoss08/trenova/internal/api/handlers/documentparsingrulehandler"
	"github.com/emoss08/trenova/internal/api/handlers/documenttypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/dothazmatreferencehandler"
	"github.com/emoss08/trenova/internal/api/handlers/driverportalhandler"
	"github.com/emoss08/trenova/internal/api/handlers/edihandler"
	"github.com/emoss08/trenova/internal/api/handlers/emailhandler"
	"github.com/emoss08/trenova/internal/api/handlers/equipmentmanufacturerhandler"
	"github.com/emoss08/trenova/internal/api/handlers/equipmenttypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/exchangeratehandler"
	"github.com/emoss08/trenova/internal/api/handlers/fiscalperiodhandler"
	"github.com/emoss08/trenova/internal/api/handlers/fiscalyearhandler"
	"github.com/emoss08/trenova/internal/api/handlers/fleetcodehandler"
	"github.com/emoss08/trenova/internal/api/handlers/formulatemplatehandler"
	"github.com/emoss08/trenova/internal/api/handlers/glaccounthandler"
	"github.com/emoss08/trenova/internal/api/handlers/glbalancehandler"
	"github.com/emoss08/trenova/internal/api/handlers/googlemapshandler"
	"github.com/emoss08/trenova/internal/api/handlers/hazardousmaterialhandler"
	"github.com/emoss08/trenova/internal/api/handlers/hazmatsegregationrulehandler"
	"github.com/emoss08/trenova/internal/api/handlers/holdreasonhandler"
	"github.com/emoss08/trenova/internal/api/handlers/iamhandler"
	"github.com/emoss08/trenova/internal/api/handlers/integrationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/invoiceadjustmentcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/invoiceadjustmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/invoicehandler"
	"github.com/emoss08/trenova/internal/api/handlers/journalentryhandler"
	"github.com/emoss08/trenova/internal/api/handlers/journalreversalhandler"
	"github.com/emoss08/trenova/internal/api/handlers/locationcategoryhandler"
	"github.com/emoss08/trenova/internal/api/handlers/locationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/manualjournalhandler"
	"github.com/emoss08/trenova/internal/api/handlers/orderhandler"
	"github.com/emoss08/trenova/internal/api/handlers/organizationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/pagefavoritehandler"
	"github.com/emoss08/trenova/internal/api/handlers/permissionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/platformcataloghandler"
	"github.com/emoss08/trenova/internal/api/handlers/pushhandler"
	"github.com/emoss08/trenova/internal/api/handlers/ratetablehandler"
	"github.com/emoss08/trenova/internal/api/handlers/realtimehandler"
	"github.com/emoss08/trenova/internal/api/handlers/recurringshipmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/reporthandler"
	"github.com/emoss08/trenova/internal/api/handlers/roleassignmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/rolehandler"
	"github.com/emoss08/trenova/internal/api/handlers/searchhandler"
	"github.com/emoss08/trenova/internal/api/handlers/sequenceconfighandler"
	"github.com/emoss08/trenova/internal/api/handlers/servicefailurehandler"
	"github.com/emoss08/trenova/internal/api/handlers/servicefailurereasoncodehandler"
	"github.com/emoss08/trenova/internal/api/handlers/servicetypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmentcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmenteventhandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmentmovehandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmenttypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/storedmileagehandler"
	"github.com/emoss08/trenova/internal/api/handlers/tablechangealerthandler"
	"github.com/emoss08/trenova/internal/api/handlers/telematicshandler"
	"github.com/emoss08/trenova/internal/api/handlers/tractorhandler"
	"github.com/emoss08/trenova/internal/api/handlers/trailerhandler"
	"github.com/emoss08/trenova/internal/api/handlers/userhandler"
	"github.com/emoss08/trenova/internal/api/handlers/usstatehandler"
	"github.com/emoss08/trenova/internal/api/handlers/versionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/weatheralerthandler"
	"github.com/emoss08/trenova/internal/api/handlers/workerhandler"
	"github.com/emoss08/trenova/internal/api/handlers/workerptohandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/requestid"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RouterParams struct {
	fx.In

	Server                  *Server
	Config                  *config.Config
	Logger                  *zap.Logger
	ObservabilityMiddleware *observability.Middleware

	AuthMiddleware                  *middleware.AuthMiddleware
	ControlPlaneAccessMiddleware    *middleware.ControlPlaneAccessMiddleware
	PermissionMiddleware            *middleware.PermissionMiddleware
	ErrorHandler                    *helpers.ErrorHandler
	DocsHandler                     *docshandler.Handler
	OrganizationHandler             *organizationhandler.Handler
	DataRetentionHandler            *dataretentionhandler.Handler
	IAMHandler                      *iamhandler.Handler
	UserHandler                     *userhandler.Handler
	AuthHandler                     *authhandler.Handler
	DriverPortalHandler             *driverportalhandler.Handler
	PushHandler                     *pushhandler.Handler
	BankReceiptHandler              *bankreceipthandler.Handler
	BankReceiptBatchHandler         *bankreceiptbatchhandler.Handler
	BankReceiptWorkItemHandler      *bankreceiptworkitemhandler.Handler
	FormulaTemplateHandler          *formulatemplatehandler.Handler
	PageFavoriteHandler             *pagefavoritehandler.Handler
	EquipmentTypeHandler            *equipmenttypehandler.Handler
	EquipmentManufacturerHandler    *equipmentmanufacturerhandler.Handler
	FleetCodeHandler                *fleetcodehandler.Handler
	TractorHandler                  *tractorhandler.Handler
	TrailerHandler                  *trailerhandler.Handler
	GraphQLHandler                  *graphqlapi.Handler
	WorkerHandler                   *workerhandler.Handler
	PermissionHandler               *permissionhandler.Handler
	PlatformCatalogHandler          *platformcataloghandler.Handler
	RealtimeHandler                 *realtimehandler.Handler
	RoleHandler                     *rolehandler.Handler
	RoleAssignmentHandler           *roleassignmenthandler.Handler
	SearchHandler                   *searchhandler.Handler
	UsStateHandler                  *usstatehandler.Handler
	CustomFieldHandler              *customfieldhandler.Handler
	DatabaseSessionHandler          *databasesessionhandler.Handler
	DocumentHandler                 *documenthandler.Handler
	ReportHandler                   *reporthandler.Handler
	DocumentOperationsHandler       *documentoperationshandler.Handler
	AccessorialChargeHandler        *accessorialchargehandler.Handler
	VersionHandler                  *versionhandler.Handler
	ControlPlaneProvisioningHandler *controlplaneprovisioninghandler.Handler
	WeatherAlertHandler             *weatheralerthandler.Handler
	ServiceTypeHandler              *servicetypehandler.Handler
	OrderHandler                    *orderhandler.Handler
	ServiceFailureReasonCodeHandler *servicefailurereasoncodehandler.Handler
	ServiceFailureHandler           *servicefailurehandler.Handler
	SequenceConfigHandler           *sequenceconfighandler.Handler
	ShipmentControlHandler          *shipmentcontrolhandler.Handler
	ShipmentMoveHandler             *shipmentmovehandler.Handler
	ShipmentHandler                 *shipmenthandler.Handler
	ShipmentEventHandler            *shipmenteventhandler.Handler
	ShipmentTypeHandler             *shipmenttypehandler.Handler
	HazardousMaterialHandler        *hazardousmaterialhandler.Handler
	HazmatSegregationRuleHandler    *hazmatsegregationrulehandler.Handler
	DotHazmatReferenceHandler       *dothazmatreferencehandler.Handler
	EDIHandler                      *edihandler.Handler
	EmailHandler                    *emailhandler.Handler
	TelematicsHandler               *telematicshandler.Handler
	CommodityHandler                *commodityhandler.Handler
	CustomerHandler                 *customerhandler.Handler
	CustomerPaymentHandler          *customerpaymenthandler.Handler
	GoogleMapsHandler               *googlemapshandler.Handler
	AccountingControlHandler        *accountingcontrolhandler.Handler
	AccountsReceivableHandler       *accountsreceivablehandler.Handler
	AccountTypeHandler              *accounttypehandler.Handler
	AssignmentHandler               *assignmenthandler.Handler
	GLAccountHandler                *glaccounthandler.Handler
	GLBalanceHandler                *glbalancehandler.Handler
	FiscalYearHandler               *fiscalyearhandler.Handler
	FiscalPeriodHandler             *fiscalperiodhandler.Handler
	LocationCategoryHandler         *locationcategoryhandler.Handler
	LocationHandler                 *locationhandler.Handler
	DocumentTypeHandler             *documenttypehandler.Handler
	HoldReasonHandler               *holdreasonhandler.Handler
	RecurringShipmentHandler        *recurringshipmenthandler.Handler
	RateTableHandler                *ratetablehandler.Handler
	IntegrationHandler              *integrationhandler.Handler
	InvoiceHandler                  *invoicehandler.Handler
	InvoiceAdjustmentHandler        *invoiceadjustmenthandler.Handler
	JournalEntryHandler             *journalentryhandler.Handler
	JournalReversalHandler          *journalreversalhandler.Handler
	ManualJournalHandler            *manualjournalhandler.Handler
	BillingControlHandler           *billingcontrolhandler.Handler
	InvoiceAdjustmentControlHandler *invoiceadjustmentcontrolhandler.Handler
	BillingQueueHandler             *billingqueuehandler.Handler
	DataEntryControlHandler         *dataentrycontrolhandler.Handler
	DispatchControlHandler          *dispatchcontrolhandler.Handler
	DocumentControlHandler          *documentcontrolhandler.Handler
	DocumentParsingRuleHandler      *documentparsingrulehandler.Handler
	WorkerPTOHandler                *workerptohandler.Handler
	ExchangeRateHandler             *exchangeratehandler.Handler
	DistanceControlHandler          *distancecontrolhandler.Handler
	DistanceOverrideHandler         *distanceoverridehandler.Handler
	DistanceProfileHandler          *distanceprofilehandler.Handler
	StoredMileageHandler            *storedmileagehandler.Handler
	AnalyticsHandler                *analyticshandler.Handler
	ApiKeyHandler                   *apikeyhandler.Handler //nolint:revive // field name follows existing router wiring
	TableChangeAlertHandler         *tablechangealerthandler.Handler
	DocumentPacketRuleHandler       *documentpacketrulehandler.Handler
}

type Router struct {
	s                               *Server
	l                               *zap.Logger
	observabilityMiddleware         *observability.Middleware
	authMiddleware                  *middleware.AuthMiddleware
	controlPlaneAccessMiddleware    *middleware.ControlPlaneAccessMiddleware
	permissionMiddleware            *middleware.PermissionMiddleware
	cfg                             *config.Config
	errorHandler                    *helpers.ErrorHandler
	docsHandler                     *docshandler.Handler
	organizationHandler             *organizationhandler.Handler
	dataRetentionHandler            *dataretentionhandler.Handler
	iamHandler                      *iamhandler.Handler
	userHandler                     *userhandler.Handler
	authHandler                     *authhandler.Handler
	driverPortalHandler             *driverportalhandler.Handler
	pushHandler                     *pushhandler.Handler
	bankReceiptHandler              *bankreceipthandler.Handler
	bankReceiptBatchHandler         *bankreceiptbatchhandler.Handler
	bankReceiptWorkItemHandler      *bankreceiptworkitemhandler.Handler
	formulaTemplateHandler          *formulatemplatehandler.Handler
	pageFavoriteHandler             *pagefavoritehandler.Handler
	serviceTypeHandler              *servicetypehandler.Handler
	orderHandler                    *orderhandler.Handler
	serviceFailureReasonCodeHandler *servicefailurereasoncodehandler.Handler
	serviceFailureHandler           *servicefailurehandler.Handler
	sequenceConfigHandler           *sequenceconfighandler.Handler
	shipmentControlHandler          *shipmentcontrolhandler.Handler
	shipmentMoveHandler             *shipmentmovehandler.Handler
	shipmentHandler                 *shipmenthandler.Handler
	shipmentEventHandler            *shipmenteventhandler.Handler
	equipmentManufacturerHandler    *equipmentmanufacturerhandler.Handler
	equipmentTypeHandler            *equipmenttypehandler.Handler
	fleetCodeHandler                *fleetcodehandler.Handler
	tractorHandler                  *tractorhandler.Handler
	trailerHandler                  *trailerhandler.Handler
	graphQLHandler                  *graphqlapi.Handler
	workerHandler                   *workerhandler.Handler
	permissionHandler               *permissionhandler.Handler
	platformCatalogHandler          *platformcataloghandler.Handler
	realtimeHandler                 *realtimehandler.Handler
	roleHandler                     *rolehandler.Handler
	roleAssignmentHandler           *roleassignmenthandler.Handler
	searchHandler                   *searchhandler.Handler
	usStateHandler                  *usstatehandler.Handler
	customFieldHandler              *customfieldhandler.Handler
	databaseSessionHandler          *databasesessionhandler.Handler
	documentHandler                 *documenthandler.Handler
	reportHandler                   *reporthandler.Handler
	documentOperationsHandler       *documentoperationshandler.Handler
	accessorialChargeHandler        *accessorialchargehandler.Handler
	versionHandler                  *versionhandler.Handler
	controlPlaneProvisioningHandler *controlplaneprovisioninghandler.Handler
	weatherAlertHandler             *weatheralerthandler.Handler
	shipmentTypeHandler             *shipmenttypehandler.Handler
	hazardousMaterialHandler        *hazardousmaterialhandler.Handler
	hazmatSegregationRuleHandler    *hazmatsegregationrulehandler.Handler
	dotHazmatReferenceHandler       *dothazmatreferencehandler.Handler
	ediHandler                      *edihandler.Handler
	emailHandler                    *emailhandler.Handler
	telematicsHandler               *telematicshandler.Handler
	commodityHandler                *commodityhandler.Handler
	customerHandler                 *customerhandler.Handler
	customerPaymentHandler          *customerpaymenthandler.Handler
	googleMapsHandler               *googlemapshandler.Handler
	accountingControlHandler        *accountingcontrolhandler.Handler
	accountsReceivableHandler       *accountsreceivablehandler.Handler
	accountTypeHandler              *accounttypehandler.Handler
	assignmentHandler               *assignmenthandler.Handler
	glAccountHandler                *glaccounthandler.Handler
	glBalanceHandler                *glbalancehandler.Handler
	fiscalYearHandler               *fiscalyearhandler.Handler
	fiscalPeriodHandler             *fiscalperiodhandler.Handler
	locationCategoryHandler         *locationcategoryhandler.Handler
	locationHandler                 *locationhandler.Handler
	documentTypeHandler             *documenttypehandler.Handler
	holdReasonHandler               *holdreasonhandler.Handler
	recurringShipmentHandler        *recurringshipmenthandler.Handler
	rateTableHandler                *ratetablehandler.Handler
	integrationHandler              *integrationhandler.Handler
	invoiceHandler                  *invoicehandler.Handler
	invoiceAdjustmentHandler        *invoiceadjustmenthandler.Handler
	journalEntryHandler             *journalentryhandler.Handler
	journalReversalHandler          *journalreversalhandler.Handler
	manualJournalHandler            *manualjournalhandler.Handler
	billingControlHandler           *billingcontrolhandler.Handler
	invoiceAdjustmentControlHandler *invoiceadjustmentcontrolhandler.Handler
	billingQueueHandler             *billingqueuehandler.Handler
	dataEntryControlHandler         *dataentrycontrolhandler.Handler
	dispatchControlHandler          *dispatchcontrolhandler.Handler
	documentControlHandler          *documentcontrolhandler.Handler
	documentParsingRuleHandler      *documentparsingrulehandler.Handler
	workerPTOHandler                *workerptohandler.Handler
	exchangeRateHandler             *exchangeratehandler.Handler
	distanceControlHandler          *distancecontrolhandler.Handler
	distanceOverrideHandler         *distanceoverridehandler.Handler
	distanceProfileHandler          *distanceprofilehandler.Handler
	storedMileageHandler            *storedmileagehandler.Handler
	analyticsHandler                *analyticshandler.Handler
	apiKeyHandler                   *apikeyhandler.Handler
	tableChangeAlertHandler         *tablechangealerthandler.Handler
	documentPacketRuleHandler       *documentpacketrulehandler.Handler
}

//nolint:gocritic // This is a constructor
func NewRouter(p RouterParams) *Router {
	return &Router{
		s:                               p.Server,
		cfg:                             p.Config,
		l:                               p.Logger,
		observabilityMiddleware:         p.ObservabilityMiddleware,
		authMiddleware:                  p.AuthMiddleware,
		controlPlaneAccessMiddleware:    p.ControlPlaneAccessMiddleware,
		permissionMiddleware:            p.PermissionMiddleware,
		errorHandler:                    p.ErrorHandler,
		docsHandler:                     p.DocsHandler,
		organizationHandler:             p.OrganizationHandler,
		dataRetentionHandler:            p.DataRetentionHandler,
		iamHandler:                      p.IAMHandler,
		userHandler:                     p.UserHandler,
		authHandler:                     p.AuthHandler,
		driverPortalHandler:             p.DriverPortalHandler,
		pushHandler:                     p.PushHandler,
		bankReceiptHandler:              p.BankReceiptHandler,
		bankReceiptBatchHandler:         p.BankReceiptBatchHandler,
		bankReceiptWorkItemHandler:      p.BankReceiptWorkItemHandler,
		formulaTemplateHandler:          p.FormulaTemplateHandler,
		pageFavoriteHandler:             p.PageFavoriteHandler,
		serviceTypeHandler:              p.ServiceTypeHandler,
		orderHandler:                    p.OrderHandler,
		serviceFailureReasonCodeHandler: p.ServiceFailureReasonCodeHandler,
		serviceFailureHandler:           p.ServiceFailureHandler,
		sequenceConfigHandler:           p.SequenceConfigHandler,
		shipmentControlHandler:          p.ShipmentControlHandler,
		shipmentMoveHandler:             p.ShipmentMoveHandler,
		shipmentHandler:                 p.ShipmentHandler,
		shipmentEventHandler:            p.ShipmentEventHandler,
		equipmentManufacturerHandler:    p.EquipmentManufacturerHandler,
		equipmentTypeHandler:            p.EquipmentTypeHandler,
		fleetCodeHandler:                p.FleetCodeHandler,
		tractorHandler:                  p.TractorHandler,
		trailerHandler:                  p.TrailerHandler,
		graphQLHandler:                  p.GraphQLHandler,
		workerHandler:                   p.WorkerHandler,
		permissionHandler:               p.PermissionHandler,
		platformCatalogHandler:          p.PlatformCatalogHandler,
		realtimeHandler:                 p.RealtimeHandler,
		roleHandler:                     p.RoleHandler,
		roleAssignmentHandler:           p.RoleAssignmentHandler,
		searchHandler:                   p.SearchHandler,
		usStateHandler:                  p.UsStateHandler,
		customFieldHandler:              p.CustomFieldHandler,
		databaseSessionHandler:          p.DatabaseSessionHandler,
		documentHandler:                 p.DocumentHandler,
		reportHandler:                   p.ReportHandler,
		documentOperationsHandler:       p.DocumentOperationsHandler,
		accessorialChargeHandler:        p.AccessorialChargeHandler,
		versionHandler:                  p.VersionHandler,
		controlPlaneProvisioningHandler: p.ControlPlaneProvisioningHandler,
		weatherAlertHandler:             p.WeatherAlertHandler,
		shipmentTypeHandler:             p.ShipmentTypeHandler,
		hazardousMaterialHandler:        p.HazardousMaterialHandler,
		hazmatSegregationRuleHandler:    p.HazmatSegregationRuleHandler,
		dotHazmatReferenceHandler:       p.DotHazmatReferenceHandler,
		ediHandler:                      p.EDIHandler,
		emailHandler:                    p.EmailHandler,
		telematicsHandler:               p.TelematicsHandler,
		commodityHandler:                p.CommodityHandler,
		customerHandler:                 p.CustomerHandler,
		customerPaymentHandler:          p.CustomerPaymentHandler,
		googleMapsHandler:               p.GoogleMapsHandler,
		accountingControlHandler:        p.AccountingControlHandler,
		accountsReceivableHandler:       p.AccountsReceivableHandler,
		accountTypeHandler:              p.AccountTypeHandler,
		assignmentHandler:               p.AssignmentHandler,
		glAccountHandler:                p.GLAccountHandler,
		glBalanceHandler:                p.GLBalanceHandler,
		fiscalYearHandler:               p.FiscalYearHandler,
		fiscalPeriodHandler:             p.FiscalPeriodHandler,
		locationCategoryHandler:         p.LocationCategoryHandler,
		locationHandler:                 p.LocationHandler,
		documentTypeHandler:             p.DocumentTypeHandler,
		holdReasonHandler:               p.HoldReasonHandler,
		recurringShipmentHandler:        p.RecurringShipmentHandler,
		rateTableHandler:                p.RateTableHandler,
		integrationHandler:              p.IntegrationHandler,
		invoiceHandler:                  p.InvoiceHandler,
		invoiceAdjustmentHandler:        p.InvoiceAdjustmentHandler,
		journalEntryHandler:             p.JournalEntryHandler,
		journalReversalHandler:          p.JournalReversalHandler,
		manualJournalHandler:            p.ManualJournalHandler,
		billingControlHandler:           p.BillingControlHandler,
		invoiceAdjustmentControlHandler: p.InvoiceAdjustmentControlHandler,
		billingQueueHandler:             p.BillingQueueHandler,
		dataEntryControlHandler:         p.DataEntryControlHandler,
		dispatchControlHandler:          p.DispatchControlHandler,
		documentControlHandler:          p.DocumentControlHandler,
		documentParsingRuleHandler:      p.DocumentParsingRuleHandler,
		workerPTOHandler:                p.WorkerPTOHandler,
		exchangeRateHandler:             p.ExchangeRateHandler,
		distanceControlHandler:          p.DistanceControlHandler,
		distanceOverrideHandler:         p.DistanceOverrideHandler,
		distanceProfileHandler:          p.DistanceProfileHandler,
		storedMileageHandler:            p.StoredMileageHandler,
		analyticsHandler:                p.AnalyticsHandler,
		apiKeyHandler:                   p.ApiKeyHandler,
		tableChangeAlertHandler:         p.TableChangeAlertHandler,
		documentPacketRuleHandler:       p.DocumentPacketRuleHandler,
	}
}

func (r *Router) setupCors() {
	if !r.cfg.CorsEnabled() {
		return
	}

	corsConfig := cors.Config{
		AllowOrigins:     r.cfg.Server.CORS.AllowedOrigins,
		AllowMethods:     r.cfg.Server.CORS.AllowedMethods,
		AllowHeaders:     r.cfg.Server.CORS.AllowedHeaders,
		ExposeHeaders:    r.cfg.Server.CORS.ExposeHeaders,
		AllowCredentials: r.cfg.Server.CORS.Credentials,
		MaxAge:           time.Duration(r.cfg.Server.CORS.MaxAge) * time.Second,
	}

	// If AllowOrigins is empty or contains "*", allow all origins with credentials disabled
	if len(corsConfig.AllowOrigins) == 0 ||
		(len(corsConfig.AllowOrigins) == 1 && corsConfig.AllowOrigins[0] == "*") {
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowOrigins = nil
		corsConfig.AllowCredentials = false
	}

	r.s.router.Use(cors.New(corsConfig))
}

func (r *Router) Setup() {
	r.setupMiddleware()
	r.setupRoutes()
}

func (r *Router) setupMiddleware() {
	r.setupCors()

	r.s.router.Use(middleware.NewSecurityHeadersMiddleware(r.cfg))
	r.s.router.Use(gin.Recovery())
	r.s.router.Use(requestid.New())
	r.s.router.Use(middleware.NewCSRFBrowserGuard(r.cfg, r.errorHandler, r.l).Guard())
	r.s.router.Use(
		gzip.Gzip(
			gzip.DefaultCompression,
			gzip.WithExcludedPaths([]string{"/metrics", "/health"}),
			gzip.WithExcludedPathsRegexs([]string{
				`^/api/v1/documents/[^/]+/(download|view|preview)/$`,
			}),
		),
	)
	r.s.router.Use(ginzap.Ginzap(r.l, time.RFC3339, true))
	r.s.router.Use(r.observabilityMiddleware.TracingMiddleware())
	r.s.router.Use(middleware.NewRateLimiter(r.cfg, r.errorHandler).Middleware())
}

func (r *Router) setupRoutes() {
	r.setupGraphQLRoutes(r.s.router.Group(""))

	v1 := r.s.router.Group("/api/v1")
	r.setupProtectedRoutes(v1)
	r.setupPublicRoutes(v1)
}

func (r *Router) setupGraphQLRoutes(rg *gin.RouterGroup) {
	r.graphQLHandler.RegisterPlaygroundRoutes(rg)

	protected := r.protectedGroup(rg)
	r.graphQLHandler.RegisterRoutes(protected)
}

func (r *Router) setupPublicRoutes(rg *gin.RouterGroup) {
	r.docsHandler.RegisterRoutes(rg)
	r.authHandler.RegisterRoutes(rg)
	r.driverPortalHandler.RegisterRoutes(rg)
	r.versionHandler.RegisterPublicRoutes(rg)
	r.controlPlaneProvisioningHandler.RegisterPublicRoutes(rg)
	r.emailHandler.RegisterPublicRoutes(rg)
	r.telematicsHandler.RegisterPublicRoutes(rg)
	r.invoiceHandler.RegisterPublicRoutes(rg)
	r.ediHandler.RegisterPublicRoutes(rg)
}

//nolint:funlen // existing workflow or route registration is intentionally kept together
func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	protected := r.protectedGroup(rg)

	r.driverPortalHandler.RegisterProtectedRoutes(protected)
	r.pushHandler.RegisterRoutes(protected)
	r.organizationHandler.RegisterRoutes(protected)
	r.dataRetentionHandler.RegisterRoutes(protected)
	r.iamHandler.RegisterRoutes(protected)
	r.userHandler.RegisterRoutes(protected)
	r.bankReceiptBatchHandler.RegisterRoutes(protected)
	r.bankReceiptHandler.RegisterRoutes(protected)
	r.bankReceiptWorkItemHandler.RegisterRoutes(protected)
	r.formulaTemplateHandler.RegisterRoutes(protected)
	r.pageFavoriteHandler.RegisterRoutes(protected)
	r.equipmentManufacturerHandler.RegisterRoutes(protected)
	r.equipmentTypeHandler.RegisterRoutes(protected)
	r.fleetCodeHandler.RegisterRoutes(protected)
	r.tractorHandler.RegisterRoutes(protected)
	r.trailerHandler.RegisterRoutes(protected)
	r.workerHandler.RegisterRoutes(protected)
	r.permissionHandler.RegisterRoutes(protected)
	r.platformCatalogHandler.RegisterRoutes(protected)
	r.realtimeHandler.RegisterRoutes(protected)
	r.roleHandler.RegisterRoutes(protected)
	r.roleAssignmentHandler.RegisterRoutes(protected)
	r.searchHandler.RegisterRoutes(protected)
	r.usStateHandler.RegisterRoutes(protected)
	r.customFieldHandler.RegisterRoutes(protected)
	r.databaseSessionHandler.RegisterRoutes(protected)
	r.documentHandler.RegisterRoutes(protected)
	r.reportHandler.RegisterRoutes(protected)
	r.documentOperationsHandler.RegisterRoutes(protected)
	r.accessorialChargeHandler.RegisterRoutes(protected)
	r.serviceTypeHandler.RegisterRoutes(protected)
	r.orderHandler.RegisterRoutes(protected)
	r.serviceFailureReasonCodeHandler.RegisterRoutes(protected)
	r.serviceFailureHandler.RegisterRoutes(protected)
	r.sequenceConfigHandler.RegisterRoutes(protected)
	r.shipmentControlHandler.RegisterRoutes(protected)
	r.shipmentMoveHandler.RegisterRoutes(protected)
	r.shipmentHandler.RegisterRoutes(protected)
	r.shipmentEventHandler.RegisterRoutes(protected)
	r.shipmentTypeHandler.RegisterRoutes(protected)
	r.hazardousMaterialHandler.RegisterRoutes(protected)
	r.hazmatSegregationRuleHandler.RegisterRoutes(protected)
	r.dotHazmatReferenceHandler.RegisterRoutes(protected)
	r.ediHandler.RegisterRoutes(protected)
	r.emailHandler.RegisterRoutes(protected)
	r.commodityHandler.RegisterRoutes(protected)
	r.customerHandler.RegisterRoutes(protected)
	r.customerPaymentHandler.RegisterRoutes(protected)
	r.googleMapsHandler.RegisterRoutes(protected)
	r.weatherAlertHandler.RegisterRoutes(protected)
	r.accountingControlHandler.RegisterRoutes(protected)
	r.accountsReceivableHandler.RegisterRoutes(protected)
	r.accountTypeHandler.RegisterRoutes(protected)
	r.assignmentHandler.RegisterRoutes(protected)
	r.glAccountHandler.RegisterRoutes(protected)
	r.glBalanceHandler.RegisterRoutes(protected)
	r.fiscalYearHandler.RegisterRoutes(protected)
	r.fiscalPeriodHandler.RegisterRoutes(protected)
	r.locationCategoryHandler.RegisterRoutes(protected)
	r.locationHandler.RegisterRoutes(protected)
	r.documentTypeHandler.RegisterRoutes(protected)
	r.holdReasonHandler.RegisterRoutes(protected)
	r.recurringShipmentHandler.RegisterRoutes(protected)
	r.rateTableHandler.RegisterRoutes(protected)
	r.integrationHandler.RegisterRoutes(protected)
	r.invoiceHandler.RegisterRoutes(protected)
	r.invoiceAdjustmentHandler.RegisterRoutes(protected)
	r.journalEntryHandler.RegisterRoutes(protected)
	r.journalReversalHandler.RegisterRoutes(protected)
	r.manualJournalHandler.RegisterRoutes(protected)
	r.billingControlHandler.RegisterRoutes(protected)
	r.invoiceAdjustmentControlHandler.RegisterRoutes(protected)
	r.billingQueueHandler.RegisterRoutes(protected)
	r.dataEntryControlHandler.RegisterRoutes(protected)
	r.dispatchControlHandler.RegisterRoutes(protected)
	r.documentControlHandler.RegisterRoutes(protected)
	r.documentParsingRuleHandler.RegisterRoutes(protected)
	r.distanceControlHandler.RegisterRoutes(protected)
	r.distanceOverrideHandler.RegisterRoutes(protected)
	r.distanceProfileHandler.RegisterRoutes(protected)
	r.storedMileageHandler.RegisterRoutes(protected)
	r.exchangeRateHandler.RegisterRoutes(protected)
	r.workerPTOHandler.RegisterRoutes(protected)
	r.analyticsHandler.RegisterRoutes(protected)
	r.apiKeyHandler.RegisterRoutes(protected)
	r.tableChangeAlertHandler.RegisterRoutes(protected)
	r.documentPacketRuleHandler.RegisterRoutes(protected)
}

func (r *Router) protectedGroup(rg *gin.RouterGroup) *gin.RouterGroup {
	protected := rg.Group("")
	protected.Use(r.authMiddleware.RequireAuth())
	protected.Use(middleware.NewCSRFMiddleware(r.cfg, r.errorHandler).RequireToken())
	protected.Use(r.controlPlaneAccessMiddleware.RequireAccess())
	return protected
}
