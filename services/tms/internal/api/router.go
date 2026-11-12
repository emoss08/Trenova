//revive:disable-next-line:var-naming
package api

import (
	"time"

	graphqlapi "github.com/emoss08/trenova/internal/api/graphql"
	"github.com/emoss08/trenova/internal/api/handlers/accountingcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/accountingwebhookhandler"
	"github.com/emoss08/trenova/internal/api/handlers/accountsreceivablehandler"
	"github.com/emoss08/trenova/internal/api/handlers/agentanalysishandler"
	"github.com/emoss08/trenova/internal/api/handlers/agentdefinitionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/agentextensionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/agentrunhandler"
	"github.com/emoss08/trenova/internal/api/handlers/aiproviderhandler"
	"github.com/emoss08/trenova/internal/api/handlers/analyticshandler"
	"github.com/emoss08/trenova/internal/api/handlers/apikeyhandler"
	"github.com/emoss08/trenova/internal/api/handlers/assignmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/assistanthandler"
	"github.com/emoss08/trenova/internal/api/handlers/authhandler"
	"github.com/emoss08/trenova/internal/api/handlers/bankreceiptbatchhandler"
	"github.com/emoss08/trenova/internal/api/handlers/bankreceipthandler"
	"github.com/emoss08/trenova/internal/api/handlers/bankreceiptworkitemhandler"
	"github.com/emoss08/trenova/internal/api/handlers/billingcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/billingqueuehandler"
	"github.com/emoss08/trenova/internal/api/handlers/capturehandler"
	"github.com/emoss08/trenova/internal/api/handlers/carrierassignmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/carrierhandler"
	"github.com/emoss08/trenova/internal/api/handlers/carriertrackingpublichandler"
	"github.com/emoss08/trenova/internal/api/handlers/casechecklisthandler"
	"github.com/emoss08/trenova/internal/api/handlers/collectionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/customerstatementhandler"
	"github.com/emoss08/trenova/internal/api/handlers/customfieldhandler"
	"github.com/emoss08/trenova/internal/api/handlers/dataentrycontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/dataretentionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/detentionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/dispatchcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/distancecontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/distanceoverridehandler"
	"github.com/emoss08/trenova/internal/api/handlers/distanceprofilehandler"
	"github.com/emoss08/trenova/internal/api/handlers/documentcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/documenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/documentpacketrulehandler"
	"github.com/emoss08/trenova/internal/api/handlers/documenttemplatehandler"
	"github.com/emoss08/trenova/internal/api/handlers/driverportalhandler"
	"github.com/emoss08/trenova/internal/api/handlers/edihandler"
	"github.com/emoss08/trenova/internal/api/handlers/ediladinghandler"
	"github.com/emoss08/trenova/internal/api/handlers/edisetuphandler"
	"github.com/emoss08/trenova/internal/api/handlers/editransferhandler"
	"github.com/emoss08/trenova/internal/api/handlers/emailhandler"
	"github.com/emoss08/trenova/internal/api/handlers/exchangeratehandler"
	"github.com/emoss08/trenova/internal/api/handlers/fiscalperiodhandler"
	"github.com/emoss08/trenova/internal/api/handlers/fiscalyearhandler"
	"github.com/emoss08/trenova/internal/api/handlers/formulalibraryhandler"
	"github.com/emoss08/trenova/internal/api/handlers/formulatemplatehandler"
	"github.com/emoss08/trenova/internal/api/handlers/glbalancehandler"
	"github.com/emoss08/trenova/internal/api/handlers/googlemapshandler"
	"github.com/emoss08/trenova/internal/api/handlers/iamhandler"
	"github.com/emoss08/trenova/internal/api/handlers/inboundhandler"
	"github.com/emoss08/trenova/internal/api/handlers/inboxpresencehandler"
	"github.com/emoss08/trenova/internal/api/handlers/insighthandler"
	"github.com/emoss08/trenova/internal/api/handlers/integrationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/invoiceadjustmentcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/invoiceadjustmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/invoicehandler"
	"github.com/emoss08/trenova/internal/api/handlers/invoicerunhandler"
	"github.com/emoss08/trenova/internal/api/handlers/journalreversalhandler"
	"github.com/emoss08/trenova/internal/api/handlers/jurisdictionrulehandler"
	"github.com/emoss08/trenova/internal/api/handlers/locationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/manualjournalhandler"
	"github.com/emoss08/trenova/internal/api/handlers/notificationpreferencehandler"
	"github.com/emoss08/trenova/internal/api/handlers/organizationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/pagefavoritehandler"
	"github.com/emoss08/trenova/internal/api/handlers/palettehandler"
	"github.com/emoss08/trenova/internal/api/handlers/permissionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/permithandler"
	"github.com/emoss08/trenova/internal/api/handlers/publicconfighandler"
	"github.com/emoss08/trenova/internal/api/handlers/pushhandler"
	"github.com/emoss08/trenova/internal/api/handlers/rateagreementhandler"
	"github.com/emoss08/trenova/internal/api/handlers/rateconfirmationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/rateconfirmationpublichandler"
	"github.com/emoss08/trenova/internal/api/handlers/rateimporthandler"
	"github.com/emoss08/trenova/internal/api/handlers/ratematrixhandler"
	"github.com/emoss08/trenova/internal/api/handlers/ratequotehandler"
	"github.com/emoss08/trenova/internal/api/handlers/ratesimulationhandler"
	"github.com/emoss08/trenova/internal/api/handlers/ratezonehandler"
	"github.com/emoss08/trenova/internal/api/handlers/realtimehandler"
	"github.com/emoss08/trenova/internal/api/handlers/recurringshipmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/reporthandler"
	"github.com/emoss08/trenova/internal/api/handlers/rolehandler"
	"github.com/emoss08/trenova/internal/api/handlers/routingguidehandler"
	"github.com/emoss08/trenova/internal/api/handlers/scimhandler"
	"github.com/emoss08/trenova/internal/api/handlers/searchhandler"
	"github.com/emoss08/trenova/internal/api/handlers/sequenceconfighandler"
	"github.com/emoss08/trenova/internal/api/handlers/servicefailurehandler"
	"github.com/emoss08/trenova/internal/api/handlers/servicefailurereasoncodehandler"
	"github.com/emoss08/trenova/internal/api/handlers/servicetypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmentcontrolhandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmenthandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmentmovehandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmenttrackingpublichandler"
	"github.com/emoss08/trenova/internal/api/handlers/shipmenttypehandler"
	"github.com/emoss08/trenova/internal/api/handlers/storedmileagehandler"
	"github.com/emoss08/trenova/internal/api/handlers/tablechangealerthandler"
	"github.com/emoss08/trenova/internal/api/handlers/tablequeryhandler"
	"github.com/emoss08/trenova/internal/api/handlers/telematicshandler"
	"github.com/emoss08/trenova/internal/api/handlers/tenderhandler"
	"github.com/emoss08/trenova/internal/api/handlers/tenderpublichandler"
	"github.com/emoss08/trenova/internal/api/handlers/tractorhandler"
	"github.com/emoss08/trenova/internal/api/handlers/trailerhandler"
	"github.com/emoss08/trenova/internal/api/handlers/userhandler"
	"github.com/emoss08/trenova/internal/api/handlers/versionhandler"
	"github.com/emoss08/trenova/internal/api/handlers/weatheralerthandler"
	"github.com/emoss08/trenova/internal/api/handlers/workerhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/responseguard"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/serverconfig"
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
	AppConfig               *appconfig.Config
	SecurityConfig          *securityconfig.Config
	ServerConfig            *serverconfig.Config
	Logger                  *zap.Logger
	ObservabilityMiddleware *observability.Middleware

	AuthMiddleware                  *middleware.AuthMiddleware
	ReadScopeMiddleware             *middleware.ReadScopeMiddleware
	QueryBudgetMiddleware           *middleware.QueryBudgetMiddleware
	ReadOnlyGuard                   *middleware.ReadOnlyGuard
	RateLimiter                     *middleware.RateLimiter
	IdempotencyMiddleware           *middleware.IdempotencyMiddleware
	PermissionMiddleware            *middleware.PermissionMiddleware
	TenantBoundaryMiddleware        *middleware.TenantBoundaryMiddleware
	ResponseGuard                   *responseguard.Guard
	ErrorHandler                    *helpers.ErrorHandler
	OrganizationHandler             *organizationhandler.Handler
	DataRetentionHandler            *dataretentionhandler.Handler
	IAMHandler                      *iamhandler.Handler
	SCIMHandler                     *scimhandler.Handler
	UserHandler                     *userhandler.Handler
	AuthHandler                     *authhandler.Handler
	PublicConfigHandler             *publicconfighandler.Handler
	DriverPortalHandler             *driverportalhandler.Handler
	PushHandler                     *pushhandler.Handler
	NotificationPreferenceHandler   *notificationpreferencehandler.Handler
	BankReceiptHandler              *bankreceipthandler.Handler
	BankReceiptBatchHandler         *bankreceiptbatchhandler.Handler
	BankReceiptWorkItemHandler      *bankreceiptworkitemhandler.Handler
	FormulaTemplateHandler          *formulatemplatehandler.Handler
	FormulaLibraryHandler           *formulalibraryhandler.Handler
	PageFavoriteHandler             *pagefavoritehandler.Handler
	TractorHandler                  *tractorhandler.Handler
	TrailerHandler                  *trailerhandler.Handler
	GraphQLHandler                  *graphqlapi.Handler
	WorkerHandler                   *workerhandler.Handler
	PermissionHandler               *permissionhandler.Handler
	RealtimeHandler                 *realtimehandler.Handler
	InboxPresenceHandler            *inboxpresencehandler.Handler
	RoleHandler                     *rolehandler.Handler
	SearchHandler                   *searchhandler.Handler
	CustomFieldHandler              *customfieldhandler.Handler
	DocumentHandler                 *documenthandler.Handler
	ReportHandler                   *reporthandler.Handler
	TableQueryHandler               *tablequeryhandler.Handler
	AIProviderHandler               *aiproviderhandler.Handler
	AgentExtensionHandler           *agentextensionhandler.Handler
	AgentAnalysisHandler            *agentanalysishandler.Handler
	AgentDefinitionHandler          *agentdefinitionhandler.Handler
	AssistantHandler                *assistanthandler.Handler
	InsightHandler                  *insighthandler.Handler
	AgentRunHandler                 *agentrunhandler.Handler
	VersionHandler                  *versionhandler.Handler
	WeatherAlertHandler             *weatheralerthandler.Handler
	ServiceTypeHandler              *servicetypehandler.Handler
	ServiceFailureReasonCodeHandler *servicefailurereasoncodehandler.Handler
	ServiceFailureHandler           *servicefailurehandler.Handler
	SequenceConfigHandler           *sequenceconfighandler.Handler
	ShipmentControlHandler          *shipmentcontrolhandler.Handler
	ShipmentMoveHandler             *shipmentmovehandler.Handler
	ShipmentHandler                 *shipmenthandler.Handler
	PermitHandler                   *permithandler.Handler
	JurisdictionRuleHandler         *jurisdictionrulehandler.Handler
	ShipmentTypeHandler             *shipmenttypehandler.Handler
	EDIHandler                      *edihandler.Handler
	EDISetupHandler                 *edisetuphandler.Handler
	EDITransferHandler              *editransferhandler.Handler
	EDILadingHandler                *ediladinghandler.Handler
	InboundHandler                  *inboundhandler.Handler
	CaptureHandler                  *capturehandler.Handler
	EmailHandler                    *emailhandler.Handler
	TelematicsHandler               *telematicshandler.Handler
	AccountingWebhookHandler        *accountingwebhookhandler.Handler
	CarrierAssignmentHandler        *carrierassignmenthandler.Handler
	TenderPublicHandler             *tenderpublichandler.Handler
	TenderHandler                   *tenderhandler.Handler
	RoutingGuideHandler             *routingguidehandler.Handler
	CarrierHandler                  *carrierhandler.Handler
	RateConfirmationHandler         *rateconfirmationhandler.Handler
	RateConfirmationPublicHandler   *rateconfirmationpublichandler.Handler
	CarrierTrackingPublicHandler    *carriertrackingpublichandler.Handler
	ShipmentTrackingPublicHandler   *shipmenttrackingpublichandler.Handler
	GoogleMapsHandler               *googlemapshandler.Handler
	AccountingControlHandler        *accountingcontrolhandler.Handler
	AccountsReceivableHandler       *accountsreceivablehandler.Handler
	CollectionHandler               *collectionhandler.Handler
	CustomerStatementHandler        *customerstatementhandler.Handler
	AssignmentHandler               *assignmenthandler.Handler
	GLBalanceHandler                *glbalancehandler.Handler
	FiscalYearHandler               *fiscalyearhandler.Handler
	FiscalPeriodHandler             *fiscalperiodhandler.Handler
	LocationHandler                 *locationhandler.Handler
	RecurringShipmentHandler        *recurringshipmenthandler.Handler
	RateAgreementHandler            *rateagreementhandler.Handler
	RateZoneHandler                 *ratezonehandler.Handler
	RateMatrixHandler               *ratematrixhandler.Handler
	RateSimulationHandler           *ratesimulationhandler.Handler
	RateImportHandler               *rateimporthandler.Handler
	RateQuoteHandler                *ratequotehandler.Handler
	DetentionHandler                *detentionhandler.Handler
	IntegrationHandler              *integrationhandler.Handler
	InvoiceHandler                  *invoicehandler.Handler
	InvoiceRunHandler               *invoicerunhandler.Handler
	InvoiceAdjustmentHandler        *invoiceadjustmenthandler.Handler
	JournalReversalHandler          *journalreversalhandler.Handler
	ManualJournalHandler            *manualjournalhandler.Handler
	BillingControlHandler           *billingcontrolhandler.Handler
	CaseChecklistHandler            *casechecklisthandler.Handler
	InvoiceAdjustmentControlHandler *invoiceadjustmentcontrolhandler.Handler
	BillingQueueHandler             *billingqueuehandler.Handler
	PaletteHandler                  *palettehandler.Handler
	DataEntryControlHandler         *dataentrycontrolhandler.Handler
	DispatchControlHandler          *dispatchcontrolhandler.Handler
	DocumentControlHandler          *documentcontrolhandler.Handler
	ExchangeRateHandler             *exchangeratehandler.Handler
	DistanceControlHandler          *distancecontrolhandler.Handler
	DistanceOverrideHandler         *distanceoverridehandler.Handler
	DistanceProfileHandler          *distanceprofilehandler.Handler
	StoredMileageHandler            *storedmileagehandler.Handler
	AnalyticsHandler                *analyticshandler.Handler
	ApiKeyHandler                   *apikeyhandler.Handler //nolint:revive // field name follows existing router wiring
	TableChangeAlertHandler         *tablechangealerthandler.Handler
	DocumentPacketRuleHandler       *documentpacketrulehandler.Handler
	DocumentTemplateHandler         *documenttemplatehandler.Handler
}

type Router struct {
	s                               *Server
	l                               *zap.Logger
	observabilityMiddleware         *observability.Middleware
	authMiddleware                  *middleware.AuthMiddleware
	readScopeMiddleware             *middleware.ReadScopeMiddleware
	queryBudgetMiddleware           *middleware.QueryBudgetMiddleware
	readOnlyGuard                   *middleware.ReadOnlyGuard
	rateLimiter                     *middleware.RateLimiter
	idempotencyMiddleware           *middleware.IdempotencyMiddleware
	permissionMiddleware            *middleware.PermissionMiddleware
	tenantBoundaryMiddleware        *middleware.TenantBoundaryMiddleware
	responseGuard                   *responseguard.Guard
	appConfig                       *appconfig.Config
	securityConfig                  *securityconfig.Config
	serverConfig                    *serverconfig.Config
	errorHandler                    *helpers.ErrorHandler
	organizationHandler             *organizationhandler.Handler
	dataRetentionHandler            *dataretentionhandler.Handler
	iamHandler                      *iamhandler.Handler
	scimHandler                     *scimhandler.Handler
	userHandler                     *userhandler.Handler
	authHandler                     *authhandler.Handler
	publicConfigHandler             *publicconfighandler.Handler
	driverPortalHandler             *driverportalhandler.Handler
	pushHandler                     *pushhandler.Handler
	notificationPreferenceHandler   *notificationpreferencehandler.Handler
	bankReceiptHandler              *bankreceipthandler.Handler
	bankReceiptBatchHandler         *bankreceiptbatchhandler.Handler
	bankReceiptWorkItemHandler      *bankreceiptworkitemhandler.Handler
	formulaTemplateHandler          *formulatemplatehandler.Handler
	formulaLibraryHandler           *formulalibraryhandler.Handler
	pageFavoriteHandler             *pagefavoritehandler.Handler
	serviceTypeHandler              *servicetypehandler.Handler
	serviceFailureReasonCodeHandler *servicefailurereasoncodehandler.Handler
	serviceFailureHandler           *servicefailurehandler.Handler
	sequenceConfigHandler           *sequenceconfighandler.Handler
	shipmentControlHandler          *shipmentcontrolhandler.Handler
	shipmentMoveHandler             *shipmentmovehandler.Handler
	shipmentHandler                 *shipmenthandler.Handler
	permitHandler                   *permithandler.Handler
	jurisdictionRuleHandler         *jurisdictionrulehandler.Handler
	tractorHandler                  *tractorhandler.Handler
	trailerHandler                  *trailerhandler.Handler
	graphQLHandler                  *graphqlapi.Handler
	workerHandler                   *workerhandler.Handler
	permissionHandler               *permissionhandler.Handler
	realtimeHandler                 *realtimehandler.Handler
	inboxPresenceHandler            *inboxpresencehandler.Handler
	roleHandler                     *rolehandler.Handler
	searchHandler                   *searchhandler.Handler
	customFieldHandler              *customfieldhandler.Handler
	documentHandler                 *documenthandler.Handler
	reportHandler                   *reporthandler.Handler
	tableQueryHandler               *tablequeryhandler.Handler
	aiProviderHandler               *aiproviderhandler.Handler
	agentExtensionHandler           *agentextensionhandler.Handler
	agentAnalysisHandler            *agentanalysishandler.Handler
	agentDefinitionHandler          *agentdefinitionhandler.Handler
	assistantHandler                *assistanthandler.Handler
	insightHandler                  *insighthandler.Handler
	agentRunHandler                 *agentrunhandler.Handler
	versionHandler                  *versionhandler.Handler
	weatherAlertHandler             *weatheralerthandler.Handler
	shipmentTypeHandler             *shipmenttypehandler.Handler
	ediHandler                      *edihandler.Handler
	ediSetupHandler                 *edisetuphandler.Handler
	ediTransferHandler              *editransferhandler.Handler
	ediLadingHandler                *ediladinghandler.Handler
	inboundHandler                  *inboundhandler.Handler
	captureHandler                  *capturehandler.Handler
	emailHandler                    *emailhandler.Handler
	telematicsHandler               *telematicshandler.Handler
	accountingWebhookHandler        *accountingwebhookhandler.Handler
	carrierAssignmentHandler        *carrierassignmenthandler.Handler
	tenderPublicHandler             *tenderpublichandler.Handler
	tenderHandler                   *tenderhandler.Handler
	routingGuideHandler             *routingguidehandler.Handler
	carrierHandler                  *carrierhandler.Handler
	rateConfirmationHandler         *rateconfirmationhandler.Handler
	rateConfirmationPublicHandler   *rateconfirmationpublichandler.Handler
	carrierTrackingPublicHandler    *carriertrackingpublichandler.Handler
	shipmentTrackingPublicHandler   *shipmenttrackingpublichandler.Handler
	googleMapsHandler               *googlemapshandler.Handler
	accountingControlHandler        *accountingcontrolhandler.Handler
	accountsReceivableHandler       *accountsreceivablehandler.Handler
	collectionHandler               *collectionhandler.Handler
	customerStatementHandler        *customerstatementhandler.Handler
	assignmentHandler               *assignmenthandler.Handler
	glBalanceHandler                *glbalancehandler.Handler
	fiscalYearHandler               *fiscalyearhandler.Handler
	fiscalPeriodHandler             *fiscalperiodhandler.Handler
	locationHandler                 *locationhandler.Handler
	recurringShipmentHandler        *recurringshipmenthandler.Handler
	rateAgreementHandler            *rateagreementhandler.Handler
	rateZoneHandler                 *ratezonehandler.Handler
	rateMatrixHandler               *ratematrixhandler.Handler
	rateSimulationHandler           *ratesimulationhandler.Handler
	rateImportHandler               *rateimporthandler.Handler
	rateQuoteHandler                *ratequotehandler.Handler
	detentionHandler                *detentionhandler.Handler
	integrationHandler              *integrationhandler.Handler
	invoiceHandler                  *invoicehandler.Handler
	invoiceRunHandler               *invoicerunhandler.Handler
	invoiceAdjustmentHandler        *invoiceadjustmenthandler.Handler
	journalReversalHandler          *journalreversalhandler.Handler
	manualJournalHandler            *manualjournalhandler.Handler
	billingControlHandler           *billingcontrolhandler.Handler
	caseChecklistHandler            *casechecklisthandler.Handler
	invoiceAdjustmentControlHandler *invoiceadjustmentcontrolhandler.Handler
	billingQueueHandler             *billingqueuehandler.Handler
	paletteHandler                  *palettehandler.Handler
	dataEntryControlHandler         *dataentrycontrolhandler.Handler
	dispatchControlHandler          *dispatchcontrolhandler.Handler
	documentControlHandler          *documentcontrolhandler.Handler
	exchangeRateHandler             *exchangeratehandler.Handler
	distanceControlHandler          *distancecontrolhandler.Handler
	distanceOverrideHandler         *distanceoverridehandler.Handler
	distanceProfileHandler          *distanceprofilehandler.Handler
	storedMileageHandler            *storedmileagehandler.Handler
	analyticsHandler                *analyticshandler.Handler
	apiKeyHandler                   *apikeyhandler.Handler
	tableChangeAlertHandler         *tablechangealerthandler.Handler
	documentPacketRuleHandler       *documentpacketrulehandler.Handler
	documentTemplateHandler         *documenttemplatehandler.Handler
}

//nolint:gocritic // This is a constructor
func NewRouter(p RouterParams) *Router {
	return &Router{
		s:                               p.Server,
		appConfig:                       p.AppConfig,
		securityConfig:                  p.SecurityConfig,
		serverConfig:                    p.ServerConfig,
		l:                               p.Logger,
		observabilityMiddleware:         p.ObservabilityMiddleware,
		authMiddleware:                  p.AuthMiddleware,
		readScopeMiddleware:             p.ReadScopeMiddleware,
		queryBudgetMiddleware:           p.QueryBudgetMiddleware,
		readOnlyGuard:                   p.ReadOnlyGuard,
		rateLimiter:                     p.RateLimiter,
		idempotencyMiddleware:           p.IdempotencyMiddleware,
		permissionMiddleware:            p.PermissionMiddleware,
		tenantBoundaryMiddleware:        p.TenantBoundaryMiddleware,
		responseGuard:                   p.ResponseGuard,
		errorHandler:                    p.ErrorHandler,
		organizationHandler:             p.OrganizationHandler,
		dataRetentionHandler:            p.DataRetentionHandler,
		iamHandler:                      p.IAMHandler,
		scimHandler:                     p.SCIMHandler,
		userHandler:                     p.UserHandler,
		authHandler:                     p.AuthHandler,
		publicConfigHandler:             p.PublicConfigHandler,
		driverPortalHandler:             p.DriverPortalHandler,
		pushHandler:                     p.PushHandler,
		notificationPreferenceHandler:   p.NotificationPreferenceHandler,
		bankReceiptHandler:              p.BankReceiptHandler,
		bankReceiptBatchHandler:         p.BankReceiptBatchHandler,
		bankReceiptWorkItemHandler:      p.BankReceiptWorkItemHandler,
		formulaTemplateHandler:          p.FormulaTemplateHandler,
		formulaLibraryHandler:           p.FormulaLibraryHandler,
		pageFavoriteHandler:             p.PageFavoriteHandler,
		serviceTypeHandler:              p.ServiceTypeHandler,
		serviceFailureReasonCodeHandler: p.ServiceFailureReasonCodeHandler,
		serviceFailureHandler:           p.ServiceFailureHandler,
		sequenceConfigHandler:           p.SequenceConfigHandler,
		shipmentControlHandler:          p.ShipmentControlHandler,
		shipmentMoveHandler:             p.ShipmentMoveHandler,
		shipmentHandler:                 p.ShipmentHandler,
		permitHandler:                   p.PermitHandler,
		jurisdictionRuleHandler:         p.JurisdictionRuleHandler,
		tractorHandler:                  p.TractorHandler,
		trailerHandler:                  p.TrailerHandler,
		graphQLHandler:                  p.GraphQLHandler,
		workerHandler:                   p.WorkerHandler,
		permissionHandler:               p.PermissionHandler,
		realtimeHandler:                 p.RealtimeHandler,
		inboxPresenceHandler:            p.InboxPresenceHandler,
		roleHandler:                     p.RoleHandler,
		searchHandler:                   p.SearchHandler,
		customFieldHandler:              p.CustomFieldHandler,
		documentHandler:                 p.DocumentHandler,
		reportHandler:                   p.ReportHandler,
		tableQueryHandler:               p.TableQueryHandler,
		aiProviderHandler:               p.AIProviderHandler,
		agentExtensionHandler:           p.AgentExtensionHandler,
		agentAnalysisHandler:            p.AgentAnalysisHandler,
		agentDefinitionHandler:          p.AgentDefinitionHandler,
		assistantHandler:                p.AssistantHandler,
		insightHandler:                  p.InsightHandler,
		agentRunHandler:                 p.AgentRunHandler,
		versionHandler:                  p.VersionHandler,
		weatherAlertHandler:             p.WeatherAlertHandler,
		shipmentTypeHandler:             p.ShipmentTypeHandler,
		ediHandler:                      p.EDIHandler,
		ediSetupHandler:                 p.EDISetupHandler,
		ediTransferHandler:              p.EDITransferHandler,
		ediLadingHandler:                p.EDILadingHandler,
		inboundHandler:                  p.InboundHandler,
		captureHandler:                  p.CaptureHandler,
		emailHandler:                    p.EmailHandler,
		telematicsHandler:               p.TelematicsHandler,
		accountingWebhookHandler:        p.AccountingWebhookHandler,
		carrierAssignmentHandler:        p.CarrierAssignmentHandler,
		tenderPublicHandler:             p.TenderPublicHandler,
		tenderHandler:                   p.TenderHandler,
		routingGuideHandler:             p.RoutingGuideHandler,
		carrierHandler:                  p.CarrierHandler,
		rateConfirmationHandler:         p.RateConfirmationHandler,
		rateConfirmationPublicHandler:   p.RateConfirmationPublicHandler,
		carrierTrackingPublicHandler:    p.CarrierTrackingPublicHandler,
		shipmentTrackingPublicHandler:   p.ShipmentTrackingPublicHandler,
		googleMapsHandler:               p.GoogleMapsHandler,
		accountingControlHandler:        p.AccountingControlHandler,
		accountsReceivableHandler:       p.AccountsReceivableHandler,
		collectionHandler:               p.CollectionHandler,
		customerStatementHandler:        p.CustomerStatementHandler,
		assignmentHandler:               p.AssignmentHandler,
		glBalanceHandler:                p.GLBalanceHandler,
		fiscalYearHandler:               p.FiscalYearHandler,
		fiscalPeriodHandler:             p.FiscalPeriodHandler,
		locationHandler:                 p.LocationHandler,
		recurringShipmentHandler:        p.RecurringShipmentHandler,
		rateAgreementHandler:            p.RateAgreementHandler,
		rateZoneHandler:                 p.RateZoneHandler,
		rateMatrixHandler:               p.RateMatrixHandler,
		rateSimulationHandler:           p.RateSimulationHandler,
		rateImportHandler:               p.RateImportHandler,
		rateQuoteHandler:                p.RateQuoteHandler,
		detentionHandler:                p.DetentionHandler,
		integrationHandler:              p.IntegrationHandler,
		invoiceHandler:                  p.InvoiceHandler,
		invoiceRunHandler:               p.InvoiceRunHandler,
		invoiceAdjustmentHandler:        p.InvoiceAdjustmentHandler,
		journalReversalHandler:          p.JournalReversalHandler,
		manualJournalHandler:            p.ManualJournalHandler,
		billingControlHandler:           p.BillingControlHandler,
		caseChecklistHandler:            p.CaseChecklistHandler,
		invoiceAdjustmentControlHandler: p.InvoiceAdjustmentControlHandler,
		billingQueueHandler:             p.BillingQueueHandler,
		paletteHandler:                  p.PaletteHandler,
		dataEntryControlHandler:         p.DataEntryControlHandler,
		dispatchControlHandler:          p.DispatchControlHandler,
		documentControlHandler:          p.DocumentControlHandler,
		exchangeRateHandler:             p.ExchangeRateHandler,
		distanceControlHandler:          p.DistanceControlHandler,
		distanceOverrideHandler:         p.DistanceOverrideHandler,
		distanceProfileHandler:          p.DistanceProfileHandler,
		storedMileageHandler:            p.StoredMileageHandler,
		analyticsHandler:                p.AnalyticsHandler,
		apiKeyHandler:                   p.ApiKeyHandler,
		tableChangeAlertHandler:         p.TableChangeAlertHandler,
		documentPacketRuleHandler:       p.DocumentPacketRuleHandler,
		documentTemplateHandler:         p.DocumentTemplateHandler,
	}
}

func (r *Router) setupCors() {
	if !r.serverConfig.CORS.Enabled {
		return
	}

	corsConfig := cors.Config{
		AllowOrigins:     r.serverConfig.CORS.AllowedOrigins,
		AllowMethods:     r.serverConfig.CORS.AllowedMethods,
		AllowHeaders:     middleware.WithSessionContextHeaders(r.serverConfig.CORS.AllowedHeaders),
		ExposeHeaders:    r.serverConfig.CORS.ExposeHeaders,
		AllowCredentials: r.serverConfig.CORS.Credentials,
		MaxAge:           time.Duration(r.serverConfig.CORS.MaxAge) * time.Second,
	}

	// If AllowOrigins is empty or contains "*", allow all origins with credentials disabled
	if len(corsConfig.AllowOrigins) == 0 ||
		(len(corsConfig.AllowOrigins) == 1 && corsConfig.AllowOrigins[0] == "*") {
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowOrigins = nil
		corsConfig.AllowCredentials = false
	}

	r.s.router.Use(cors.New(corsConfig))
	r.s.router.Use(middleware.NewVaryOriginMiddleware())
}

func (r *Router) Setup() {
	r.setupMiddleware()
	r.setupRoutes()
}

func (r *Router) setupMiddleware() {
	r.setupCors()

	r.s.router.Use(middleware.NewSecurityHeadersMiddleware(r.appConfig))
	r.s.router.Use(gin.Recovery())
	r.s.router.Use(middleware.NewMultipartCleanupMiddleware(r.l))
	r.s.router.Use(requestid.New())
	r.s.router.Use(middleware.NewRequestMetaMiddleware())
	r.s.router.Use(r.tenantBoundaryMiddleware.Track())
	r.s.router.Use(
		middleware.NewCSRFBrowserGuard(r.securityConfig, r.serverConfig, r.errorHandler, r.l).
			Guard(),
	)
	r.s.router.Use(responseCompression())
	r.s.router.Use(middleware.NewLocaleMiddleware().Resolve())
	r.s.router.Use(middleware.NewTokenRedactionMiddleware())
	r.s.router.Use(ginzap.Ginzap(r.l, time.RFC3339, true))
	r.s.router.Use(r.observabilityMiddleware.TracingMiddleware())
}

func (r *Router) setupRoutes() {
	r.setupGraphQLRoutes(r.s.router.Group(""))

	v1 := r.s.router.Group("/api/v1")
	v1.Use(r.queryBudgetMiddleware.Install())
	r.setupProtectedRoutes(v1)
	r.setupPublicRoutes(v1)
	r.scimHandler.RegisterRoutes(v1)
}

func (r *Router) setupGraphQLRoutes(rg *gin.RouterGroup) {
	protected := r.protectedGroup(rg)
	r.graphQLHandler.RegisterRoutes(protected)
}

func (r *Router) setupPublicRoutes(parent *gin.RouterGroup) {
	rg := r.publicGroup(parent)

	r.authHandler.RegisterRoutes(rg)
	r.publicConfigHandler.RegisterPublicRoutes(rg)
	r.driverPortalHandler.RegisterRoutes(rg)
	r.versionHandler.RegisterPublicRoutes(rg)
	r.emailHandler.RegisterPublicRoutes(rg)
	r.telematicsHandler.RegisterPublicRoutes(rg)
	r.accountingWebhookHandler.RegisterPublicRoutes(rg)
	r.agentExtensionHandler.RegisterPublicRoutes(rg)
	r.invoiceHandler.RegisterPublicRoutes(rg)
	r.ediHandler.RegisterPublicRoutes(rg)
	r.inboundHandler.RegisterPublicRoutes(rg)
	r.captureHandler.RegisterPublicRoutes(rg)
	r.captureHandler.RegisterDeviceRoutes(r.captureDeviceGroup(parent))
	r.tenderPublicHandler.RegisterPublicRoutes(rg)
	r.rateConfirmationPublicHandler.RegisterPublicRoutes(rg)
	r.carrierTrackingPublicHandler.RegisterPublicRoutes(rg)
	r.shipmentTrackingPublicHandler.RegisterPublicRoutes(rg)
}

//nolint:funlen // existing workflow or route registration is intentionally kept together
func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	protected := r.protectedGroup(rg)

	r.driverPortalHandler.RegisterProtectedRoutes(protected)
	r.pushHandler.RegisterRoutes(protected)
	r.notificationPreferenceHandler.RegisterRoutes(protected)
	r.organizationHandler.RegisterRoutes(protected)
	r.dataRetentionHandler.RegisterRoutes(protected)
	r.iamHandler.RegisterRoutes(protected)
	r.userHandler.RegisterRoutes(protected)
	r.bankReceiptBatchHandler.RegisterRoutes(protected)
	r.bankReceiptHandler.RegisterRoutes(protected)
	r.bankReceiptWorkItemHandler.RegisterRoutes(protected)
	r.formulaTemplateHandler.RegisterRoutes(protected)
	r.formulaLibraryHandler.RegisterRoutes(protected)
	r.pageFavoriteHandler.RegisterRoutes(protected)
	r.tractorHandler.RegisterRoutes(protected)
	r.trailerHandler.RegisterRoutes(protected)
	r.workerHandler.RegisterRoutes(protected)
	r.permissionHandler.RegisterRoutes(protected)
	r.realtimeHandler.RegisterRoutes(protected)
	r.inboxPresenceHandler.RegisterRoutes(protected)
	r.roleHandler.RegisterRoutes(protected)
	r.searchHandler.RegisterRoutes(protected)
	r.customFieldHandler.RegisterRoutes(protected)
	r.documentHandler.RegisterRoutes(protected)
	r.reportHandler.RegisterRoutes(protected)
	r.tableQueryHandler.RegisterRoutes(protected)
	r.aiProviderHandler.RegisterRoutes(protected)
	r.agentExtensionHandler.RegisterRoutes(protected)
	r.agentAnalysisHandler.RegisterRoutes(protected)
	r.agentDefinitionHandler.RegisterRoutes(protected)
	r.assistantHandler.RegisterRoutes(protected)
	r.insightHandler.RegisterRoutes(protected)
	r.agentRunHandler.RegisterRoutes(protected)
	r.serviceTypeHandler.RegisterRoutes(protected)
	r.serviceFailureReasonCodeHandler.RegisterRoutes(protected)
	r.serviceFailureHandler.RegisterRoutes(protected)
	r.sequenceConfigHandler.RegisterRoutes(protected)
	r.shipmentControlHandler.RegisterRoutes(protected)
	r.shipmentMoveHandler.RegisterRoutes(protected)
	r.shipmentHandler.RegisterRoutes(protected)
	r.permitHandler.RegisterRoutes(protected)
	r.jurisdictionRuleHandler.RegisterRoutes(protected)
	r.shipmentTypeHandler.RegisterRoutes(protected)
	r.ediHandler.RegisterRoutes(protected)
	r.ediSetupHandler.RegisterRoutes(protected)
	r.ediTransferHandler.RegisterRoutes(protected)
	r.ediLadingHandler.RegisterRoutes(protected)
	r.emailHandler.RegisterRoutes(protected)
	r.carrierAssignmentHandler.RegisterRoutes(protected)
	r.tenderHandler.RegisterRoutes(protected)
	r.routingGuideHandler.RegisterRoutes(protected)
	r.carrierHandler.RegisterRoutes(protected)
	r.rateConfirmationHandler.RegisterRoutes(protected)
	r.googleMapsHandler.RegisterRoutes(protected)
	r.weatherAlertHandler.RegisterRoutes(protected)
	r.accountingControlHandler.RegisterRoutes(protected)
	r.accountsReceivableHandler.RegisterRoutes(protected)
	r.collectionHandler.RegisterRoutes(protected)
	r.customerStatementHandler.RegisterRoutes(protected)
	r.assignmentHandler.RegisterRoutes(protected)
	r.glBalanceHandler.RegisterRoutes(protected)
	r.fiscalYearHandler.RegisterRoutes(protected)
	r.fiscalPeriodHandler.RegisterRoutes(protected)
	r.locationHandler.RegisterRoutes(protected)
	r.recurringShipmentHandler.RegisterRoutes(protected)
	r.rateAgreementHandler.RegisterRoutes(protected)
	r.rateZoneHandler.RegisterRoutes(protected)
	r.rateMatrixHandler.RegisterRoutes(protected)
	r.rateSimulationHandler.RegisterRoutes(protected)
	r.rateImportHandler.RegisterRoutes(protected)
	r.rateQuoteHandler.RegisterRoutes(protected)
	r.detentionHandler.RegisterRoutes(protected)
	r.integrationHandler.RegisterRoutes(protected)
	r.invoiceHandler.RegisterRoutes(protected)
	r.invoiceRunHandler.RegisterRoutes(protected)
	r.invoiceAdjustmentHandler.RegisterRoutes(protected)
	r.journalReversalHandler.RegisterRoutes(protected)
	r.manualJournalHandler.RegisterRoutes(protected)
	r.billingControlHandler.RegisterRoutes(protected)
	r.caseChecklistHandler.RegisterRoutes(protected)
	r.invoiceAdjustmentControlHandler.RegisterRoutes(protected)
	r.billingQueueHandler.RegisterRoutes(protected)
	r.paletteHandler.RegisterRoutes(protected)
	r.dataEntryControlHandler.RegisterRoutes(protected)
	r.dispatchControlHandler.RegisterRoutes(protected)
	r.documentControlHandler.RegisterRoutes(protected)
	r.distanceControlHandler.RegisterRoutes(protected)
	r.distanceOverrideHandler.RegisterRoutes(protected)
	r.distanceProfileHandler.RegisterRoutes(protected)
	r.storedMileageHandler.RegisterRoutes(protected)
	r.exchangeRateHandler.RegisterRoutes(protected)
	r.analyticsHandler.RegisterRoutes(protected)
	r.apiKeyHandler.RegisterRoutes(protected)
	r.tableChangeAlertHandler.RegisterRoutes(protected)
	r.documentPacketRuleHandler.RegisterRoutes(protected)
	r.documentTemplateHandler.RegisterRoutes(protected)
	r.captureHandler.RegisterRoutes(protected)
}

func (r *Router) protectedGroup(rg *gin.RouterGroup) *gin.RouterGroup {
	protected := rg.Group("")
	protected.Use(r.authMiddleware.RequireAuth())
	protected.Use(r.responseGuard.Install())
	protected.Use(r.responseGuard.InstallListGuard())
	protected.Use(r.readScopeMiddleware.Install())
	protected.Use(r.rateLimiter.ByAPIKey())
	protected.Use(middleware.NewLocaleMiddleware().Resolve())
	protected.Use(middleware.NewSessionContextMiddleware(r.errorHandler).RequireCurrent())
	protected.Use(middleware.NewCSRFMiddleware(r.securityConfig, r.errorHandler).RequireToken())
	// After RequireAuth, which is what puts the flag in the context, and before any
	// handler: a session that owes a password change may only reach the endpoints that
	// let it resolve one.
	protected.Use(middleware.NewPasswordChangeMiddleware(r.errorHandler).RequireCurrentPassword())
	protected.Use(r.readOnlyGuard.Guard())
	protected.Use(r.idempotencyMiddleware.Handle())
	return protected
}

// captureDeviceGroup is where a paired Trenova Capture device calls. It takes
// a device credential and nothing else, then applies the same entitlement
// checks a signed-in session gets. Like a session it is first-party and not
// request-rate limited: a scanned batch uploads one page per request. There is
// no CSRF check because nothing here is reachable with a cookie.
func (r *Router) captureDeviceGroup(rg *gin.RouterGroup) *gin.RouterGroup {
	device := rg.Group("")
	device.Use(r.captureHandler.RequireDevice())
	device.Use(r.readScopeMiddleware.Install())
	device.Use(r.readOnlyGuard.Guard())
	return device
}

func (r *Router) publicGroup(rg *gin.RouterGroup) *gin.RouterGroup {
	public := rg.Group("")
	public.Use(r.rateLimiter.ByClientIP(), middleware.NewPublicBodyLimit())
	return public
}

func responseCompression() gin.HandlerFunc {
	return gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedPaths([]string{"/metrics", "/health"}),
		gzip.WithExcludedPathsRegexs([]string{
			`^/api/v1/documents/[^/]+/(download|view|preview)/$`,
			// An event stream is flushed a frame at a time; compressing it
			// adds a buffer between each frame and the reader, which is what
			// turned an assistant reply into clumps of text.
			`^/api/v1/realtime/stream/$`,
			`^/api/v1/assistant/turns/[^/]+/stream/$`,
			`^/api/v1/capture/device/stream/$`,
			`^/api/v1/capture/pages/[^/]+/content/$`,
		}),
	)
}
