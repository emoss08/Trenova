package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/services/accessorialchargeservice"
	"github.com/emoss08/trenova/internal/core/services/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/core/services/accounttypeservice"
	"github.com/emoss08/trenova/internal/core/services/apikeyservice"
	"github.com/emoss08/trenova/internal/core/services/benefitsservice"
	"github.com/emoss08/trenova/internal/core/services/billingtransferservice"
	"github.com/emoss08/trenova/internal/core/services/bulkeditservice"
	"github.com/emoss08/trenova/internal/core/services/capturereleaseservice"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/internal/core/services/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/core/services/carriercapacityservice"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/services/carrierservice"
	"github.com/emoss08/trenova/internal/core/services/carriersettlementservice"
	"github.com/emoss08/trenova/internal/core/services/commodityservice"
	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/internal/core/services/customerservice"
	"github.com/emoss08/trenova/internal/core/services/customerupdateservice"
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
	"github.com/emoss08/trenova/internal/core/services/tableexportservice"
	"github.com/emoss08/trenova/internal/core/services/tableinsightservice"
	"github.com/emoss08/trenova/internal/core/services/tablelayoutservice"
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
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalyearports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentrosterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentshadowports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicontrolsummaryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aitrainingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aituneupports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/shipmentbriefports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentsuggestionports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/ports/jurisdictionruleports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                       *zap.Logger
	AnalyticsService             kernelports.AnalyticsService
	OrganizationService          tenantports.OrganizationService
	ShipmentService              shipmentports.ShipmentService
	ShipmentCommentService       shipmentports.ShipmentCommentService
	ShipmentEventService         shipmenteventports.ShipmentEventService
	CarrierCapacityService       *carriercapacityservice.Service
	CustomerUpdateService        *customerupdateservice.Service
	BoardCapacity                shipmentports.ShipmentCapacityReader
	BoardCoverage                shipmentports.ShipmentCoverageSuggester
	BoardTenderer                shipmentports.ShipmentTenderer
	BoardSuggestions             shipmentsuggestionports.ShipmentSuggestionReader
	BoardSuggestionDecider       shipmentsuggestionports.ShipmentSuggestionDecider
	BoardBriefing                shipmentbriefports.ShipmentBriefingReader
	BoardWatchlist               shipmentports.ShipmentWatchlistReader
	BoardQuickFilterCounts       shipmentports.ShipmentQuickFilterCounter
	BoardStageSummaries          shipmentports.ShipmentStageSummaryReader
	BoardGroups                  shipmentports.ShipmentBoardGroupSummarizer
	BoardCapabilities            shipmentports.ShipmentBoardCapabilitiesReader
	ShipmentImportAssistant      shipmentports.ShipmentImportAssistantService `optional:"true"`
	EquipmentManufacturerService *equipmentmanufacturerservice.Service
	EDIService                   *ediservice.Service
	EDIInboundService            *ediinboundservice.Service
	EquipmentTypeService         *equipmenttypeservice.Service
	AccessorialChargeService     *accessorialchargeservice.Service
	AccountTypeService           *accounttypeservice.Service
	AccountsReceivableService    *accountsreceivableservice.Service
	CustomerPaymentService       customerpaymentports.CustomerPaymentService
	CustomerPaymentRepo          customerpaymentports.CustomerPaymentRepository
	InvoiceDisputeRepo           invoiceports.InvoiceDisputeRepository
	CarrierRepo                  carrierports.CarrierRepository
	CustomerRepo                 customerports.CustomerRepository
	GLAccountRepo                glaccountports.GLAccountRepository
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
	JurisdictionRuleService      jurisdictionruleports.JurisdictionRuleService
	JurisdictionRuleRepo         permitports.JurisdictionRuleRepository
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
	WorkerPTOService             workerports.WorkerPTOService
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
	WorkflowStarter              kernelports.WorkflowStarter
	FiscalYearService            *fiscalyearservice.Service
	FormulaTemplateService       *formulatemplateservice.Service
	FuelSurchargeService         *fuelsurchargeservice.Service
	CostingService               *costingservice.Service
	DetentionService             *detentionservice.Service
	AccountingConnectionService  accountingsyncports.AccountingConnectionService
	AccountingMappingService     accountingsyncports.AccountingMappingService
	AccountingSyncService        accountingsyncports.AccountingSyncService
	AccountingInboundService     accountingsyncports.AccountingInboundService
	AccountingDriftService       accountingsyncports.AccountingDriftService
	DetentionPolicyService       *detentionpolicyservice.Service
	RateAgreementService         *rateagreementservice.Service
	RateZoneService              *ratezoneservice.Service
	RateMatrixService            *ratematrixservice.Service
	RateQuoteService             *ratequoteservice.Service
	FuelIndexRepo                fuelsurchargeports.FuelIndexRepository
	FuelIndexPriceRepo           fuelsurchargeports.FuelIndexPriceRepository
	FuelSurchargeProgramRepo     fuelsurchargeports.FuelSurchargeProgramRepository
	FiscalYearRepo               fiscalyearports.FiscalYearRepository
	FiscalPeriodRepo             fiscalperiodports.FiscalPeriodRepository
	EmailService                 *emailservice.Service
	TelematicsService            *telematicsservice.Service
	DispatchConsoleService       agentports.DispatchConsoleService
	DispatchAutoAssignService    agentports.DispatchAutoAssignService
	AssignmentService            shipmentports.AssignmentService
	DocumentPacketRuleService    *documentpacketruleservice.Service
	DocumentTemplateService      *documenttemplateservice.Service
	DistanceOverrideService      *distanceoverrideservice.Service
	DistanceProfileService       *distanceprofileservice.Service
	StoredMileageService         *storedmileageservice.Service
	ManualJournalService         *manualjournalservice.Service
	JournalEntryService          *journalentryservice.Service
	JournalReview                journalentryports.JournalReviewService
	JournalReversalService       *journalreversalservice.Service
	AuditService                 auditports.AuditService
	ServiceFailureReasonCodeSvc  servicefailureports.ServiceFailureReasonCodeService
	ServiceFailureSvc            servicefailureports.ServiceFailureService
	BillingQueueService          billingqueueports.BillingQueueService
	BillingQueueReviewService    billingqueueports.BillingQueueReviewService
	InvoiceService               invoiceports.InvoiceService
	InvoiceAdjustmentService     invoiceadjustmentports.InvoiceAdjustmentService
	InvoiceDisputeService        invoiceports.InvoiceDisputeService
	LateChargeService            latechargeports.LateChargeService
	AgentRunService              agentports.AgentRunService
	AgentDefinitionService       agentdefinitionports.AgentDefinitionService
	AgentCapabilityService       agentports.AgentCapabilityService
	AgentShadowService           agentshadowports.AgentShadowService
	AgentTrustService            agentports.AgentTrustService
	AgentTestPromptService       agentdefinitionports.AgentTestPromptService
	AgentDraftingService         agentports.AgentDraftingService
	AIControlSummaryService      aicontrolsummaryports.AIControlSummaryService
	AITuneUpService              aituneupports.AITuneUpService
	AgentRosterService           agentrosterports.AgentRosterService
	AIProviderService            agentports.AIProviderService
	AIUsageService               aiusageports.AIUsageService
	AIRetrievalStatusService     agentports.AIRetrievalStatusService
	AgentProposalService         agentports.AgentProposalService
	AgentPlanService             agentports.AgentPlanService
	AgentMemoryService           agentports.AgentMemoryService
	AgentReflectionService       agentports.AgentReflectionService
	AIFeedbackService            aifeedbackports.AIFeedbackService
	AgentEvaluationService       agentports.AgentEvaluationService
	AgentEvalCaseService         agentports.AgentEvalCaseService
	ExtractionEvalService        extractionevalports.ExtractionEvalService
	ExtractionShadowService      extractionshadowports.ExtractionShadowService
	ExtractionRolloutService     extractionrolloutports.ExtractionRolloutService
	AITrainingHistoryService     aitrainingports.AITrainingHistoryService
	AgentQualityService          agentqualityports.AgentQualityService
	AgentActivityService         agentports.AgentActivityService
	AgentExceptionService        agentports.AgentExceptionService
	AgentDecisionService         agentports.AgentDecisionService
	AgentDecisionQueueService    agentports.AgentDecisionQueueService
	ApprovalCommitter            agentports.ApprovalCommitter
	AgentAccessService           agentdefinitionports.AgentAccessService
	AgentSafetyService           agentports.AgentSafetyService
	AgentToolRuleService         agentports.AgentToolRuleService
	WatchtowerService            watchtowerports.WatchtowerService
	InboundMessageService        *inboundmessageservice.Service
	CaptureService               *captureservice.Service
	CaptureReleaseService        *capturereleaseservice.Service
	AgentScorecardService        agentports.AgentScorecardService
	AgentRunEventRepo            agentports.AgentRunEventRepository
	AIAuditService               aiauditports.AIAuditService
	BriefingService              briefingports.BriefingService
	AgentTools                   agentports.AgentToolRegistry
	AgentControlService          agentports.AgentControlService
	ProposalPreviewService       agentports.ProposalPreviewService `optional:"true"`
	IAMService                   iamports.IAMService
	RoleService                  *roleservice.Service
	UserService                  *userservice.Service
	APIKeyService                *apikeyservice.Service
	TableChangeAlertService      *tablechangealertservice.Service
	TableConfigurationService    *tableconfigurationservice.Service
	SidebarPreferenceService     *sidebarpreferenceservice.Service
	TableLayoutService           *tablelayoutservice.Service
	TableInsightService          *tableinsightservice.Service
	BulkEditService              *bulkeditservice.Service
	TableExportService           *tableexportservice.Service
	HomeLayoutService            *homelayoutservice.Service
	BillingTransferService       *billingtransferservice.Service
	ReportingService             *reportingservice.Service
	NotificationService          *notificationservice.Service
	PermissionEngine             permissionports.PermissionEngine
	DriverPayService             *driverpayservice.Service
	DriverSettlementService      *driversettlementservice.Service
	DriverPortalService          *driverportalservice.Service
	SettlementControlService     *settlementcontrolservice.Service
	DashControlService           *dashcontrolservice.Service
	PayProfileRepo               driverpayports.PayProfileRepository
	RecurringDeductionRepo       driverpayports.RecurringDeductionRepository
	RecurringEarningRepo         driverpayports.RecurringEarningRepository
	PayAdvanceRepo               driverpayports.PayAdvanceRepository
	EscrowAccountRepo            driverpayports.EscrowAccountRepository
	DriverSettlementRepo         driversettlementports.DriverSettlementRepository
	SettlementBatchRepo          driversettlementports.SettlementBatchRepository
	PayEventRepo                 driversettlementports.PayEventRepository
	CarrierSettlementService     *carriersettlementservice.Service
	CarrierSettlementRepo        carriersettlementports.CarrierSettlementRepository
	CarrierSettlementBatchRepo   carriersettlementports.CarrierSettlementBatchRepository
	CarrierCostEventRepo         carriersettlementports.CarrierCostEventRepository
	FuelPurchaseService          *fuelpurchaseservice.Service
	IFTAService                  *iftaservice.Service
	DistanceCalculationService   shipmentports.DistanceCalculationService `optional:"true"`
	Config                       *config.Config                           `optional:"true"`
}

type Services struct {
	*base.Core
	AnalyticsService             kernelports.AnalyticsService
	OrganizationService          tenantports.OrganizationService
	ShipmentService              shipmentports.ShipmentService
	ShipmentCommentService       shipmentports.ShipmentCommentService
	ShipmentEventService         shipmenteventports.ShipmentEventService
	CarrierCapacityService       *carriercapacityservice.Service
	CustomerUpdateService        *customerupdateservice.Service
	BoardCapacity                shipmentports.ShipmentCapacityReader
	BoardCoverage                shipmentports.ShipmentCoverageSuggester
	BoardTenderer                shipmentports.ShipmentTenderer
	BoardSuggestions             shipmentsuggestionports.ShipmentSuggestionReader
	BoardSuggestionDecider       shipmentsuggestionports.ShipmentSuggestionDecider
	BoardBriefing                shipmentbriefports.ShipmentBriefingReader
	BoardWatchlist               shipmentports.ShipmentWatchlistReader
	BoardQuickFilterCounts       shipmentports.ShipmentQuickFilterCounter
	BoardStageSummaries          shipmentports.ShipmentStageSummaryReader
	BoardGroups                  shipmentports.ShipmentBoardGroupSummarizer
	BoardCapabilities            shipmentports.ShipmentBoardCapabilitiesReader
	ShipmentImportAssistant      shipmentports.ShipmentImportAssistantService
	EdiService                   *ediservice.Service
	EdiInboundService            *ediinboundservice.Service
	EquipmentTypeService         *equipmenttypeservice.Service
	AccessorialChargeService     *accessorialchargeservice.Service
	AccountTypeService           *accounttypeservice.Service
	AccountsReceivableService    *accountsreceivableservice.Service
	CustomerPaymentService       customerpaymentports.CustomerPaymentService
	CustomerPaymentRepo          customerpaymentports.CustomerPaymentRepository
	InvoiceDisputeRepo           invoiceports.InvoiceDisputeRepository
	CarrierRepo                  carrierports.CarrierRepository
	CustomerRepo                 customerports.CustomerRepository
	GlAccountRepo                glaccountports.GLAccountRepository
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
	JurisdictionRuleService      jurisdictionruleports.JurisdictionRuleService
	JurisdictionRuleRepo         permitports.JurisdictionRuleRepository
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
	UsStateService               *usstateservice.Service
	WorkerService                *workerservice.Service
	PtoPolicyService             *ptopolicyservice.Service
	PtoLedgerService             *ptoledgerservice.Service
	WorkerCredentialService      *workercredentialservice.Service
	WorkerTrainingService        *workertrainingservice.Service
	WorkerSafetyService          *workersafetyservice.Service
	WorkerDrugAlcoholService     *workerdrugalcoholservice.Service
	WorkerDQFService             *workerdqfservice.Service
	WorkerInjuryService          *workerinjuryservice.Service
	WorkerLeaveService           *workerleaveservice.Service
	BenefitsService              *benefitsservice.Service
	SchedulingService            *schedulingservice.Service
	SelfServiceService           *selfserviceservice.Service
	TimesheetService             *timesheetservice.Service
	WorkerOverviewService        *workeroverviewservice.Service
	PerformanceReviewService     *performancereviewservice.Service
	WorkerEmploymentService      *workeremploymentservice.Service
	WorkerChecklistService       *workerchecklistservice.Service
	OrgHolidayService            *orgholidayservice.Service
	WorkflowStarter              kernelports.WorkflowStarter
	FiscalYearService            *fiscalyearservice.Service
	FormulaTemplateService       *formulatemplateservice.Service
	FuelSurchargeService         *fuelsurchargeservice.Service
	CostingService               *costingservice.Service
	DetentionService             *detentionservice.Service
	AccountingConnections        accountingsyncports.AccountingConnectionService
	AccountingMappingService     accountingsyncports.AccountingMappingService
	AccountingSync               accountingsyncports.AccountingSyncService
	AccountingInbound            accountingsyncports.AccountingInboundService
	AccountingDrift              accountingsyncports.AccountingDriftService
	DetentionPolicyService       *detentionpolicyservice.Service
	RateAgreementService         *rateagreementservice.Service
	RateZoneService              *ratezoneservice.Service
	RateMatrixService            *ratematrixservice.Service
	RateQuoteService             *ratequoteservice.Service
	FuelIndexRepo                fuelsurchargeports.FuelIndexRepository
	FuelIndexPriceRepo           fuelsurchargeports.FuelIndexPriceRepository
	FuelSurchargeProgramRepo     fuelsurchargeports.FuelSurchargeProgramRepository
	FiscalYearRepo               fiscalyearports.FiscalYearRepository
	FiscalPeriodRepo             fiscalperiodports.FiscalPeriodRepository
	EmailService                 *emailservice.Service
	TelematicsService            *telematicsservice.Service
	DispatchConsoleService       agentports.DispatchConsoleService
	DispatchAutoAssignService    agentports.DispatchAutoAssignService
	AssignmentService            shipmentports.AssignmentService
	DocumentPacketRuleService    *documentpacketruleservice.Service
	DocumentTemplateService      *documenttemplateservice.Service
	DistanceOverrideService      *distanceoverrideservice.Service
	DistanceProfileService       *distanceprofileservice.Service
	StoredMileageService         *storedmileageservice.Service
	ManualJournalService         *manualjournalservice.Service
	JournalEntryService          *journalentryservice.Service
	JournalReview                journalentryports.JournalReviewService
	JournalReversalService       *journalreversalservice.Service
	AuditService                 auditports.AuditService
	ServiceFailureReasonCodeSvc  servicefailureports.ServiceFailureReasonCodeService
	ServiceFailureSvc            servicefailureports.ServiceFailureService
	BillingQueueService          billingqueueports.BillingQueueService
	BillingQueueReviewService    billingqueueports.BillingQueueReviewService
	InvoiceService               invoiceports.InvoiceService
	InvoiceAdjustmentService     invoiceadjustmentports.InvoiceAdjustmentService
	InvoiceDisputeService        invoiceports.InvoiceDisputeService
	LateChargeService            latechargeports.LateChargeService
	AgentRunService              agentports.AgentRunService
	AgentDefinitionService       agentdefinitionports.AgentDefinitionService
	AgentCapabilityService       agentports.AgentCapabilityService
	AgentShadowService           agentshadowports.AgentShadowService
	AgentTrustService            agentports.AgentTrustService
	AgentTestPromptService       agentdefinitionports.AgentTestPromptService
	AgentDraftingService         agentports.AgentDraftingService
	AiControlSummaryService      aicontrolsummaryports.AIControlSummaryService
	AiTuneUpService              aituneupports.AITuneUpService
	AgentRosterService           agentrosterports.AgentRosterService
	AiProviderService            agentports.AIProviderService
	AiUsageService               aiusageports.AIUsageService
	AiRetrievalStatusService     agentports.AIRetrievalStatusService
	AgentProposalService         agentports.AgentProposalService
	AgentPlanService             agentports.AgentPlanService
	AgentMemoryService           agentports.AgentMemoryService
	AgentReflectionService       agentports.AgentReflectionService
	AiFeedbackService            aifeedbackports.AIFeedbackService
	AgentEvaluationService       agentports.AgentEvaluationService
	AgentEvalCaseService         agentports.AgentEvalCaseService
	ExtractionEvalService        extractionevalports.ExtractionEvalService
	ExtractionShadowService      extractionshadowports.ExtractionShadowService
	ExtractionRolloutService     extractionrolloutports.ExtractionRolloutService
	AiTrainingHistoryService     aitrainingports.AITrainingHistoryService
	AgentQualityService          agentqualityports.AgentQualityService
	AgentActivityService         agentports.AgentActivityService
	AgentExceptionService        agentports.AgentExceptionService
	AgentDecisionService         agentports.AgentDecisionService
	AgentDecisionQueueService    agentports.AgentDecisionQueueService
	ApprovalCommitter            agentports.ApprovalCommitter
	AgentAccessService           agentdefinitionports.AgentAccessService
	AgentSafetyService           agentports.AgentSafetyService
	AgentToolRuleService         agentports.AgentToolRuleService
	WatchtowerService            watchtowerports.WatchtowerService
	InboundMessageService        *inboundmessageservice.Service
	CaptureService               *captureservice.Service
	CaptureReleaseService        *capturereleaseservice.Service
	AgentScorecardService        agentports.AgentScorecardService
	AgentRunEventRepo            agentports.AgentRunEventRepository
	AiAuditService               aiauditports.AIAuditService
	BriefingService              briefingports.BriefingService
	AgentTools                   agentports.AgentToolRegistry
	AgentControlService          agentports.AgentControlService
	ProposalPreviewService       agentports.ProposalPreviewService
	IamService                   iamports.IAMService
	RoleService                  *roleservice.Service
	UserService                  *userservice.Service
	APIKeyService                *apikeyservice.Service
	TableChangeAlertService      *tablechangealertservice.Service
	TableConfigurationService    *tableconfigurationservice.Service
	SidebarPreferenceService     *sidebarpreferenceservice.Service
	TableLayoutService           *tablelayoutservice.Service
	TableInsightService          *tableinsightservice.Service
	BulkEditService              *bulkeditservice.Service
	TableExportService           *tableexportservice.Service
	HomeLayoutService            *homelayoutservice.Service
	NotificationService          *notificationservice.Service
	DriverPayService             *driverpayservice.Service
	DriverSettlementService      *driversettlementservice.Service
	DriverPortalService          *driverportalservice.Service
	SettlementControlService     *settlementcontrolservice.Service
	DashControlService           *dashcontrolservice.Service
	PayProfileRepo               driverpayports.PayProfileRepository
	RecurringDeductionRepo       driverpayports.RecurringDeductionRepository
	RecurringEarningRepo         driverpayports.RecurringEarningRepository
	PayAdvanceRepo               driverpayports.PayAdvanceRepository
	EscrowAccountRepo            driverpayports.EscrowAccountRepository
	DriverSettlementRepo         driversettlementports.DriverSettlementRepository
	SettlementBatchRepo          driversettlementports.SettlementBatchRepository
	PayEventRepo                 driversettlementports.PayEventRepository
	CarrierSettlementService     *carriersettlementservice.Service
	CarrierSettlementRepo        carriersettlementports.CarrierSettlementRepository
	CarrierSettlementBatchRepo   carriersettlementports.CarrierSettlementBatchRepository
	CarrierCostEventRepo         carriersettlementports.CarrierCostEventRepository
	FuelPurchaseService          *fuelpurchaseservice.Service
	IftaService                  *iftaservice.Service
	DistanceCalculationService   shipmentports.DistanceCalculationService
	BillingTransferService       *billingtransferservice.Service
	ReportingService             *reportingservice.Service
}

//nolint:funlen // one assignment per service; splitting it would only scatter the wiring
func newServices(p *Params) *Services {
	return &Services{
		Core: &base.Core{
			L:                   p.Logger.Named("api.graphql.resolver"),
			WorkerPTOService:    p.WorkerPTOService,
			OrgStructureService: p.OrgStructureService,
			PermissionEngine:    p.PermissionEngine,
			TraceURL:            base.TraceURLBuilder(p.Config),
		},
		AnalyticsService:             p.AnalyticsService,
		OrganizationService:          p.OrganizationService,
		ShipmentService:              p.ShipmentService,
		ShipmentCommentService:       p.ShipmentCommentService,
		ShipmentEventService:         p.ShipmentEventService,
		CarrierCapacityService:       p.CarrierCapacityService,
		CustomerUpdateService:        p.CustomerUpdateService,
		BoardCapacity:                p.BoardCapacity,
		BoardCoverage:                p.BoardCoverage,
		BoardTenderer:                p.BoardTenderer,
		BoardSuggestions:             p.BoardSuggestions,
		BoardSuggestionDecider:       p.BoardSuggestionDecider,
		BoardBriefing:                p.BoardBriefing,
		BoardWatchlist:               p.BoardWatchlist,
		BoardQuickFilterCounts:       p.BoardQuickFilterCounts,
		BoardStageSummaries:          p.BoardStageSummaries,
		BoardGroups:                  p.BoardGroups,
		BoardCapabilities:            p.BoardCapabilities,
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
		CarrierRepo:                  p.CarrierRepo,
		CustomerRepo:                 p.CustomerRepo,
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
		UsStateService:               p.USStateService,
		WorkerService:                p.WorkerService,
		PtoPolicyService:             p.PTOPolicyService,
		PtoLedgerService:             p.PTOLedgerService,
		WorkerCredentialService:      p.WorkerCredentialService,
		WorkerTrainingService:        p.WorkerTrainingService,
		WorkerSafetyService:          p.WorkerSafetyService,
		WorkerDrugAlcoholService:     p.WorkerDrugAlcoholService,
		WorkerDQFService:             p.WorkerDQFService,
		WorkerInjuryService:          p.WorkerInjuryService,
		WorkerLeaveService:           p.WorkerLeaveService,
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
		FiscalYearRepo:               p.FiscalYearRepo,
		FiscalPeriodRepo:             p.FiscalPeriodRepo,
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
		BillingQueueReviewService:    p.BillingQueueReviewService,
		InvoiceService:               p.InvoiceService,
		InvoiceAdjustmentService:     p.InvoiceAdjustmentService,
		InvoiceDisputeService:        p.InvoiceDisputeService,
		LateChargeService:            p.LateChargeService,
		AgentRunService:              p.AgentRunService,
		AgentDefinitionService:       p.AgentDefinitionService,
		AgentCapabilityService:       p.AgentCapabilityService,
		AgentShadowService:           p.AgentShadowService,
		AgentTrustService:            p.AgentTrustService,
		AgentTestPromptService:       p.AgentTestPromptService,
		AgentDraftingService:         p.AgentDraftingService,
		AiControlSummaryService:      p.AIControlSummaryService,
		AiTuneUpService:              p.AITuneUpService,
		AgentRosterService:           p.AgentRosterService,
		AiProviderService:            p.AIProviderService,
		AiUsageService:               p.AIUsageService,
		AiRetrievalStatusService:     p.AIRetrievalStatusService,
		AgentProposalService:         p.AgentProposalService,
		AgentPlanService:             p.AgentPlanService,
		AgentMemoryService:           p.AgentMemoryService,
		AgentReflectionService:       p.AgentReflectionService,
		AiFeedbackService:            p.AIFeedbackService,
		AgentEvaluationService:       p.AgentEvaluationService,
		AgentEvalCaseService:         p.AgentEvalCaseService,
		ExtractionEvalService:        p.ExtractionEvalService,
		ExtractionShadowService:      p.ExtractionShadowService,
		ExtractionRolloutService:     p.ExtractionRolloutService,
		AiTrainingHistoryService:     p.AITrainingHistoryService,
		AgentQualityService:          p.AgentQualityService,
		AgentActivityService:         p.AgentActivityService,
		AgentExceptionService:        p.AgentExceptionService,
		AgentDecisionService:         p.AgentDecisionService,
		AgentDecisionQueueService:    p.AgentDecisionQueueService,
		ApprovalCommitter:            p.ApprovalCommitter,
		AgentAccessService:           p.AgentAccessService,
		AgentSafetyService:           p.AgentSafetyService,
		AgentToolRuleService:         p.AgentToolRuleService,
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
		TableLayoutService:           p.TableLayoutService,
		TableInsightService:          p.TableInsightService,
		BulkEditService:              p.BulkEditService,
		TableExportService:           p.TableExportService,
		HomeLayoutService:            p.HomeLayoutService,
		NotificationService:          p.NotificationService,
		BillingTransferService:       p.BillingTransferService,
		ReportingService:             p.ReportingService,
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
	}
}
