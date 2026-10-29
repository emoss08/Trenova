package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accounttypeservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalyearservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalcardservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalentryservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalreversalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/manualjournalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalyearports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/app/boardcardservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/accountingbriefports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentrosterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentshadowports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentsuggestionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicontrolsummaryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aitrainingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aituneupports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/shipmentbriefports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/app/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturereleaseservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentpacketruleservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documenttemplateservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundreplyservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagebriefingservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageconflictservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagecorrectionservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagepeekservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageruleservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edimessageview"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioverview"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/fleetcodeservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/apikeyservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/dashcontrolservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/pageaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/settlementcontrolservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/tableinsightservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/usstateservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/customerupdateservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionpolicyservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/recurringshipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/routingguideservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/ratingreviewports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcapacityports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentimportassistantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentsuggestionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carriercapacityservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelenrollmentservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelmonitorservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelreviewservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelsourcingservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelverifyservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelvetservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/customerservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/factoringcompanyservice"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/places/app/distanceoverrideservice"
	"github.com/emoss08/trenova/internal/modules/places/app/distanceprofileservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationcategoryservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/places/app/storedmileageservice"
	"github.com/emoss08/trenova/internal/modules/places/ports/jurisdictionruleports"
	"github.com/emoss08/trenova/internal/modules/rating/app/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulalibraryservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratematrixservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratequoteservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratezoneservice"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/app/accessorialchargeservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/commodityservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/documenttypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/hazardousmaterialservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/hazmatsegregationruleservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/holdreasonservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/servicetypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/shipmenttypeservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fueldeskcardservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/benefitsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/orgholidayservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/ptopolicyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/selfserviceservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/telematicsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/timesheetservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerchecklistservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeremploymentservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerinjuryservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeroverviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/bulkeditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/customfieldservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/emailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/homelayoutservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	reportingservice "github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/sidebarpreferenceservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tablechangealertservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tableconfigurationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tableexportservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tablelayoutservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                        *zap.Logger
	AnalyticsService              kernelports.AnalyticsService
	OrganizationService           tenantports.OrganizationService
	ShipmentService               shipmentserviceports.ShipmentService
	ShipmentCommentService        shipmentcommentports.ShipmentCommentService
	ShipmentRatingReviewService   ratingreviewports.RatingReviewService
	ShipmentEventService          shipmenteventports.ShipmentEventService
	TrackingEventService          trackingeventports.TrackingEventService
	CheckCallService              checkcallports.CheckCallService
	CustomerTrackingService       customertrackingports.CustomerTrackingService
	CarrierCapacityService        *carriercapacityservice.Service
	CustomerUpdateService         *customerupdateservice.Service
	BoardCapacity                 shipmentcapacityports.ShipmentCapacityReader
	BoardCoverage                 shipmentcapacityports.ShipmentCoverageSuggester
	BoardTenderer                 shipmentcapacityports.ShipmentTenderer
	BoardSuggestions              shipmentsuggestionports.ShipmentSuggestionReader
	BoardSuggestionDecider        shipmentsuggestionports.ShipmentSuggestionDecider
	BoardBriefing                 shipmentbriefports.ShipmentBriefingReader
	BoardWatchlist                shipmentboardports.ShipmentWatchlistReader
	BoardQuickFilterCounts        shipmentboardports.ShipmentQuickFilterCounter
	BoardStageSummaries           shipmentboardports.ShipmentStageSummaryReader
	BoardGroups                   shipmentboardports.ShipmentBoardGroupSummarizer
	BoardCapabilities             shipmentboardports.ShipmentBoardCapabilitiesReader
	ShipmentImportAssistant       shipmentimportassistantports.ShipmentImportAssistantService `optional:"true"`
	EquipmentManufacturerService  *equipmentmanufacturerservice.Service
	EDISetupService               *edisetup.Service
	EDITransferService            *editransferservice.Service
	EDIOutboundService            *edioutbound.Service
	EDIMessageService             *edimessageview.Service
	EDIOverviewService            *edioverview.Service
	EDIInboundService             *ediinboundservice.Service
	EquipmentTypeService          *equipmenttypeservice.Service
	AccessorialChargeService      *accessorialchargeservice.Service
	AccountTypeService            *accounttypeservice.Service
	AccountsReceivableService     *accountsreceivableservice.Service
	AccountingBriefs              accountingbriefports.AccountingBriefingReader
	CustomerPaymentService        customerpaymentports.CustomerPaymentService
	CustomerPaymentRepo           customerpaymentports.CustomerPaymentRepository
	InvoiceDisputeRepo            invoiceports.InvoiceDisputeRepository
	CarrierRepo                   carrierports.CarrierRepository
	CustomerRepo                  customerports.CustomerRepository
	GLAccountRepo                 glaccountports.GLAccountRepository
	CommodityService              *commodityservice.Service
	CarrierAssignmentService      *carrierassignmentservice.Service
	RoutingGuideService           *routingguideservice.Service
	TenderService                 *tenderservice.Service
	CarrierService                *carrierservice.Service
	CarrierIntelService           *carrierintelservice.Service
	CarrierIntelEnrollmentService *carrierintelenrollmentservice.Service
	CarrierIntelMonitorService    *carrierintelmonitorservice.Service
	CarrierIntelReviewService     *carrierintelreviewservice.Service
	CarrierIntelVetService        *carrierintelvetservice.Service
	CarrierIntelSourcingService   *carrierintelsourcingservice.Service
	CarrierIntelVerifyService     *carrierintelverifyservice.Service
	CustomerService               *customerservice.Service
	CustomFieldService            *customfieldservice.Service
	FleetCodeService              *fleetcodeservice.Service
	HazardousMaterialService      *hazardousmaterialservice.Service
	HazmatSegregationRuleService  *hazmatsegregationruleservice.Service
	HoldReasonService             *holdreasonservice.Service
	FactoringCompanyService       *factoringcompanyservice.Service
	FactoringCompanyRepo          factoringcompanyports.FactoringCompanyRepository
	JurisdictionRuleService       jurisdictionruleports.JurisdictionRuleService
	JurisdictionRuleRepo          permitports.JurisdictionRuleRepository
	RecurringShipmentService      *recurringshipmentservice.Service
	LocationService               *locationservice.Service
	LocationCategoryService       *locationcategoryservice.Service
	DocumentTypeService           *documenttypeservice.Service
	ServiceTypeService            *servicetypeservice.Service
	OrderService                  *orderservice.Service
	ShipmentTypeService           *shipmenttypeservice.Service
	TractorService                *tractorservice.Service
	TrailerService                *trailerservice.Service
	USStateService                *usstateservice.Service
	WorkerService                 *workerservice.Service
	WorkerPTOService              workerptoports.WorkerPTOService
	PTOPolicyService              *ptopolicyservice.Service
	PTOLedgerService              *ptoledgerservice.Service
	WorkerCredentialService       *workercredentialservice.Service
	WorkerTrainingService         *workertrainingservice.Service
	WorkerSafetyService           *workersafetyservice.Service
	WorkerDrugAlcoholService      *workerdrugalcoholservice.Service
	WorkerDQFService              *workerdqfservice.Service
	WorkerInjuryService           *workerinjuryservice.Service
	WorkerLeaveService            *workerleaveservice.Service
	OrgStructureService           *orgstructureservice.Service
	BenefitsService               *benefitsservice.Service
	SchedulingService             *schedulingservice.Service
	SelfServiceService            *selfserviceservice.Service
	TimesheetService              *timesheetservice.Service
	WorkerOverviewService         *workeroverviewservice.Service
	PerformanceReviewService      *performancereviewservice.Service
	WorkerEmploymentService       *workeremploymentservice.Service
	WorkerChecklistService        *workerchecklistservice.Service
	OrgHolidayService             *orgholidayservice.Service
	WorkflowStarter               kernelports.WorkflowStarter
	FiscalYearService             *fiscalyearservice.Service
	FiscalPeriodService           *fiscalperiodservice.Service
	FormulaTemplateService        *formulatemplateservice.Service
	FormulaLibraryService         *formulalibraryservice.Service
	FuelSurchargeService          *fuelsurchargeservice.Service
	CostingService                *costingservice.Service
	DetentionService              *detentionservice.Service
	AccountingConnectionService   accountingconnectionports.AccountingConnectionService
	AccountingMappingService      accountingmappingports.AccountingMappingService
	AccountingSyncService         accountingsyncports.AccountingSyncService
	AccountingInboundService      accountinginboundports.AccountingInboundService
	AccountingDriftService        accountingdriftports.AccountingDriftService
	DetentionPolicyService        *detentionpolicyservice.Service
	RateAgreementService          *rateagreementservice.Service
	RateZoneService               *ratezoneservice.Service
	RateMatrixService             *ratematrixservice.Service
	RateQuoteService              *ratequoteservice.Service
	FuelIndexRepo                 fuelsurchargeports.FuelIndexRepository
	FuelIndexPriceRepo            fuelsurchargeports.FuelIndexPriceRepository
	FuelSurchargeProgramRepo      fuelsurchargeports.FuelSurchargeProgramRepository
	FiscalYearRepo                fiscalyearports.FiscalYearRepository
	FiscalPeriodRepo              fiscalperiodports.FiscalPeriodRepository
	EmailService                  *emailservice.Service
	TelematicsService             *telematicsservice.Service
	DispatchConsoleService        dispatchconsoleports.DispatchConsoleService
	DispatchAutoAssignService     dispatchconsoleports.DispatchAutoAssignService
	AssignmentService             assignmentports.AssignmentService
	DocumentPacketRuleService     *documentpacketruleservice.Service
	DocumentTemplateService       *documenttemplateservice.Service
	DistanceOverrideService       *distanceoverrideservice.Service
	DistanceProfileService        *distanceprofileservice.Service
	StoredMileageService          *storedmileageservice.Service
	ManualJournalService          *manualjournalservice.Service
	JournalEntryService           *journalentryservice.Service
	JournalCards                  *journalcardservice.Service
	FuelDeskCards                 *fueldeskcardservice.Service
	BoardCards                    *boardcardservice.Service
	JournalReview                 journalentryports.JournalReviewService
	JournalReversalService        *journalreversalservice.Service
	AuditService                  auditports.AuditService
	ServiceFailureReasonCodeSvc   servicefailureports.ServiceFailureReasonCodeService
	ServiceFailureSvc             servicefailureports.ServiceFailureService
	BillingQueueService           billingqueueports.BillingQueueService
	BillingQueueReviewService     billingqueueports.BillingQueueReviewService
	InvoiceService                invoiceports.InvoiceService
	InvoiceAdjustmentService      invoiceadjustmentports.InvoiceAdjustmentService
	FactoringAgreementService     factoringports.AgreementService
	FactoringNoaService           factoringports.NoaService
	FactoringReadinessService     factoringports.ReadinessService
	FactoringSubmissionService    factoringsubmissionports.SubmissionService
	FactoringLedgerService        factoringledgerports.LedgerService
	FactoringRemittanceService    factoringremittanceports.RemittanceService
	FactoringTransportService     factoringsubmissionports.TransportService
	InvoiceDisputeService         invoiceports.InvoiceDisputeService
	LateChargeService             latechargeports.LateChargeService
	AgentRunService               agentports.AgentRunService
	AgentDefinitionService        agentdefinitionports.AgentDefinitionService
	AgentCapabilityService        agentports.AgentCapabilityService
	AgentRecommender              agentports.AgentRecommender
	AgentShadowService            agentshadowports.AgentShadowService
	AgentTrustService             agentports.AgentTrustService
	AgentTestPromptService        agentdefinitionports.AgentTestPromptService
	AgentDraftingService          agentports.AgentDraftingService
	AIControlSummaryService       aicontrolsummaryports.AIControlSummaryService
	AITuneUpService               aituneupports.AITuneUpService
	AIAnalyticsService            aianalyticsports.AnalyticsService
	AgentRosterService            agentrosterports.AgentRosterService
	AIProviderService             aiproviderports.AIProviderService
	AIUsageService                aiusageports.AIUsageService
	AIRetrievalStatusService      retrievalports.AIRetrievalStatusService
	AgentProposalService          agentports.AgentProposalService
	AgentPlanService              agentports.AgentPlanService
	AgentMemoryService            agentmemoryports.AgentMemoryService
	AgentReflectionService        agentports.AgentReflectionService
	AIFeedbackService             aifeedbackports.AIFeedbackService
	AgentSuggestionService        agentsuggestionports.AgentSuggestionService
	AgentEvaluationService        agentports.AgentEvaluationService
	AgentEvalCaseService          agentports.AgentEvalCaseService
	ExtractionEvalService         extractionevalports.ExtractionEvalService
	ExtractionShadowService       extractionshadowports.ExtractionShadowService
	ExtractionRolloutService      extractionrolloutports.ExtractionRolloutService
	AITrainingHistoryService      aitrainingports.AITrainingHistoryService
	AgentQualityService           agentqualityports.AgentQualityService
	AgentActivityService          agentports.AgentActivityService
	AgentExceptionService         agentports.AgentExceptionService
	AgentDecisionService          agentports.AgentDecisionService
	AgentDecisionQueueService     agentports.AgentDecisionQueueService
	ApprovalCommitter             agentports.ApprovalCommitter
	AgentAccessService            agentdefinitionports.AgentAccessService
	AgentSafetyService            agenttoolports.AgentSafetyService
	AgentToolRuleService          agenttoolports.AgentToolRuleService
	WatchtowerService             watchtowerports.WatchtowerService
	WatchtowerExceptions          watchtowerports.ExceptionService
	InboundMessageService         *inboundmessageservice.Service
	InboundReplyService           *inboundreplyservice.Service
	TriageWorkService             *triageworkservice.Service
	TriagePeekService             *triagepeekservice.Service
	TriageRuleService             *triageruleservice.Service
	TriageBriefingService         *triagebriefingservice.Service
	TriageCorrectionService       *triagecorrectionservice.Service
	TriageConflictService         *triageconflictservice.Service
	CaptureService                *captureservice.Service
	CaptureReleaseService         *capturereleaseservice.Service
	AgentScorecardService         agentports.AgentScorecardService
	AgentRunEventRepo             agentruntimeports.AgentRunEventRepository
	AIAuditService                aiauditports.AIAuditService
	BriefingService               briefingports.BriefingService
	AgentProfileService           agentprofileports.Service
	AgentTools                    agenttoolports.AgentToolRegistry
	AgentControlService           agentports.AgentControlService
	ProposalPreviewService        agentruntimeports.ProposalPreviewService `optional:"true"`
	IAMService                    iamports.IAMService
	RoleService                   *roleservice.Service
	PageAccessService             *pageaccessservice.Service
	UserService                   *userservice.Service
	APIKeyService                 *apikeyservice.Service
	TableChangeAlertService       *tablechangealertservice.Service
	TableConfigurationService     *tableconfigurationservice.Service
	SidebarPreferenceService      *sidebarpreferenceservice.Service
	TableLayoutService            *tablelayoutservice.Service
	TableInsightService           *tableinsightservice.Service
	BulkEditService               *bulkeditservice.Service
	TableExportService            *tableexportservice.Service
	HomeLayoutService             *homelayoutservice.Service
	BillingTransferService        *billingtransferservice.Service
	ReportingService              *reportingservice.Service
	NotificationService           *notificationservice.Service
	PermissionEngine              permissionports.PermissionEngine
	DriverPayService              *driverpayservice.Service
	DriverSettlementService       *driversettlementservice.Service
	DriverPortalService           *driverportalservice.Service
	SettlementControlService      *settlementcontrolservice.Service
	AccountingControlRepo         tenantports.AccountingControlRepository
	DashControlService            *dashcontrolservice.Service
	PayProfileRepo                driverpayports.PayProfileRepository
	RecurringDeductionRepo        driverpayports.RecurringDeductionRepository
	RecurringEarningRepo          driverpayports.RecurringEarningRepository
	PayAdvanceRepo                driverpayports.PayAdvanceRepository
	EscrowAccountRepo             driverpayports.EscrowAccountRepository
	DriverSettlementRepo          driversettlementports.DriverSettlementRepository
	SettlementBatchRepo           driversettlementports.SettlementBatchRepository
	PayEventRepo                  driversettlementports.PayEventRepository
	CarrierSettlementService      *carriersettlementservice.Service
	CarrierSettlementRepo         carriersettlementports.CarrierSettlementRepository
	CarrierSettlementBatchRepo    carriersettlementports.CarrierSettlementBatchRepository
	CarrierCostEventRepo          carriersettlementports.CarrierCostEventRepository
	FuelPurchaseService           *fuelpurchaseservice.Service
	IFTAService                   *iftaservice.Service
	DistanceCalculationService    shipmentdistanceports.DistanceCalculationService `optional:"true"`
	MonitoringConfig              *monitoringconfig.Config                         `optional:"true"`
}

type Services struct {
	*resolvercore.Core
	AnalyticsService              kernelports.AnalyticsService
	OrganizationService           tenantports.OrganizationService
	ShipmentService               shipmentserviceports.ShipmentService
	ShipmentCommentService        shipmentcommentports.ShipmentCommentService
	ShipmentRatingReviewService   ratingreviewports.RatingReviewService
	ShipmentEventService          shipmenteventports.ShipmentEventService
	TrackingEventService          trackingeventports.TrackingEventService
	CheckCallService              checkcallports.CheckCallService
	CustomerTrackingService       customertrackingports.CustomerTrackingService
	CarrierCapacityService        *carriercapacityservice.Service
	CustomerUpdateService         *customerupdateservice.Service
	BoardCapacity                 shipmentcapacityports.ShipmentCapacityReader
	BoardCoverage                 shipmentcapacityports.ShipmentCoverageSuggester
	BoardTenderer                 shipmentcapacityports.ShipmentTenderer
	BoardSuggestions              shipmentsuggestionports.ShipmentSuggestionReader
	BoardSuggestionDecider        shipmentsuggestionports.ShipmentSuggestionDecider
	BoardBriefing                 shipmentbriefports.ShipmentBriefingReader
	BoardWatchlist                shipmentboardports.ShipmentWatchlistReader
	BoardQuickFilterCounts        shipmentboardports.ShipmentQuickFilterCounter
	BoardStageSummaries           shipmentboardports.ShipmentStageSummaryReader
	BoardGroups                   shipmentboardports.ShipmentBoardGroupSummarizer
	BoardCapabilities             shipmentboardports.ShipmentBoardCapabilitiesReader
	ShipmentImportAssistant       shipmentimportassistantports.ShipmentImportAssistantService
	EdiSetupService               *edisetup.Service
	EdiTransferService            *editransferservice.Service
	EdiOutboundService            *edioutbound.Service
	EdiMessageService             *edimessageview.Service
	EdiOverviewService            *edioverview.Service
	EdiInboundService             *ediinboundservice.Service
	EquipmentTypeService          *equipmenttypeservice.Service
	AccessorialChargeService      *accessorialchargeservice.Service
	AccountTypeService            *accounttypeservice.Service
	AccountsReceivableService     *accountsreceivableservice.Service
	AccountingBriefs              accountingbriefports.AccountingBriefingReader
	CustomerPaymentService        customerpaymentports.CustomerPaymentService
	CustomerPaymentRepo           customerpaymentports.CustomerPaymentRepository
	InvoiceDisputeRepo            invoiceports.InvoiceDisputeRepository
	CarrierRepo                   carrierports.CarrierRepository
	CustomerRepo                  customerports.CustomerRepository
	GlAccountRepo                 glaccountports.GLAccountRepository
	CommodityService              *commodityservice.Service
	CarrierAssignmentService      *carrierassignmentservice.Service
	RoutingGuideService           *routingguideservice.Service
	TenderService                 *tenderservice.Service
	CarrierService                *carrierservice.Service
	CarrierIntelService           *carrierintelservice.Service
	CarrierIntelEnrollmentService *carrierintelenrollmentservice.Service
	CarrierIntelMonitorService    *carrierintelmonitorservice.Service
	CarrierIntelReviewService     *carrierintelreviewservice.Service
	CarrierIntelVetService        *carrierintelvetservice.Service
	CarrierIntelSourcingService   *carrierintelsourcingservice.Service
	CarrierIntelVerifyService     *carrierintelverifyservice.Service
	CustomerService               *customerservice.Service
	CustomFieldService            *customfieldservice.Service
	FleetCodeService              *fleetcodeservice.Service
	HazardousMaterialService      *hazardousmaterialservice.Service
	HazmatSegregationRuleService  *hazmatsegregationruleservice.Service
	HoldReasonService             *holdreasonservice.Service
	FactoringCompanyService       *factoringcompanyservice.Service
	FactoringCompanyRepo          factoringcompanyports.FactoringCompanyRepository
	JurisdictionRuleService       jurisdictionruleports.JurisdictionRuleService
	JurisdictionRuleRepo          permitports.JurisdictionRuleRepository
	RecurringShipmentService      *recurringshipmentservice.Service
	LocationService               *locationservice.Service
	LocationCategoryService       *locationcategoryservice.Service
	DocumentTypeService           *documenttypeservice.Service
	ServiceTypeService            *servicetypeservice.Service
	OrderService                  *orderservice.Service
	ShipmentTypeService           *shipmenttypeservice.Service
	EquipmentManufacturerService  *equipmentmanufacturerservice.Service
	TractorService                *tractorservice.Service
	TrailerService                *trailerservice.Service
	UsStateService                *usstateservice.Service
	WorkerService                 *workerservice.Service
	PtoPolicyService              *ptopolicyservice.Service
	PtoLedgerService              *ptoledgerservice.Service
	WorkerCredentialService       *workercredentialservice.Service
	WorkerTrainingService         *workertrainingservice.Service
	WorkerSafetyService           *workersafetyservice.Service
	WorkerDrugAlcoholService      *workerdrugalcoholservice.Service
	WorkerDQFService              *workerdqfservice.Service
	WorkerInjuryService           *workerinjuryservice.Service
	WorkerLeaveService            *workerleaveservice.Service
	BenefitsService               *benefitsservice.Service
	SchedulingService             *schedulingservice.Service
	SelfServiceService            *selfserviceservice.Service
	TimesheetService              *timesheetservice.Service
	WorkerOverviewService         *workeroverviewservice.Service
	PerformanceReviewService      *performancereviewservice.Service
	WorkerEmploymentService       *workeremploymentservice.Service
	WorkerChecklistService        *workerchecklistservice.Service
	OrgHolidayService             *orgholidayservice.Service
	WorkflowStarter               kernelports.WorkflowStarter
	FiscalYearService             *fiscalyearservice.Service
	FiscalPeriodService           *fiscalperiodservice.Service
	FormulaTemplateService        *formulatemplateservice.Service
	FormulaLibraryService         *formulalibraryservice.Service
	FuelSurchargeService          *fuelsurchargeservice.Service
	CostingService                *costingservice.Service
	DetentionService              *detentionservice.Service
	AccountingConnections         accountingconnectionports.AccountingConnectionService
	AccountingMappingService      accountingmappingports.AccountingMappingService
	AccountingSync                accountingsyncports.AccountingSyncService
	AccountingInbound             accountinginboundports.AccountingInboundService
	AccountingDrift               accountingdriftports.AccountingDriftService
	DetentionPolicyService        *detentionpolicyservice.Service
	RateAgreementService          *rateagreementservice.Service
	RateZoneService               *ratezoneservice.Service
	RateMatrixService             *ratematrixservice.Service
	RateQuoteService              *ratequoteservice.Service
	FuelIndexRepo                 fuelsurchargeports.FuelIndexRepository
	FuelIndexPriceRepo            fuelsurchargeports.FuelIndexPriceRepository
	FuelSurchargeProgramRepo      fuelsurchargeports.FuelSurchargeProgramRepository
	FiscalYearRepo                fiscalyearports.FiscalYearRepository
	FiscalPeriodRepo              fiscalperiodports.FiscalPeriodRepository
	EmailService                  *emailservice.Service
	TelematicsService             *telematicsservice.Service
	DispatchConsoleService        dispatchconsoleports.DispatchConsoleService
	DispatchAutoAssignService     dispatchconsoleports.DispatchAutoAssignService
	AssignmentService             assignmentports.AssignmentService
	DocumentPacketRuleService     *documentpacketruleservice.Service
	DocumentTemplateService       *documenttemplateservice.Service
	DistanceOverrideService       *distanceoverrideservice.Service
	DistanceProfileService        *distanceprofileservice.Service
	StoredMileageService          *storedmileageservice.Service
	ManualJournalService          *manualjournalservice.Service
	JournalEntryService           *journalentryservice.Service
	JournalCards                  *journalcardservice.Service
	FuelDeskCards                 *fueldeskcardservice.Service
	BoardCards                    *boardcardservice.Service
	JournalReview                 journalentryports.JournalReviewService
	JournalReversalService        *journalreversalservice.Service
	AuditService                  auditports.AuditService
	ServiceFailureReasonCodeSvc   servicefailureports.ServiceFailureReasonCodeService
	ServiceFailureSvc             servicefailureports.ServiceFailureService
	BillingQueueService           billingqueueports.BillingQueueService
	BillingQueueReviewService     billingqueueports.BillingQueueReviewService
	InvoiceService                invoiceports.InvoiceService
	InvoiceAdjustmentService      invoiceadjustmentports.InvoiceAdjustmentService
	FactoringAgreementService     factoringports.AgreementService
	FactoringNoaService           factoringports.NoaService
	FactoringReadinessService     factoringports.ReadinessService
	FactoringSubmissionService    factoringsubmissionports.SubmissionService
	FactoringLedgerService        factoringledgerports.LedgerService
	FactoringRemittanceService    factoringremittanceports.RemittanceService
	FactoringTransportService     factoringsubmissionports.TransportService
	InvoiceDisputeService         invoiceports.InvoiceDisputeService
	LateChargeService             latechargeports.LateChargeService
	AgentRunService               agentports.AgentRunService
	AgentDefinitionService        agentdefinitionports.AgentDefinitionService
	AgentCapabilityService        agentports.AgentCapabilityService
	AgentRecommender              agentports.AgentRecommender
	AgentShadowService            agentshadowports.AgentShadowService
	AgentTrustService             agentports.AgentTrustService
	AgentTestPromptService        agentdefinitionports.AgentTestPromptService
	AgentDraftingService          agentports.AgentDraftingService
	AiControlSummaryService       aicontrolsummaryports.AIControlSummaryService
	AiTuneUpService               aituneupports.AITuneUpService
	AiAnalyticsService            aianalyticsports.AnalyticsService
	AgentRosterService            agentrosterports.AgentRosterService
	AiProviderService             aiproviderports.AIProviderService
	AiUsageService                aiusageports.AIUsageService
	AiRetrievalStatusService      retrievalports.AIRetrievalStatusService
	AgentProposalService          agentports.AgentProposalService
	AgentPlanService              agentports.AgentPlanService
	AgentMemoryService            agentmemoryports.AgentMemoryService
	AgentReflectionService        agentports.AgentReflectionService
	AiFeedbackService             aifeedbackports.AIFeedbackService
	AgentSuggestionService        agentsuggestionports.AgentSuggestionService
	AgentEvaluationService        agentports.AgentEvaluationService
	AgentEvalCaseService          agentports.AgentEvalCaseService
	ExtractionEvalService         extractionevalports.ExtractionEvalService
	ExtractionShadowService       extractionshadowports.ExtractionShadowService
	ExtractionRolloutService      extractionrolloutports.ExtractionRolloutService
	AiTrainingHistoryService      aitrainingports.AITrainingHistoryService
	AgentQualityService           agentqualityports.AgentQualityService
	AgentActivityService          agentports.AgentActivityService
	AgentExceptionService         agentports.AgentExceptionService
	AgentDecisionService          agentports.AgentDecisionService
	AgentDecisionQueueService     agentports.AgentDecisionQueueService
	ApprovalCommitter             agentports.ApprovalCommitter
	AgentAccessService            agentdefinitionports.AgentAccessService
	AgentSafetyService            agenttoolports.AgentSafetyService
	AgentToolRuleService          agenttoolports.AgentToolRuleService
	WatchtowerService             watchtowerports.WatchtowerService
	WatchtowerExceptions          watchtowerports.ExceptionService
	InboundMessageService         *inboundmessageservice.Service
	InboundReplyService           *inboundreplyservice.Service
	TriageWorkService             *triageworkservice.Service
	TriagePeekService             *triagepeekservice.Service
	TriageRuleService             *triageruleservice.Service
	TriageBriefingService         *triagebriefingservice.Service
	TriageCorrectionService       *triagecorrectionservice.Service
	TriageConflictService         *triageconflictservice.Service
	CaptureService                *captureservice.Service
	CaptureReleaseService         *capturereleaseservice.Service
	AgentScorecardService         agentports.AgentScorecardService
	AgentRunEventRepo             agentruntimeports.AgentRunEventRepository
	AiAuditService                aiauditports.AIAuditService
	BriefingService               briefingports.BriefingService
	AgentProfileService           agentprofileports.Service
	AgentTools                    agenttoolports.AgentToolRegistry
	AgentControlService           agentports.AgentControlService
	ProposalPreviewService        agentruntimeports.ProposalPreviewService
	IamService                    iamports.IAMService
	RoleService                   *roleservice.Service
	PageAccessService             *pageaccessservice.Service
	UserService                   *userservice.Service
	APIKeyService                 *apikeyservice.Service
	TableChangeAlertService       *tablechangealertservice.Service
	TableConfigurationService     *tableconfigurationservice.Service
	SidebarPreferenceService      *sidebarpreferenceservice.Service
	TableLayoutService            *tablelayoutservice.Service
	TableInsightService           *tableinsightservice.Service
	BulkEditService               *bulkeditservice.Service
	TableExportService            *tableexportservice.Service
	HomeLayoutService             *homelayoutservice.Service
	NotificationService           *notificationservice.Service
	DriverPayService              *driverpayservice.Service
	DriverSettlementService       *driversettlementservice.Service
	DriverPortalService           *driverportalservice.Service
	SettlementControlService      *settlementcontrolservice.Service
	AccountingControlRepo         tenantports.AccountingControlRepository
	DashControlService            *dashcontrolservice.Service
	PayProfileRepo                driverpayports.PayProfileRepository
	RecurringDeductionRepo        driverpayports.RecurringDeductionRepository
	RecurringEarningRepo          driverpayports.RecurringEarningRepository
	PayAdvanceRepo                driverpayports.PayAdvanceRepository
	EscrowAccountRepo             driverpayports.EscrowAccountRepository
	DriverSettlementRepo          driversettlementports.DriverSettlementRepository
	SettlementBatchRepo           driversettlementports.SettlementBatchRepository
	PayEventRepo                  driversettlementports.PayEventRepository
	CarrierSettlementService      *carriersettlementservice.Service
	CarrierSettlementRepo         carriersettlementports.CarrierSettlementRepository
	CarrierSettlementBatchRepo    carriersettlementports.CarrierSettlementBatchRepository
	CarrierCostEventRepo          carriersettlementports.CarrierCostEventRepository
	FuelPurchaseService           *fuelpurchaseservice.Service
	IftaService                   *iftaservice.Service
	DistanceCalculationService    shipmentdistanceports.DistanceCalculationService
	BillingTransferService        *billingtransferservice.Service
	ReportingService              *reportingservice.Service
}

//nolint:funlen // one assignment per service; splitting it would only scatter the wiring
func newServices(p *Params) *Services {
	return &Services{
		Core: &resolvercore.Core{
			L:                   p.Logger.Named("api.graphql.resolver"),
			WorkerPTOService:    p.WorkerPTOService,
			OrgStructureService: p.OrgStructureService,
			PermissionEngine:    p.PermissionEngine,
			TraceURL:            resolvercore.TraceURLBuilder(p.MonitoringConfig),
		},
		AnalyticsService:              p.AnalyticsService,
		OrganizationService:           p.OrganizationService,
		ShipmentService:               p.ShipmentService,
		ShipmentCommentService:        p.ShipmentCommentService,
		ShipmentRatingReviewService:   p.ShipmentRatingReviewService,
		ShipmentEventService:          p.ShipmentEventService,
		TrackingEventService:          p.TrackingEventService,
		CheckCallService:              p.CheckCallService,
		CustomerTrackingService:       p.CustomerTrackingService,
		CarrierCapacityService:        p.CarrierCapacityService,
		CustomerUpdateService:         p.CustomerUpdateService,
		BoardCapacity:                 p.BoardCapacity,
		BoardCoverage:                 p.BoardCoverage,
		BoardTenderer:                 p.BoardTenderer,
		BoardSuggestions:              p.BoardSuggestions,
		BoardSuggestionDecider:        p.BoardSuggestionDecider,
		BoardBriefing:                 p.BoardBriefing,
		BoardWatchlist:                p.BoardWatchlist,
		BoardQuickFilterCounts:        p.BoardQuickFilterCounts,
		BoardStageSummaries:           p.BoardStageSummaries,
		BoardGroups:                   p.BoardGroups,
		BoardCapabilities:             p.BoardCapabilities,
		ShipmentImportAssistant:       p.ShipmentImportAssistant,
		EdiSetupService:               p.EDISetupService,
		EdiTransferService:            p.EDITransferService,
		EdiOutboundService:            p.EDIOutboundService,
		EdiMessageService:             p.EDIMessageService,
		EdiOverviewService:            p.EDIOverviewService,
		EdiInboundService:             p.EDIInboundService,
		EquipmentTypeService:          p.EquipmentTypeService,
		AccessorialChargeService:      p.AccessorialChargeService,
		AccountTypeService:            p.AccountTypeService,
		AccountsReceivableService:     p.AccountsReceivableService,
		AccountingBriefs:              p.AccountingBriefs,
		CustomerPaymentService:        p.CustomerPaymentService,
		CustomerPaymentRepo:           p.CustomerPaymentRepo,
		InvoiceDisputeRepo:            p.InvoiceDisputeRepo,
		CarrierRepo:                   p.CarrierRepo,
		CustomerRepo:                  p.CustomerRepo,
		GlAccountRepo:                 p.GLAccountRepo,
		CommodityService:              p.CommodityService,
		CarrierAssignmentService:      p.CarrierAssignmentService,
		RoutingGuideService:           p.RoutingGuideService,
		TenderService:                 p.TenderService,
		CarrierService:                p.CarrierService,
		CarrierIntelService:           p.CarrierIntelService,
		CarrierIntelEnrollmentService: p.CarrierIntelEnrollmentService,
		CarrierIntelMonitorService:    p.CarrierIntelMonitorService,
		CarrierIntelReviewService:     p.CarrierIntelReviewService,
		CarrierIntelVetService:        p.CarrierIntelVetService,
		CarrierIntelSourcingService:   p.CarrierIntelSourcingService,
		CarrierIntelVerifyService:     p.CarrierIntelVerifyService,
		CustomerService:               p.CustomerService,
		CustomFieldService:            p.CustomFieldService,
		FleetCodeService:              p.FleetCodeService,
		HazardousMaterialService:      p.HazardousMaterialService,
		HazmatSegregationRuleService:  p.HazmatSegregationRuleService,
		HoldReasonService:             p.HoldReasonService,
		FactoringCompanyService:       p.FactoringCompanyService,
		FactoringCompanyRepo:          p.FactoringCompanyRepo,
		JurisdictionRuleService:       p.JurisdictionRuleService,
		JurisdictionRuleRepo:          p.JurisdictionRuleRepo,
		RecurringShipmentService:      p.RecurringShipmentService,
		LocationService:               p.LocationService,
		LocationCategoryService:       p.LocationCategoryService,
		DocumentTypeService:           p.DocumentTypeService,
		ServiceTypeService:            p.ServiceTypeService,
		OrderService:                  p.OrderService,
		ShipmentTypeService:           p.ShipmentTypeService,
		EquipmentManufacturerService:  p.EquipmentManufacturerService,
		TractorService:                p.TractorService,
		TrailerService:                p.TrailerService,
		UsStateService:                p.USStateService,
		WorkerService:                 p.WorkerService,
		PtoPolicyService:              p.PTOPolicyService,
		PtoLedgerService:              p.PTOLedgerService,
		WorkerCredentialService:       p.WorkerCredentialService,
		WorkerTrainingService:         p.WorkerTrainingService,
		WorkerSafetyService:           p.WorkerSafetyService,
		WorkerDrugAlcoholService:      p.WorkerDrugAlcoholService,
		WorkerDQFService:              p.WorkerDQFService,
		WorkerInjuryService:           p.WorkerInjuryService,
		WorkerLeaveService:            p.WorkerLeaveService,
		BenefitsService:               p.BenefitsService,
		SchedulingService:             p.SchedulingService,
		SelfServiceService:            p.SelfServiceService,
		TimesheetService:              p.TimesheetService,
		WorkerOverviewService:         p.WorkerOverviewService,
		PerformanceReviewService:      p.PerformanceReviewService,
		WorkerEmploymentService:       p.WorkerEmploymentService,
		WorkerChecklistService:        p.WorkerChecklistService,
		OrgHolidayService:             p.OrgHolidayService,
		WorkflowStarter:               p.WorkflowStarter,
		FiscalYearService:             p.FiscalYearService,
		FiscalPeriodService:           p.FiscalPeriodService,
		FormulaTemplateService:        p.FormulaTemplateService,
		FormulaLibraryService:         p.FormulaLibraryService,
		FuelSurchargeService:          p.FuelSurchargeService,
		CostingService:                p.CostingService,
		DetentionService:              p.DetentionService,
		AccountingConnections:         p.AccountingConnectionService,
		AccountingMappingService:      p.AccountingMappingService,
		AccountingSync:                p.AccountingSyncService,
		AccountingInbound:             p.AccountingInboundService,
		AccountingDrift:               p.AccountingDriftService,
		DetentionPolicyService:        p.DetentionPolicyService,
		RateAgreementService:          p.RateAgreementService,
		RateZoneService:               p.RateZoneService,
		RateMatrixService:             p.RateMatrixService,
		RateQuoteService:              p.RateQuoteService,
		FuelIndexRepo:                 p.FuelIndexRepo,
		FuelIndexPriceRepo:            p.FuelIndexPriceRepo,
		FuelSurchargeProgramRepo:      p.FuelSurchargeProgramRepo,
		FiscalYearRepo:                p.FiscalYearRepo,
		FiscalPeriodRepo:              p.FiscalPeriodRepo,
		EmailService:                  p.EmailService,
		TelematicsService:             p.TelematicsService,
		DispatchConsoleService:        p.DispatchConsoleService,
		DispatchAutoAssignService:     p.DispatchAutoAssignService,
		AssignmentService:             p.AssignmentService,
		DocumentPacketRuleService:     p.DocumentPacketRuleService,
		DocumentTemplateService:       p.DocumentTemplateService,
		DistanceOverrideService:       p.DistanceOverrideService,
		DistanceProfileService:        p.DistanceProfileService,
		StoredMileageService:          p.StoredMileageService,
		ManualJournalService:          p.ManualJournalService,
		JournalEntryService:           p.JournalEntryService,
		JournalCards:                  p.JournalCards,
		FuelDeskCards:                 p.FuelDeskCards,
		BoardCards:                    p.BoardCards,
		JournalReview:                 p.JournalReview,
		JournalReversalService:        p.JournalReversalService,
		AuditService:                  p.AuditService,
		ServiceFailureReasonCodeSvc:   p.ServiceFailureReasonCodeSvc,
		ServiceFailureSvc:             p.ServiceFailureSvc,
		BillingQueueService:           p.BillingQueueService,
		BillingQueueReviewService:     p.BillingQueueReviewService,
		InvoiceService:                p.InvoiceService,
		InvoiceAdjustmentService:      p.InvoiceAdjustmentService,
		InvoiceDisputeService:         p.InvoiceDisputeService,
		FactoringAgreementService:     p.FactoringAgreementService,
		FactoringNoaService:           p.FactoringNoaService,
		FactoringReadinessService:     p.FactoringReadinessService,
		FactoringSubmissionService:    p.FactoringSubmissionService,
		FactoringLedgerService:        p.FactoringLedgerService,
		FactoringRemittanceService:    p.FactoringRemittanceService,
		FactoringTransportService:     p.FactoringTransportService,
		LateChargeService:             p.LateChargeService,
		AgentRunService:               p.AgentRunService,
		AgentDefinitionService:        p.AgentDefinitionService,
		AgentCapabilityService:        p.AgentCapabilityService,
		AgentRecommender:              p.AgentRecommender,
		AgentShadowService:            p.AgentShadowService,
		AgentTrustService:             p.AgentTrustService,
		AgentTestPromptService:        p.AgentTestPromptService,
		AgentDraftingService:          p.AgentDraftingService,
		AiControlSummaryService:       p.AIControlSummaryService,
		AiTuneUpService:               p.AITuneUpService,
		AiAnalyticsService:            p.AIAnalyticsService,
		AgentRosterService:            p.AgentRosterService,
		AiProviderService:             p.AIProviderService,
		AiUsageService:                p.AIUsageService,
		AiRetrievalStatusService:      p.AIRetrievalStatusService,
		AgentProposalService:          p.AgentProposalService,
		AgentPlanService:              p.AgentPlanService,
		AgentMemoryService:            p.AgentMemoryService,
		AgentReflectionService:        p.AgentReflectionService,
		AiFeedbackService:             p.AIFeedbackService,
		AgentSuggestionService:        p.AgentSuggestionService,
		AgentEvaluationService:        p.AgentEvaluationService,
		AgentEvalCaseService:          p.AgentEvalCaseService,
		ExtractionEvalService:         p.ExtractionEvalService,
		ExtractionShadowService:       p.ExtractionShadowService,
		ExtractionRolloutService:      p.ExtractionRolloutService,
		AiTrainingHistoryService:      p.AITrainingHistoryService,
		AgentQualityService:           p.AgentQualityService,
		AgentActivityService:          p.AgentActivityService,
		AgentExceptionService:         p.AgentExceptionService,
		AgentDecisionService:          p.AgentDecisionService,
		AgentDecisionQueueService:     p.AgentDecisionQueueService,
		ApprovalCommitter:             p.ApprovalCommitter,
		AgentAccessService:            p.AgentAccessService,
		AgentSafetyService:            p.AgentSafetyService,
		AgentToolRuleService:          p.AgentToolRuleService,
		WatchtowerService:             p.WatchtowerService,
		WatchtowerExceptions:          p.WatchtowerExceptions,
		InboundMessageService:         p.InboundMessageService,
		InboundReplyService:           p.InboundReplyService,
		TriageWorkService:             p.TriageWorkService,
		TriagePeekService:             p.TriagePeekService,
		TriageRuleService:             p.TriageRuleService,
		TriageBriefingService:         p.TriageBriefingService,
		TriageCorrectionService:       p.TriageCorrectionService,
		TriageConflictService:         p.TriageConflictService,
		CaptureService:                p.CaptureService,
		CaptureReleaseService:         p.CaptureReleaseService,
		AgentScorecardService:         p.AgentScorecardService,
		AgentRunEventRepo:             p.AgentRunEventRepo,
		AiAuditService:                p.AIAuditService,
		BriefingService:               p.BriefingService,
		AgentProfileService:           p.AgentProfileService,
		AgentTools:                    p.AgentTools,
		AgentControlService:           p.AgentControlService,
		ProposalPreviewService:        p.ProposalPreviewService,
		IamService:                    p.IAMService,
		RoleService:                   p.RoleService,
		PageAccessService:             p.PageAccessService,
		UserService:                   p.UserService,
		APIKeyService:                 p.APIKeyService,
		TableChangeAlertService:       p.TableChangeAlertService,
		TableConfigurationService:     p.TableConfigurationService,
		SidebarPreferenceService:      p.SidebarPreferenceService,
		TableLayoutService:            p.TableLayoutService,
		TableInsightService:           p.TableInsightService,
		BulkEditService:               p.BulkEditService,
		TableExportService:            p.TableExportService,
		HomeLayoutService:             p.HomeLayoutService,
		NotificationService:           p.NotificationService,
		BillingTransferService:        p.BillingTransferService,
		ReportingService:              p.ReportingService,
		DriverPayService:              p.DriverPayService,
		DriverSettlementService:       p.DriverSettlementService,
		DriverPortalService:           p.DriverPortalService,
		SettlementControlService:      p.SettlementControlService,
		AccountingControlRepo:         p.AccountingControlRepo,
		DashControlService:            p.DashControlService,
		PayProfileRepo:                p.PayProfileRepo,
		RecurringDeductionRepo:        p.RecurringDeductionRepo,
		RecurringEarningRepo:          p.RecurringEarningRepo,
		PayAdvanceRepo:                p.PayAdvanceRepo,
		EscrowAccountRepo:             p.EscrowAccountRepo,
		DriverSettlementRepo:          p.DriverSettlementRepo,
		SettlementBatchRepo:           p.SettlementBatchRepo,
		PayEventRepo:                  p.PayEventRepo,
		CarrierSettlementService:      p.CarrierSettlementService,
		CarrierSettlementRepo:         p.CarrierSettlementRepo,
		CarrierSettlementBatchRepo:    p.CarrierSettlementBatchRepo,
		CarrierCostEventRepo:          p.CarrierCostEventRepo,
		FuelPurchaseService:           p.FuelPurchaseService,
		IftaService:                   p.IFTAService,
		DistanceCalculationService:    p.DistanceCalculationService,
	}
}
