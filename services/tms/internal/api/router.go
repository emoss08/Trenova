package api

import (
	"time"

	"github.com/emoss08/trenova/internal/api/handlers/accessorialchargehandler"
	"github.com/emoss08/trenova/internal/api/handlers/accountingcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/accounttypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/analyticshandler"
	"github.com/emoss08/trenova/internal/api/handlers/apikeyhandler"
	"github.com/emoss08/trenova/internal/api/handlers/assignmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/audithandler"
	"github.com/emoss08/trenova/internal/api/handlers/authhandler"
	"github.com/emoss08/trenova/internal/api/handlers/billingcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/billingqueuehandler"
	"github.com/emoss08/trenova/internal/api/handlers/commodityhandler"
	"github.com/emoss08/trenova/internal/api/handlers/customerhandler"
	"github.com/emoss08/trenova/internal/api/handlers/customfieldhandler"
	"github.com/emoss08/trenova/internal/api/handlers/databasesessionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/dataentrycontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/dispatchcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/distanceoverridehandler"
	"github.com/emoss08/trenova/internal/api/handlers/docshandler"
	"github.com/emoss08/trenova/internal/api/handlers/documentcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/documenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/documentoperationshandler"
	"github.com/emoss08/trenova/internal/api/handlers/documentpacketrulehandler"
	"github.com/emoss08/trenova/internal/api/handlers/documentparsingrulehandler"
	"github.com/emoss08/trenova/internal/api/handlers/documenttypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/dothazmatreferencehandler"
	"github.com/emoss08/trenova/internal/api/handlers/equipmentmanufacturerhandler"
	"github.com/emoss08/trenova/internal/api/handlers/equipmenttypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/fiscalperiodhandler"
	"github.com/emoss08/trenova/internal/api/handlers/fiscalyearhandler"
	"github.com/emoss08/trenova/internal/api/handlers/fleetcodehandler"
	"github.com/emoss08/trenova/internal/api/handlers/formulatemplatehandler"
	"github.com/emoss08/trenova/internal/api/handlers/glaccounthandler"
	"github.com/emoss08/trenova/internal/api/handlers/googlemapshandler"
	"github.com/emoss08/trenova/internal/api/handlers/hazardousmaterialhandler"
	"github.com/emoss08/trenova/internal/api/handlers/hazmatsegregationrulehandler"
	"github.com/emoss08/trenova/internal/api/handlers/holdreasonhandler"
	"github.com/emoss08/trenova/internal/api/handlers/integrationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/invoiceadjustmentcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/invoiceadjustmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/invoicehandler"
	"github.com/emoss08/trenova/internal/api/handlers/locationcategoryhandler"
	"github.com/emoss08/trenova/internal/api/handlers/locationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/notificationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/organizationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/pagefavoritehandler"
	"github.com/emoss08/trenova/internal/api/handlers/permissionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/realtimehandler"
	"github.com/emoss08/trenova/internal/api/handlers/roleassignmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/rolehandler"
	"github.com/emoss08/trenova/internal/api/handlers/searchhandler"
	"github.com/emoss08/trenova/internal/api/handlers/sequenceconfighandler"
	"github.com/emoss08/trenova/internal/api/handlers/servicetypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmentcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmentmovehandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmenttypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/tablechangealerthandler"
	"github.com/emoss08/trenova/internal/api/handlers/tableconfigurationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/tractorhandler"
	"github.com/emoss08/trenova/internal/api/handlers/trailerhandler"
	"github.com/emoss08/trenova/internal/api/handlers/userhandler"
	"github.com/emoss08/trenova/internal/api/handlers/usstatehandler"
	"github.com/emoss08/trenova/internal/api/handlers/versionhandler"
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
	PermissionMiddleware            *middleware.PermissionMiddleware
	ErrorHandler                    *helpers.ErrorHandler
	DocsHandler                     *docshandler.Handler
	OrganizationHandler             *organizationhandler.Handler
	UserHandler                     *userhandler.Handler
	AuthHandler                     *authhandler.Handler
	AuditHandler                    *audithandler.Handler
	FormulaTemplateHandler          *formulatemplatehandler.Handler
	TableConfigurationHandler       *tableconfigurationhandler.Handler
	PageFavoriteHandler             *pagefavoritehandler.Handler
	EquipmentTypeHandler            *equipmenttypehandler.Handler
	EquipmentManufacturerHandler    *equipmentmanufacturerhandler.Handler
	FleetCodeHandler                *fleetcodehandler.Handler
	TractorHandler                  *tractorhandler.Handler
	TrailerHandler                  *trailerhandler.Handler
	WorkerHandler                   *workerhandler.Handler
	PermissionHandler               *permissionhandler.Handler
	RealtimeHandler                 *realtimehandler.Handler
	RoleHandler                     *rolehandler.Handler
	RoleAssignmentHandler           *roleassignmenthandler.Handler
	SearchHandler                   *searchhandler.Handler
	UsStateHandler                  *usstatehandler.Handler
	CustomFieldHandler              *customfieldhandler.Handler
	DatabaseSessionHandler          *databasesessionhandler.Handler
	DocumentHandler                 *documenthandler.Handler
	DocumentOperationsHandler       *documentoperationshandler.Handler
	AccessorialChargeHandler        *accessorialchargehandler.Handler
	VersionHandler                  *versionhandler.Handler
	ServiceTypeHandler              *servicetypehandler.Handler
	SequenceConfigHandler           *sequenceconfighandler.Handler
	ShipmentControlHandler          *shipmentcontrolhandler.Handler
	ShipmentMoveHandler             *shipmentmovehandler.Handler
	ShipmentHandler                 *shipmenthandler.Handler
	ShipmentTypeHandler             *shipmenttypehandler.Handler
	HazardousMaterialHandler        *hazardousmaterialhandler.Handler
	HazmatSegregationRuleHandler    *hazmatsegregationrulehandler.Handler
	DotHazmatReferenceHandler       *dothazmatreferencehandler.Handler
	CommodityHandler                *commodityhandler.Handler
	CustomerHandler                 *customerhandler.Handler
	GoogleMapsHandler               *googlemapshandler.Handler
	AccountingControlHandler        *accountingcontrolhandler.Handler
	AccountTypeHandler              *accounttypehandler.Handler
	AssignmentHandler               *assignmenthandler.Handler
	GLAccountHandler                *glaccounthandler.Handler
	FiscalYearHandler               *fiscalyearhandler.Handler
	FiscalPeriodHandler             *fiscalperiodhandler.Handler
	LocationCategoryHandler         *locationcategoryhandler.Handler
	LocationHandler                 *locationhandler.Handler
	DocumentTypeHandler             *documenttypehandler.Handler
	HoldReasonHandler               *holdreasonhandler.Handler
	IntegrationHandler              *integrationhandler.Handler
	InvoiceHandler                  *invoicehandler.Handler
	InvoiceAdjustmentHandler        *invoiceadjustmenthandler.Handler
	BillingControlHandler           *billingcontrolhandler.Handler
	InvoiceAdjustmentControlHandler *invoiceadjustmentcontrolhandler.Handler
	BillingQueueHandler             *billingqueuehandler.Handler
	DataEntryControlHandler         *dataentrycontrolhandler.Handler
	DispatchControlHandler          *dispatchcontrolhandler.Handler
	DocumentControlHandler          *documentcontrolhandler.Handler
	DocumentParsingRuleHandler      *documentparsingrulehandler.Handler
	WorkerPTOHandler                *workerptohandler.Handler
	DistanceOverrideHandler         *distanceoverridehandler.Handler
	AnalyticsHandler                *analyticshandler.Handler
	ApiKeyHandler                   *apikeyhandler.Handler
	TableChangeAlertHandler         *tablechangealerthandler.Handler
	NotificationHandler             *notificationhandler.Handler
	DocumentPacketRuleHandler       *documentpacketrulehandler.Handler
}

type Router struct {
	s                               *Server
	l                               *zap.Logger
	observabilityMiddleware         *observability.Middleware
	authMiddleware                  *middleware.AuthMiddleware
	permissionMiddleware            *middleware.PermissionMiddleware
	cfg                             *config.Config
	errorHandler                    *helpers.ErrorHandler
	docsHandler                     *docshandler.Handler
	organizationHandler             *organizationhandler.Handler
	userHandler                     *userhandler.Handler
	authHandler                     *authhandler.Handler
	auditHandler                    *audithandler.Handler
	formulaTemplateHandler          *formulatemplatehandler.Handler
	tableConfigurationHandler       *tableconfigurationhandler.Handler
	pageFavoriteHandler             *pagefavoritehandler.Handler
	serviceTypeHandler              *servicetypehandler.Handler
	sequenceConfigHandler           *sequenceconfighandler.Handler
	shipmentControlHandler          *shipmentcontrolhandler.Handler
	shipmentMoveHandler             *shipmentmovehandler.Handler
	shipmentHandler                 *shipmenthandler.Handler
	equipmentManufacturerHandler    *equipmentmanufacturerhandler.Handler
	equipmentTypeHandler            *equipmenttypehandler.Handler
	fleetCodeHandler                *fleetcodehandler.Handler
	tractorHandler                  *tractorhandler.Handler
	trailerHandler                  *trailerhandler.Handler
	workerHandler                   *workerhandler.Handler
	permissionHandler               *permissionhandler.Handler
	realtimeHandler                 *realtimehandler.Handler
	roleHandler                     *rolehandler.Handler
	roleAssignmentHandler           *roleassignmenthandler.Handler
	searchHandler                   *searchhandler.Handler
	usStateHandler                  *usstatehandler.Handler
	customFieldHandler              *customfieldhandler.Handler
	databaseSessionHandler          *databasesessionhandler.Handler
	documentHandler                 *documenthandler.Handler
	documentOperationsHandler       *documentoperationshandler.Handler
	accessorialChargeHandler        *accessorialchargehandler.Handler
	versionHandler                  *versionhandler.Handler
	shipmentTypeHandler             *shipmenttypehandler.Handler
	hazardousMaterialHandler        *hazardousmaterialhandler.Handler
	hazmatSegregationRuleHandler    *hazmatsegregationrulehandler.Handler
	dotHazmatReferenceHandler       *dothazmatreferencehandler.Handler
	commodityHandler                *commodityhandler.Handler
	customerHandler                 *customerhandler.Handler
	googleMapsHandler               *googlemapshandler.Handler
	accountingControlHandler        *accountingcontrolhandler.Handler
	accountTypeHandler              *accounttypehandler.Handler
	assignmentHandler               *assignmenthandler.Handler
	glAccountHandler                *glaccounthandler.Handler
	fiscalYearHandler               *fiscalyearhandler.Handler
	fiscalPeriodHandler             *fiscalperiodhandler.Handler
	locationCategoryHandler         *locationcategoryhandler.Handler
	locationHandler                 *locationhandler.Handler
	documentTypeHandler             *documenttypehandler.Handler
	holdReasonHandler               *holdreasonhandler.Handler
	integrationHandler              *integrationhandler.Handler
	invoiceHandler                  *invoicehandler.Handler
	invoiceAdjustmentHandler        *invoiceadjustmenthandler.Handler
	billingControlHandler           *billingcontrolhandler.Handler
	invoiceAdjustmentControlHandler *invoiceadjustmentcontrolhandler.Handler
	billingQueueHandler             *billingqueuehandler.Handler
	dataEntryControlHandler         *dataentrycontrolhandler.Handler
	dispatchControlHandler          *dispatchcontrolhandler.Handler
	documentControlHandler          *documentcontrolhandler.Handler
	documentParsingRuleHandler      *documentparsingrulehandler.Handler
	workerPTOHandler                *workerptohandler.Handler
	distanceOverrideHandler         *distanceoverridehandler.Handler
	analyticsHandler                *analyticshandler.Handler
	apiKeyHandler                   *apikeyhandler.Handler
	tableChangeAlertHandler         *tablechangealerthandler.Handler
	notificationHandler             *notificationhandler.Handler
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
		permissionMiddleware:            p.PermissionMiddleware,
		errorHandler:                    p.ErrorHandler,
		docsHandler:                     p.DocsHandler,
		organizationHandler:             p.OrganizationHandler,
		userHandler:                     p.UserHandler,
		authHandler:                     p.AuthHandler,
		auditHandler:                    p.AuditHandler,
		formulaTemplateHandler:          p.FormulaTemplateHandler,
		tableConfigurationHandler:       p.TableConfigurationHandler,
		pageFavoriteHandler:             p.PageFavoriteHandler,
		serviceTypeHandler:              p.ServiceTypeHandler,
		sequenceConfigHandler:           p.SequenceConfigHandler,
		shipmentControlHandler:          p.ShipmentControlHandler,
		shipmentMoveHandler:             p.ShipmentMoveHandler,
		shipmentHandler:                 p.ShipmentHandler,
		equipmentManufacturerHandler:    p.EquipmentManufacturerHandler,
		equipmentTypeHandler:            p.EquipmentTypeHandler,
		fleetCodeHandler:                p.FleetCodeHandler,
		tractorHandler:                  p.TractorHandler,
		trailerHandler:                  p.TrailerHandler,
		workerHandler:                   p.WorkerHandler,
		permissionHandler:               p.PermissionHandler,
		realtimeHandler:                 p.RealtimeHandler,
		roleHandler:                     p.RoleHandler,
		roleAssignmentHandler:           p.RoleAssignmentHandler,
		searchHandler:                   p.SearchHandler,
		usStateHandler:                  p.UsStateHandler,
		customFieldHandler:              p.CustomFieldHandler,
		databaseSessionHandler:          p.DatabaseSessionHandler,
		documentHandler:                 p.DocumentHandler,
		documentOperationsHandler:       p.DocumentOperationsHandler,
		accessorialChargeHandler:        p.AccessorialChargeHandler,
		versionHandler:                  p.VersionHandler,
		shipmentTypeHandler:             p.ShipmentTypeHandler,
		hazardousMaterialHandler:        p.HazardousMaterialHandler,
		hazmatSegregationRuleHandler:    p.HazmatSegregationRuleHandler,
		dotHazmatReferenceHandler:       p.DotHazmatReferenceHandler,
		commodityHandler:                p.CommodityHandler,
		customerHandler:                 p.CustomerHandler,
		googleMapsHandler:               p.GoogleMapsHandler,
		accountingControlHandler:        p.AccountingControlHandler,
		accountTypeHandler:              p.AccountTypeHandler,
		assignmentHandler:               p.AssignmentHandler,
		glAccountHandler:                p.GLAccountHandler,
		fiscalYearHandler:               p.FiscalYearHandler,
		fiscalPeriodHandler:             p.FiscalPeriodHandler,
		locationCategoryHandler:         p.LocationCategoryHandler,
		locationHandler:                 p.LocationHandler,
		documentTypeHandler:             p.DocumentTypeHandler,
		holdReasonHandler:               p.HoldReasonHandler,
		integrationHandler:              p.IntegrationHandler,
		invoiceHandler:                  p.InvoiceHandler,
		invoiceAdjustmentHandler:        p.InvoiceAdjustmentHandler,
		billingControlHandler:           p.BillingControlHandler,
		invoiceAdjustmentControlHandler: p.InvoiceAdjustmentControlHandler,
		billingQueueHandler:             p.BillingQueueHandler,
		dataEntryControlHandler:         p.DataEntryControlHandler,
		dispatchControlHandler:          p.DispatchControlHandler,
		documentControlHandler:          p.DocumentControlHandler,
		documentParsingRuleHandler:      p.DocumentParsingRuleHandler,
		workerPTOHandler:                p.WorkerPTOHandler,
		distanceOverrideHandler:         p.DistanceOverrideHandler,
		analyticsHandler:                p.AnalyticsHandler,
		apiKeyHandler:                   p.ApiKeyHandler,
		tableChangeAlertHandler:         p.TableChangeAlertHandler,
		notificationHandler:             p.NotificationHandler,
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

	r.s.router.Use(gin.Recovery())
	r.s.router.Use(requestid.New())
	r.s.router.Use(
		gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/metrics", "/health"})),
	)
	r.s.router.Use(ginzap.Ginzap(r.l, time.RFC3339, true))
	r.s.router.Use(r.observabilityMiddleware.TracingMiddleware())
}

func (r *Router) setupRoutes() {
	v1 := r.s.router.Group("/api/v1")
	r.setupProtectedRoutes(v1)
	r.setupPublicRoutes(v1)
}

func (r *Router) setupPublicRoutes(rg *gin.RouterGroup) {
	r.docsHandler.RegisterRoutes(rg)
	r.authHandler.RegisterRoutes(rg)
	r.versionHandler.RegisterPublicRoutes(rg)
}

func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	protected := rg.Group("")
	protected.Use(r.authMiddleware.RequireAuth())

	r.organizationHandler.RegisterRoutes(protected)
	r.userHandler.RegisterRoutes(protected)
	r.auditHandler.RegisterRoutes(protected)
	r.formulaTemplateHandler.RegisterRoutes(protected)
	r.tableConfigurationHandler.RegisterRoutes(protected)
	r.pageFavoriteHandler.RegisterRoutes(protected)
	r.equipmentManufacturerHandler.RegisterRoutes(protected)
	r.equipmentTypeHandler.RegisterRoutes(protected)
	r.fleetCodeHandler.RegisterRoutes(protected)
	r.tractorHandler.RegisterRoutes(protected)
	r.trailerHandler.RegisterRoutes(protected)
	r.workerHandler.RegisterRoutes(protected)
	r.permissionHandler.RegisterRoutes(protected)
	r.realtimeHandler.RegisterRoutes(protected)
	r.roleHandler.RegisterRoutes(protected)
	r.roleAssignmentHandler.RegisterRoutes(protected)
	r.searchHandler.RegisterRoutes(protected)
	r.usStateHandler.RegisterRoutes(protected)
	r.customFieldHandler.RegisterRoutes(protected)
	r.databaseSessionHandler.RegisterRoutes(protected)
	r.documentHandler.RegisterRoutes(protected)
	r.documentOperationsHandler.RegisterRoutes(protected)
	r.accessorialChargeHandler.RegisterRoutes(protected)
	r.serviceTypeHandler.RegisterRoutes(protected)
	r.sequenceConfigHandler.RegisterRoutes(protected)
	r.shipmentControlHandler.RegisterRoutes(protected)
	r.shipmentMoveHandler.RegisterRoutes(protected)
	r.shipmentHandler.RegisterRoutes(protected)
	r.shipmentTypeHandler.RegisterRoutes(protected)
	r.hazardousMaterialHandler.RegisterRoutes(protected)
	r.hazmatSegregationRuleHandler.RegisterRoutes(protected)
	r.dotHazmatReferenceHandler.RegisterRoutes(protected)
	r.commodityHandler.RegisterRoutes(protected)
	r.customerHandler.RegisterRoutes(protected)
	r.googleMapsHandler.RegisterRoutes(protected)
	r.accountingControlHandler.RegisterRoutes(protected)
	r.accountTypeHandler.RegisterRoutes(protected)
	r.assignmentHandler.RegisterRoutes(protected)
	r.glAccountHandler.RegisterRoutes(protected)
	r.fiscalYearHandler.RegisterRoutes(protected)
	r.fiscalPeriodHandler.RegisterRoutes(protected)
	r.locationCategoryHandler.RegisterRoutes(protected)
	r.locationHandler.RegisterRoutes(protected)
	r.documentTypeHandler.RegisterRoutes(protected)
	r.holdReasonHandler.RegisterRoutes(protected)
	r.integrationHandler.RegisterRoutes(protected)
	r.invoiceHandler.RegisterRoutes(protected)
	r.invoiceAdjustmentHandler.RegisterRoutes(protected)
	r.billingControlHandler.RegisterRoutes(protected)
	r.invoiceAdjustmentControlHandler.RegisterRoutes(protected)
	r.billingQueueHandler.RegisterRoutes(protected)
	r.dataEntryControlHandler.RegisterRoutes(protected)
	r.dispatchControlHandler.RegisterRoutes(protected)
	r.documentControlHandler.RegisterRoutes(protected)
	r.documentParsingRuleHandler.RegisterRoutes(protected)
	r.distanceOverrideHandler.RegisterRoutes(protected)
	r.workerPTOHandler.RegisterRoutes(protected)
	r.analyticsHandler.RegisterRoutes(protected)
	r.apiKeyHandler.RegisterRoutes(protected)
	r.tableChangeAlertHandler.RegisterRoutes(protected)
	r.notificationHandler.RegisterRoutes(protected)
	r.documentPacketRuleHandler.RegisterRoutes(protected)
}
