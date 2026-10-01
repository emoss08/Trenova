package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accessorialchargeservice"
	"github.com/emoss08/trenova/internal/core/services/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/core/services/accounttypeservice"
	"github.com/emoss08/trenova/internal/core/services/apikeyservice"
	"github.com/emoss08/trenova/internal/core/services/benefitsservice"
	"github.com/emoss08/trenova/internal/core/services/billingtransferservice"
	"github.com/emoss08/trenova/internal/core/services/capturereleaseservice"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/internal/core/services/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/services/carrierservice"
	"github.com/emoss08/trenova/internal/core/services/carriersettlementservice"
	"github.com/emoss08/trenova/internal/core/services/commodityservice"
	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/internal/core/services/customerservice"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/internal/core/services/dashcontrolservice"
	"github.com/emoss08/trenova/internal/core/services/detentionpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/distanceoverrideservice"
	"github.com/emoss08/trenova/internal/core/services/distanceprofileservice"
	"github.com/emoss08/trenova/internal/core/services/documentpacketruleservice"
	"github.com/emoss08/trenova/internal/core/services/documenttemplateservice"
	"github.com/emoss08/trenova/internal/core/services/documenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/driverpayservice"
	"github.com/emoss08/trenova/internal/core/services/driverportalservice"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/emailservice"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/core/services/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/fiscalyearservice"
	"github.com/emoss08/trenova/internal/core/services/fleetcodeservice"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/core/services/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/core/services/hazardousmaterialservice"
	"github.com/emoss08/trenova/internal/core/services/hazmatsegregationruleservice"
	"github.com/emoss08/trenova/internal/core/services/holdreasonservice"
	"github.com/emoss08/trenova/internal/core/services/homelayoutservice"
	"github.com/emoss08/trenova/internal/core/services/iftaservice"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/internal/core/services/journalentryservice"
	"github.com/emoss08/trenova/internal/core/services/journalreversalservice"
	"github.com/emoss08/trenova/internal/core/services/locationcategoryservice"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/core/services/manualjournalservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/core/services/orderservice"
	"github.com/emoss08/trenova/internal/core/services/orgholidayservice"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/internal/core/services/performancereviewservice"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/internal/core/services/ptopolicyservice"
	"github.com/emoss08/trenova/internal/core/services/rateagreementservice"
	"github.com/emoss08/trenova/internal/core/services/ratematrixservice"
	"github.com/emoss08/trenova/internal/core/services/ratequoteservice"
	"github.com/emoss08/trenova/internal/core/services/ratezoneservice"
	"github.com/emoss08/trenova/internal/core/services/recurringshipmentservice"
	reportingservice "github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/roleservice"
	"github.com/emoss08/trenova/internal/core/services/routingguideservice"
	"github.com/emoss08/trenova/internal/core/services/schedulingservice"
	"github.com/emoss08/trenova/internal/core/services/selfserviceservice"
	"github.com/emoss08/trenova/internal/core/services/servicetypeservice"
	"github.com/emoss08/trenova/internal/core/services/settlementcontrolservice"
	"github.com/emoss08/trenova/internal/core/services/shipmenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/sidebarpreferenceservice"
	"github.com/emoss08/trenova/internal/core/services/storedmileageservice"
	"github.com/emoss08/trenova/internal/core/services/tablechangealertservice"
	"github.com/emoss08/trenova/internal/core/services/tableconfigurationservice"
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/emoss08/trenova/internal/core/services/timesheetservice"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"github.com/emoss08/trenova/internal/core/services/userservice"
	"github.com/emoss08/trenova/internal/core/services/usstateservice"
	"github.com/emoss08/trenova/internal/core/services/workerchecklistservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/core/services/workerdqfservice"
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/core/services/workeremploymentservice"
	"github.com/emoss08/trenova/internal/core/services/workerinjuryservice"
	"github.com/emoss08/trenova/internal/core/services/workerleaveservice"
	"github.com/emoss08/trenova/internal/core/services/workeroverviewservice"
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                       *zap.Logger
	AnalyticsService             services.AnalyticsService
	OrganizationService          services.OrganizationService
	ShipmentService              services.ShipmentService
	ShipmentCommentService       services.ShipmentCommentService
	ShipmentEventService         services.ShipmentEventService
	ShipmentImportAssistant      services.ShipmentImportAssistantService `optional:"true"`
	EquipmentManufacturerService *equipmentmanufacturerservice.Service
	EDIService                   *ediservice.Service
	EDIInboundService            *ediinboundservice.Service
	EquipmentTypeService         *equipmenttypeservice.Service
	AccessorialChargeService     *accessorialchargeservice.Service
	AccountTypeService           *accounttypeservice.Service
	AccountsReceivableService    *accountsreceivableservice.Service
	CustomerPaymentService       services.CustomerPaymentService
	CustomerPaymentRepo          repositories.CustomerPaymentRepository
	InvoiceDisputeRepo           repositories.InvoiceDisputeRepository
	CarrierRepo                  repositories.CarrierRepository
	CustomerRepo                 repositories.CustomerRepository
	GLAccountRepo                repositories.GLAccountRepository
	CommodityService             *commodityservice.Service
	CarrierAssignmentService     *carrierassignmentservice.Service
	RoutingGuideService          *routingguideservice.Service
	TenderService                *tenderservice.Service
	CarrierService               *carrierservice.Service
	CarrierIntelService          *carrierintelservice.Service
	CustomerService              *customerservice.Service
	CustomFieldService           *customfieldservice.Service
	FleetCodeService             *fleetcodeservice.Service
	HazardousMaterialService     *hazardousmaterialservice.Service
	HazmatSegregationRuleService *hazmatsegregationruleservice.Service
	HoldReasonService            *holdreasonservice.Service
	JurisdictionRuleService      services.JurisdictionRuleService
	JurisdictionRuleRepo         repositories.JurisdictionRuleRepository
	RecurringShipmentService     *recurringshipmentservice.Service
	LocationService              *locationservice.Service
	LocationCategoryService      *locationcategoryservice.Service
	DocumentTypeService          *documenttypeservice.Service
	ServiceTypeService           *servicetypeservice.Service
	OrderService                 *orderservice.Service
	ShipmentTypeService          *shipmenttypeservice.Service
	TractorService               *tractorservice.Service
	TrailerService               *trailerservice.Service
	USStateService               *usstateservice.Service
	WorkerService                *workerservice.Service
	WorkerPTOService             services.WorkerPTOService
	PTOPolicyService             *ptopolicyservice.Service
	PTOLedgerService             *ptoledgerservice.Service
	WorkerCredentialService      *workercredentialservice.Service
	WorkerTrainingService        *workertrainingservice.Service
	WorkerSafetyService          *workersafetyservice.Service
	WorkerDrugAlcoholService     *workerdrugalcoholservice.Service
	WorkerDQFService             *workerdqfservice.Service
	WorkerInjuryService          *workerinjuryservice.Service
	WorkerLeaveService           *workerleaveservice.Service
	OrgStructureService          *orgstructureservice.Service
	BenefitsService              *benefitsservice.Service
	SchedulingService            *schedulingservice.Service
	SelfServiceService           *selfserviceservice.Service
	TimesheetService             *timesheetservice.Service
	WorkerOverviewService        *workeroverviewservice.Service
	PerformanceReviewService     *performancereviewservice.Service
	WorkerEmploymentService      *workeremploymentservice.Service
	WorkerChecklistService       *workerchecklistservice.Service
	OrgHolidayService            *orgholidayservice.Service
	WorkflowStarter              services.WorkflowStarter
	FiscalYearService            *fiscalyearservice.Service
	FormulaTemplateService       *formulatemplateservice.Service
	FuelSurchargeService         *fuelsurchargeservice.Service
	CostingService               *costingservice.Service
	DetentionService             *detentionservice.Service
	AccountingConnectionService  services.AccountingConnectionService
	AccountingMappingService     services.AccountingMappingService
	AccountingSyncService        services.AccountingSyncService
	AccountingInboundService     services.AccountingInboundService
	AccountingDriftService       services.AccountingDriftService
	DetentionPolicyService       *detentionpolicyservice.Service
	RateAgreementService         *rateagreementservice.Service
	RateZoneService              *ratezoneservice.Service
	RateMatrixService            *ratematrixservice.Service
	RateQuoteService             *ratequoteservice.Service
	FuelIndexRepo                repositories.FuelIndexRepository
	FuelIndexPriceRepo           repositories.FuelIndexPriceRepository
	FuelSurchargeProgramRepo     repositories.FuelSurchargeProgramRepository
	FiscalYearRepo               repositories.FiscalYearRepository
	FiscalPeriodRepo             repositories.FiscalPeriodRepository
	EmailService                 *emailservice.Service
	TelematicsService            *telematicsservice.Service
	DispatchConsoleService       services.DispatchConsoleService
	DispatchAutoAssignService    services.DispatchAutoAssignService
	AssignmentService            services.AssignmentService
	DocumentPacketRuleService    *documentpacketruleservice.Service
	DocumentTemplateService      *documenttemplateservice.Service
	DistanceOverrideService      *distanceoverrideservice.Service
	DistanceProfileService       *distanceprofileservice.Service
	StoredMileageService         *storedmileageservice.Service
	ManualJournalService         *manualjournalservice.Service
	JournalEntryService          *journalentryservice.Service
	JournalReview                services.JournalReviewService
	JournalReversalService       *journalreversalservice.Service
	AuditService                 services.AuditService
	ServiceFailureReasonCodeSvc  services.ServiceFailureReasonCodeService
	ServiceFailureSvc            services.ServiceFailureService
	BillingQueueService          services.BillingQueueService
	InvoiceService               services.InvoiceService
	InvoiceAdjustmentService     services.InvoiceAdjustmentService
	InvoiceDisputeService        services.InvoiceDisputeService
	LateChargeService            services.LateChargeService
	AgentRunService              services.AgentRunService
	AgentDefinitionService       services.AgentDefinitionService
	AIProviderService            services.AIProviderService
	AIUsageService               services.AIUsageService
	AIRetrievalStatusService     services.AIRetrievalStatusService
	AgentProposalService         services.AgentProposalService
	AgentPlanService             services.AgentPlanService
	AgentMemoryService           services.AgentMemoryService
	AIFeedbackService            services.AIFeedbackService
	AgentEvaluationService       services.AgentEvaluationService
	AgentEvalCaseService         services.AgentEvalCaseService
	ExtractionEvalService        services.ExtractionEvalService
	ExtractionShadowService      services.ExtractionShadowService
	ExtractionRolloutService     services.ExtractionRolloutService
	AITrainingHistoryService     services.AITrainingHistoryService
	AgentQualityService          services.AgentQualityService
	AgentExceptionService        services.AgentExceptionService
	AgentDecisionService         services.AgentDecisionService
	AgentDecisionQueueService    services.AgentDecisionQueueService
	AgentAccessService           services.AgentAccessService
	AgentSafetyService           services.AgentSafetyService
	WatchtowerService            services.WatchtowerService
	InboundMessageService        *inboundmessageservice.Service
	CaptureService               *captureservice.Service
	CaptureReleaseService        *capturereleaseservice.Service
	AgentScorecardService        services.AgentScorecardService
	AgentRunEventRepo            repositories.AgentRunEventRepository
	AIAuditService               services.AIAuditService
	BriefingService              services.BriefingService
	AgentTools                   services.AgentToolRegistry
	AgentControlService          services.AgentControlService
	ProposalPreviewService       services.ProposalPreviewService `optional:"true"`
	IAMService                   services.IAMService
	RoleService                  *roleservice.Service
	UserService                  *userservice.Service
	APIKeyService                *apikeyservice.Service
	TableChangeAlertService      *tablechangealertservice.Service
	TableConfigurationService    *tableconfigurationservice.Service
	SidebarPreferenceService     *sidebarpreferenceservice.Service
	HomeLayoutService            *homelayoutservice.Service
	BillingTransferService       *billingtransferservice.Service
	ReportingService             *reportingservice.Service
	NotificationService          *notificationservice.Service
	PermissionEngine             services.PermissionEngine
	DriverPayService             *driverpayservice.Service
	DriverSettlementService      *driversettlementservice.Service
	DriverPortalService          *driverportalservice.Service
	SettlementControlService     *settlementcontrolservice.Service
	DashControlService           *dashcontrolservice.Service
	PayProfileRepo               repositories.PayProfileRepository
	RecurringDeductionRepo       repositories.RecurringDeductionRepository
	RecurringEarningRepo         repositories.RecurringEarningRepository
	PayAdvanceRepo               repositories.PayAdvanceRepository
	EscrowAccountRepo            repositories.EscrowAccountRepository
	DriverSettlementRepo         repositories.DriverSettlementRepository
	SettlementBatchRepo          repositories.SettlementBatchRepository
	PayEventRepo                 repositories.PayEventRepository
	CarrierSettlementService     *carriersettlementservice.Service
	CarrierSettlementRepo        repositories.CarrierSettlementRepository
	CarrierSettlementBatchRepo   repositories.CarrierSettlementBatchRepository
	CarrierCostEventRepo         repositories.CarrierCostEventRepository
	FuelPurchaseService          *fuelpurchaseservice.Service
	IFTAService                  *iftaservice.Service
	DistanceCalculationService   services.DistanceCalculationService `optional:"true"`
	Config                       *config.Config                      `optional:"true"`
}

type Resolver struct {
	L                            *zap.Logger
	AnalyticsService             services.AnalyticsService
	OrganizationService          services.OrganizationService
	ShipmentService              services.ShipmentService
	ShipmentCommentService       services.ShipmentCommentService
	ShipmentEventService         services.ShipmentEventService
	ShipmentImportAssistant      services.ShipmentImportAssistantService
	EdiService                   *ediservice.Service
	EdiInboundService            *ediinboundservice.Service
	EquipmentTypeService         *equipmenttypeservice.Service
	AccessorialChargeService     *accessorialchargeservice.Service
	AccountTypeService           *accounttypeservice.Service
	AccountsReceivableService    *accountsreceivableservice.Service
	CustomerPaymentService       services.CustomerPaymentService
	CustomerPaymentRepo          repositories.CustomerPaymentRepository
	InvoiceDisputeRepo           repositories.InvoiceDisputeRepository
	carrierRepo                  repositories.CarrierRepository
	customerRepo                 repositories.CustomerRepository
	GlAccountRepo                repositories.GLAccountRepository
	CommodityService             *commodityservice.Service
	CarrierAssignmentService     *carrierassignmentservice.Service
	RoutingGuideService          *routingguideservice.Service
	TenderService                *tenderservice.Service
	CarrierService               *carrierservice.Service
	CarrierIntelService          *carrierintelservice.Service
	CustomerService              *customerservice.Service
	CustomFieldService           *customfieldservice.Service
	FleetCodeService             *fleetcodeservice.Service
	HazardousMaterialService     *hazardousmaterialservice.Service
	HazmatSegregationRuleService *hazmatsegregationruleservice.Service
	HoldReasonService            *holdreasonservice.Service
	JurisdictionRuleService      services.JurisdictionRuleService
	JurisdictionRuleRepo         repositories.JurisdictionRuleRepository
	RecurringShipmentService     *recurringshipmentservice.Service
	LocationService              *locationservice.Service
	LocationCategoryService      *locationcategoryservice.Service
	DocumentTypeService          *documenttypeservice.Service
	ServiceTypeService           *servicetypeservice.Service
	OrderService                 *orderservice.Service
	ShipmentTypeService          *shipmenttypeservice.Service
	EquipmentManufacturerService *equipmentmanufacturerservice.Service
	TractorService               *tractorservice.Service
	TrailerService               *trailerservice.Service
	usStateService               *usstateservice.Service
	WorkerService                *workerservice.Service
	WorkerPTOService             services.WorkerPTOService
	PtoPolicyService             *ptopolicyservice.Service
	PtoLedgerService             *ptoledgerservice.Service
	WorkerCredentialService      *workercredentialservice.Service
	WorkerTrainingService        *workertrainingservice.Service
	WorkerSafetyService          *workersafetyservice.Service
	WorkerDrugAlcoholService     *workerdrugalcoholservice.Service
	WorkerDQFService             *workerdqfservice.Service
	WorkerInjuryService          *workerinjuryservice.Service
	WorkerLeaveService           *workerleaveservice.Service
	OrgStructureService          *orgstructureservice.Service
	BenefitsService              *benefitsservice.Service
	SchedulingService            *schedulingservice.Service
	SelfServiceService           *selfserviceservice.Service
	TimesheetService             *timesheetservice.Service
	WorkerOverviewService        *workeroverviewservice.Service
	PerformanceReviewService     *performancereviewservice.Service
	WorkerEmploymentService      *workeremploymentservice.Service
	WorkerChecklistService       *workerchecklistservice.Service
	OrgHolidayService            *orgholidayservice.Service
	WorkflowStarter              services.WorkflowStarter
	FiscalYearService            *fiscalyearservice.Service
	FormulaTemplateService       *formulatemplateservice.Service
	FuelSurchargeService         *fuelsurchargeservice.Service
	CostingService               *costingservice.Service
	DetentionService             *detentionservice.Service
	AccountingConnections        services.AccountingConnectionService
	AccountingMappingService     services.AccountingMappingService
	AccountingSync               services.AccountingSyncService
	AccountingInbound            services.AccountingInboundService
	AccountingDrift              services.AccountingDriftService
	DetentionPolicyService       *detentionpolicyservice.Service
	RateAgreementService         *rateagreementservice.Service
	RateZoneService              *ratezoneservice.Service
	RateMatrixService            *ratematrixservice.Service
	RateQuoteService             *ratequoteservice.Service
	FuelIndexRepo                repositories.FuelIndexRepository
	FuelIndexPriceRepo           repositories.FuelIndexPriceRepository
	FuelSurchargeProgramRepo     repositories.FuelSurchargeProgramRepository
	fiscalYearRepo               repositories.FiscalYearRepository
	fiscalPeriodRepo             repositories.FiscalPeriodRepository
	EmailService                 *emailservice.Service
	TelematicsService            *telematicsservice.Service
	DispatchConsoleService       services.DispatchConsoleService
	DispatchAutoAssignService    services.DispatchAutoAssignService
	AssignmentService            services.AssignmentService
	DocumentPacketRuleService    *documentpacketruleservice.Service
	DocumentTemplateService      *documenttemplateservice.Service
	DistanceOverrideService      *distanceoverrideservice.Service
	DistanceProfileService       *distanceprofileservice.Service
	StoredMileageService         *storedmileageservice.Service
	ManualJournalService         *manualjournalservice.Service
	JournalEntryService          *journalentryservice.Service
	JournalReview                services.JournalReviewService
	JournalReversalService       *journalreversalservice.Service
	AuditService                 services.AuditService
	ServiceFailureReasonCodeSvc  services.ServiceFailureReasonCodeService
	ServiceFailureSvc            services.ServiceFailureService
	BillingQueueService          services.BillingQueueService
	InvoiceService               services.InvoiceService
	InvoiceAdjustmentService     services.InvoiceAdjustmentService
	InvoiceDisputeService        services.InvoiceDisputeService
	LateChargeService            services.LateChargeService
	AgentRunService              services.AgentRunService
	AgentDefinitionService       services.AgentDefinitionService
	AiProviderService            services.AIProviderService
	AiUsageService               services.AIUsageService
	AiRetrievalStatusService     services.AIRetrievalStatusService
	AgentProposalService         services.AgentProposalService
	AgentPlanService             services.AgentPlanService
	AgentMemoryService           services.AgentMemoryService
	AiFeedbackService            services.AIFeedbackService
	AgentEvaluationService       services.AgentEvaluationService
	AgentEvalCaseService         services.AgentEvalCaseService
	ExtractionEvalService        services.ExtractionEvalService
	ExtractionShadowService      services.ExtractionShadowService
	ExtractionRolloutService     services.ExtractionRolloutService
	AiTrainingHistoryService     services.AITrainingHistoryService
	AgentQualityService          services.AgentQualityService
	AgentExceptionService        services.AgentExceptionService
	AgentDecisionService         services.AgentDecisionService
	AgentDecisionQueueService    services.AgentDecisionQueueService
	AgentAccessService           services.AgentAccessService
	AgentSafetyService           services.AgentSafetyService
	WatchtowerService            services.WatchtowerService
	InboundMessageService        *inboundmessageservice.Service
	CaptureService               *captureservice.Service
	CaptureReleaseService        *capturereleaseservice.Service
	AgentScorecardService        services.AgentScorecardService
	AgentRunEventRepo            repositories.AgentRunEventRepository
	AiAuditService               services.AIAuditService
	BriefingService              services.BriefingService
	AgentTools                   services.AgentToolRegistry
	AgentControlService          services.AgentControlService
	ProposalPreviewService       services.ProposalPreviewService
	IamService                   services.IAMService
	RoleService                  *roleservice.Service
	UserService                  *userservice.Service
	APIKeyService                *apikeyservice.Service
	TableChangeAlertService      *tablechangealertservice.Service
	TableConfigurationService    *tableconfigurationservice.Service
	SidebarPreferenceService     *sidebarpreferenceservice.Service
	HomeLayoutService            *homelayoutservice.Service
	NotificationService          *notificationservice.Service
	DriverPayService             *driverpayservice.Service
	DriverSettlementService      *driversettlementservice.Service
	DriverPortalService          *driverportalservice.Service
	SettlementControlService     *settlementcontrolservice.Service
	DashControlService           *dashcontrolservice.Service
	PayProfileRepo               repositories.PayProfileRepository
	RecurringDeductionRepo       repositories.RecurringDeductionRepository
	RecurringEarningRepo         repositories.RecurringEarningRepository
	PayAdvanceRepo               repositories.PayAdvanceRepository
	EscrowAccountRepo            repositories.EscrowAccountRepository
	DriverSettlementRepo         repositories.DriverSettlementRepository
	SettlementBatchRepo          repositories.SettlementBatchRepository
	PayEventRepo                 repositories.PayEventRepository
	CarrierSettlementService     *carriersettlementservice.Service
	CarrierSettlementRepo        repositories.CarrierSettlementRepository
	CarrierSettlementBatchRepo   repositories.CarrierSettlementBatchRepository
	CarrierCostEventRepo         repositories.CarrierCostEventRepository
	FuelPurchaseService          *fuelpurchaseservice.Service
	IftaService                  *iftaservice.Service
	DistanceCalculationService   services.DistanceCalculationService
	BillingTransferService       *billingtransferservice.Service
	ReportingService             *reportingservice.Service
	PermissionEngine             services.PermissionEngine
	TraceURL                     func(traceID string) string
}

func New(p Params) *Resolver {
	return &Resolver{
		L:                            p.Logger.Named("api.graphql.resolver"),
		AnalyticsService:             p.AnalyticsService,
		OrganizationService:          p.OrganizationService,
		ShipmentService:              p.ShipmentService,
		ShipmentCommentService:       p.ShipmentCommentService,
		ShipmentEventService:         p.ShipmentEventService,
		ShipmentImportAssistant:      p.ShipmentImportAssistant,
		EdiService:                   p.EDIService,
		EdiInboundService:            p.EDIInboundService,
		EquipmentTypeService:         p.EquipmentTypeService,
		AccessorialChargeService:     p.AccessorialChargeService,
		AccountTypeService:           p.AccountTypeService,
		AccountsReceivableService:    p.AccountsReceivableService,
		CustomerPaymentService:       p.CustomerPaymentService,
		CustomerPaymentRepo:          p.CustomerPaymentRepo,
		InvoiceDisputeRepo:           p.InvoiceDisputeRepo,
		carrierRepo:                  p.CarrierRepo,
		customerRepo:                 p.CustomerRepo,
		GlAccountRepo:                p.GLAccountRepo,
		CommodityService:             p.CommodityService,
		CarrierAssignmentService:     p.CarrierAssignmentService,
		RoutingGuideService:          p.RoutingGuideService,
		TenderService:                p.TenderService,
		CarrierService:               p.CarrierService,
		CarrierIntelService:          p.CarrierIntelService,
		CustomerService:              p.CustomerService,
		CustomFieldService:           p.CustomFieldService,
		FleetCodeService:             p.FleetCodeService,
		HazardousMaterialService:     p.HazardousMaterialService,
		HazmatSegregationRuleService: p.HazmatSegregationRuleService,
		HoldReasonService:            p.HoldReasonService,
		JurisdictionRuleService:      p.JurisdictionRuleService,
		JurisdictionRuleRepo:         p.JurisdictionRuleRepo,
		RecurringShipmentService:     p.RecurringShipmentService,
		LocationService:              p.LocationService,
		LocationCategoryService:      p.LocationCategoryService,
		DocumentTypeService:          p.DocumentTypeService,
		ServiceTypeService:           p.ServiceTypeService,
		OrderService:                 p.OrderService,
		ShipmentTypeService:          p.ShipmentTypeService,
		EquipmentManufacturerService: p.EquipmentManufacturerService,
		TractorService:               p.TractorService,
		TrailerService:               p.TrailerService,
		usStateService:               p.USStateService,
		WorkerService:                p.WorkerService,
		WorkerPTOService:             p.WorkerPTOService,
		PtoPolicyService:             p.PTOPolicyService,
		PtoLedgerService:             p.PTOLedgerService,
		WorkerCredentialService:      p.WorkerCredentialService,
		WorkerTrainingService:        p.WorkerTrainingService,
		WorkerSafetyService:          p.WorkerSafetyService,
		WorkerDrugAlcoholService:     p.WorkerDrugAlcoholService,
		WorkerDQFService:             p.WorkerDQFService,
		WorkerInjuryService:          p.WorkerInjuryService,
		WorkerLeaveService:           p.WorkerLeaveService,
		OrgStructureService:          p.OrgStructureService,
		BenefitsService:              p.BenefitsService,
		SchedulingService:            p.SchedulingService,
		SelfServiceService:           p.SelfServiceService,
		TimesheetService:             p.TimesheetService,
		WorkerOverviewService:        p.WorkerOverviewService,
		PerformanceReviewService:     p.PerformanceReviewService,
		WorkerEmploymentService:      p.WorkerEmploymentService,
		WorkerChecklistService:       p.WorkerChecklistService,
		OrgHolidayService:            p.OrgHolidayService,
		WorkflowStarter:              p.WorkflowStarter,
		FiscalYearService:            p.FiscalYearService,
		FormulaTemplateService:       p.FormulaTemplateService,
		FuelSurchargeService:         p.FuelSurchargeService,
		CostingService:               p.CostingService,
		DetentionService:             p.DetentionService,
		AccountingConnections:        p.AccountingConnectionService,
		AccountingMappingService:     p.AccountingMappingService,
		AccountingSync:               p.AccountingSyncService,
		AccountingInbound:            p.AccountingInboundService,
		AccountingDrift:              p.AccountingDriftService,
		DetentionPolicyService:       p.DetentionPolicyService,
		RateAgreementService:         p.RateAgreementService,
		RateZoneService:              p.RateZoneService,
		RateMatrixService:            p.RateMatrixService,
		RateQuoteService:             p.RateQuoteService,
		FuelIndexRepo:                p.FuelIndexRepo,
		FuelIndexPriceRepo:           p.FuelIndexPriceRepo,
		FuelSurchargeProgramRepo:     p.FuelSurchargeProgramRepo,
		fiscalYearRepo:               p.FiscalYearRepo,
		fiscalPeriodRepo:             p.FiscalPeriodRepo,
		EmailService:                 p.EmailService,
		TelematicsService:            p.TelematicsService,
		DispatchConsoleService:       p.DispatchConsoleService,
		DispatchAutoAssignService:    p.DispatchAutoAssignService,
		AssignmentService:            p.AssignmentService,
		DocumentPacketRuleService:    p.DocumentPacketRuleService,
		DocumentTemplateService:      p.DocumentTemplateService,
		DistanceOverrideService:      p.DistanceOverrideService,
		DistanceProfileService:       p.DistanceProfileService,
		StoredMileageService:         p.StoredMileageService,
		ManualJournalService:         p.ManualJournalService,
		JournalEntryService:          p.JournalEntryService,
		JournalReview:                p.JournalReview,
		JournalReversalService:       p.JournalReversalService,
		AuditService:                 p.AuditService,
		ServiceFailureReasonCodeSvc:  p.ServiceFailureReasonCodeSvc,
		ServiceFailureSvc:            p.ServiceFailureSvc,
		BillingQueueService:          p.BillingQueueService,
		InvoiceService:               p.InvoiceService,
		InvoiceAdjustmentService:     p.InvoiceAdjustmentService,
		InvoiceDisputeService:        p.InvoiceDisputeService,
		LateChargeService:            p.LateChargeService,
		AgentRunService:              p.AgentRunService,
		AgentDefinitionService:       p.AgentDefinitionService,
		AiProviderService:            p.AIProviderService,
		AiUsageService:               p.AIUsageService,
		AiRetrievalStatusService:     p.AIRetrievalStatusService,
		AgentProposalService:         p.AgentProposalService,
		AgentPlanService:             p.AgentPlanService,
		AgentMemoryService:           p.AgentMemoryService,
		AiFeedbackService:            p.AIFeedbackService,
		AgentEvaluationService:       p.AgentEvaluationService,
		AgentEvalCaseService:         p.AgentEvalCaseService,
		ExtractionEvalService:        p.ExtractionEvalService,
		ExtractionShadowService:      p.ExtractionShadowService,
		ExtractionRolloutService:     p.ExtractionRolloutService,
		AiTrainingHistoryService:     p.AITrainingHistoryService,
		AgentQualityService:          p.AgentQualityService,
		AgentExceptionService:        p.AgentExceptionService,
		AgentDecisionService:         p.AgentDecisionService,
		AgentDecisionQueueService:    p.AgentDecisionQueueService,
		AgentAccessService:           p.AgentAccessService,
		AgentSafetyService:           p.AgentSafetyService,
		WatchtowerService:            p.WatchtowerService,
		InboundMessageService:        p.InboundMessageService,
		CaptureService:               p.CaptureService,
		CaptureReleaseService:        p.CaptureReleaseService,
		AgentScorecardService:        p.AgentScorecardService,
		AgentRunEventRepo:            p.AgentRunEventRepo,
		AiAuditService:               p.AIAuditService,
		BriefingService:              p.BriefingService,
		AgentTools:                   p.AgentTools,
		AgentControlService:          p.AgentControlService,
		ProposalPreviewService:       p.ProposalPreviewService,
		IamService:                   p.IAMService,
		RoleService:                  p.RoleService,
		UserService:                  p.UserService,
		APIKeyService:                p.APIKeyService,
		TableChangeAlertService:      p.TableChangeAlertService,
		TableConfigurationService:    p.TableConfigurationService,
		SidebarPreferenceService:     p.SidebarPreferenceService,
		HomeLayoutService:            p.HomeLayoutService,
		NotificationService:          p.NotificationService,
		BillingTransferService:       p.BillingTransferService,
		ReportingService:             p.ReportingService,
		PermissionEngine:             p.PermissionEngine,
		DriverPayService:             p.DriverPayService,
		DriverSettlementService:      p.DriverSettlementService,
		DriverPortalService:          p.DriverPortalService,
		SettlementControlService:     p.SettlementControlService,
		DashControlService:           p.DashControlService,
		PayProfileRepo:               p.PayProfileRepo,
		RecurringDeductionRepo:       p.RecurringDeductionRepo,
		RecurringEarningRepo:         p.RecurringEarningRepo,
		PayAdvanceRepo:               p.PayAdvanceRepo,
		EscrowAccountRepo:            p.EscrowAccountRepo,
		DriverSettlementRepo:         p.DriverSettlementRepo,
		SettlementBatchRepo:          p.SettlementBatchRepo,
		PayEventRepo:                 p.PayEventRepo,
		CarrierSettlementService:     p.CarrierSettlementService,
		CarrierSettlementRepo:        p.CarrierSettlementRepo,
		CarrierSettlementBatchRepo:   p.CarrierSettlementBatchRepo,
		CarrierCostEventRepo:         p.CarrierCostEventRepo,
		FuelPurchaseService:          p.FuelPurchaseService,
		IftaService:                  p.IFTAService,
		DistanceCalculationService:   p.DistanceCalculationService,
		TraceURL:                     TraceURLBuilder(p.Config),
	}
}

func (r *Resolver) RequirePermission(
	ctx context.Context,
	resource permission.Resource,
	operation permission.Operation,
) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	allowed, err := r.checkPermission(ctx, authCtx, resource, operation)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errortypes.NewAuthorizationError(
			"You don't have permission to perform this action: {0} {1}", resource, operation,
		)
	}

	return authCtx, nil
}

func (r *Resolver) HasPermission(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	resource permission.Resource,
	operation permission.Operation,
) bool {
	allowed, err := r.checkPermission(ctx, authCtx, resource, operation)
	if err != nil {
		r.L.Warn("permission check failed",
			zap.String("resource", resource.String()),
			zap.Error(err))
		return false
	}

	return allowed
}

func (r *Resolver) checkPermission(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	resource permission.Resource,
	operation permission.Operation,
) (bool, error) {
	memo, memoised := gqlctx.PermissionMemoFrom(ctx)
	key := resource.String() + "|" + string(operation)
	if memoised {
		if allowed, found := memo.Lookup(key); found {
			return allowed, nil
		}
	}

	result, err := r.PermissionEngine.Check(
		ctx,
		middleware.BuildPermissionCheckRequest(authCtx, resource.String(), operation),
	)
	if err != nil {
		return false, err
	}
	if memoised {
		memo.Store(key, result.Allowed)
	}

	return result.Allowed, nil
}

func (r *Resolver) RequireAuth(ctx context.Context) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	return authCtx, nil
}

func TenantInfo(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
