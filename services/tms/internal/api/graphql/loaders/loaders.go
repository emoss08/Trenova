package loaders

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/latecharge"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type contextKey struct{}

type FactoryParams struct {
	fx.In

	OrganizationByID                          *OrganizationByIDLoaderFactory
	LocationByID                              *LocationByIDLoaderFactory
	OrderByID                                 *OrderByIDLoaderFactory
	ShipmentProfitabilityByID                 *ShipmentProfitabilityLoaderFactory
	ShipmentEtaByID                           *ShipmentEtaLoaderFactory
	EDIPartnerByCustomerID                    *EDIPartnerByCustomerIDLoaderFactory
	FormulaTemplateStatsByID                  *FormulaTemplateStatsLoaderFactory
	AgentDefinitionStatsByID                  *AgentDefinitionStatsLoaderFactory
	PayProfileActiveAssignmentCount           *PayProfileActiveAssignmentCountLoaderFactory
	PTOPolicyOpenAssignmentCount              *PTOPolicyOpenAssignmentCountLoaderFactory
	ShiftTemplateActiveAssignmentCount        *ShiftTemplateActiveAssignmentCountLoaderFactory
	AIProviderMonthSpend                      *AIProviderMonthSpendLoaderFactory
	WorkerChecklistTemplateOpenChecklistCount *WorkerChecklistTemplateOpenChecklistCountLoaderFactory
	WorkerCredentialTypeActiveCredentialCount *WorkerCredentialTypeActiveCredentialCountLoaderFactory
	TrainingCourseOpenRecordCount             *TrainingCourseOpenRecordCountLoaderFactory
	PerformanceReviewTemplateOpenReviewCount  *PerformanceReviewTemplateOpenReviewCountLoaderFactory
	CustomerByID                              *CustomerByIDLoaderFactory
	InvoiceByID                               *InvoiceByIDLoaderFactory
	GLAccountByID                             *GLAccountByIDLoaderFactory
	RoutingGuideWithEntriesByID               *RoutingGuideWithEntriesByIDLoaderFactory
	WorkerDQFVerificationsByWorkerID          *WorkerDQFVerificationsByWorkerIDLoaderFactory
	WorkerLeaveEntriesByCaseID                *WorkerLeaveEntriesByCaseIDLoaderFactory
	FiscalPeriodsByFiscalYearID               *FiscalPeriodsByFiscalYearIDLoaderFactory
	AgentDecisionsByProposalID                *AgentDecisionsByProposalIDLoaderFactory
	AgentRunByID                              *AgentRunByIDLoaderFactory
	AgentRunTranscriptByID                    *AgentRunTranscriptByIDLoaderFactory
	AgentMemoryByID                           *AgentMemoryByIDLoaderFactory
	AgentMemoryReplacement                    *AgentMemoryReplacementLoaderFactory
	AgentDefinitionByID                       *AgentDefinitionByIDLoaderFactory
	SuggestedAgentByEventKind                 *SuggestedAgentByEventKindLoaderFactory
	ThreadAgentByID                           *ThreadAgentByIDLoaderFactory
	UsableAgentByID                           *UsableAgentByIDLoaderFactory
	AccessRolesByAgentID                      *AccessRolesByAgentIDLoaderFactory
	AgentsByRoleID                            *AgentsByRoleIDLoaderFactory
	ToolTrustByAgentID                        *ToolTrustByAgentIDLoaderFactory
	DocumentTemplateKindByTemplateID          *DocumentTemplateKindByTemplateIDLoaderFactory
	IFTAJurisdictionByID                      *IFTAJurisdictionByIDLoaderFactory
	IFTAReturnByID                            *IFTAReturnByIDLoaderFactory
	FuelCardByID                              *FuelCardByIDLoaderFactory
	TractorByID                               *TractorByIDLoaderFactory
	ShipmentMoveJurisdictionMilesByMoveID     *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory
	UserByID                                  *UserByIDLoaderFactory
	DocumentByID                              *DocumentByIDLoaderFactory
	UsStateByID                               *UsStateByIDLoaderFactory
	InvoicesByShipmentID                      *InvoicesByShipmentIDLoaderFactory
	ChargeAllocationsByShipmentID             *ChargeAllocationsByShipmentIDLoaderFactory
	ChargeAllocationsByOrderChargeID          *ChargeAllocationsByOrderChargeIDLoaderFactory
	CustomerPaymentApplicationsByInvoiceID    *CustomerPaymentApplicationsByInvoiceIDLoaderFactory
	CreditMemoApplicationsByInvoiceID         *CreditMemoApplicationsByInvoiceIDLoaderFactory
	InvoiceDisputesByInvoiceID                *InvoiceDisputesByInvoiceIDLoaderFactory
	PayAdvanceByID                            *PayAdvanceByIDLoaderFactory
	EscrowAccountByID                         *EscrowAccountByIDLoaderFactory
	SettlementDisputesBySettlementID          *SettlementDisputesBySettlementIDLoaderFactory
	CarrierInvoiceMatchesByMoveID             *CarrierInvoiceMatchesByMoveIDLoaderFactory
	FactoringCompanyByID                      *FactoringCompanyByIDLoaderFactory
	FactoringStateByInvoiceID                 *FactoringStateByInvoiceIDLoaderFactory
	FactoringAgreementsByCompanyID            *FactoringAgreementsByCompanyIDLoaderFactory
	FactoringSubmissionItems                  *FactoringSubmissionItemsLoaderFactory
	FactoringSubmissionByID                   *FactoringSubmissionByIDLoaderFactory
	FactoringLiveItemByInvoiceID              *FactoringLiveItemByInvoiceIDLoaderFactory
	LateChargeAssessmentsByInvoiceID          *LateChargeAssessmentsByInvoiceIDLoaderFactory
	InvoiceEDISendPlanByInvoiceID             *InvoiceEDISendPlanByInvoiceIDLoaderFactory
	CarrierIntelSnapshotByCarrierID           *CarrierIntelSnapshotByCarrierIDLoaderFactory
	CarrierIntelSnapshotByCustomerID          *CarrierIntelSnapshotByCustomerIDLoaderFactory
	CarrierIntelOpenEventCount                *CarrierIntelOpenEventCountLoaderFactory
	InboundAttachmentCount                    *InboundAttachmentCountLoaderFactory
	InboundThreadCount                        *InboundThreadCountLoaderFactory
	InboundActivitiesByMessageID              *InboundActivitiesByMessageIDLoaderFactory
	DocumentTypeNameByID                      *DocumentTypeNameByIDLoaderFactory
	TriageViewWaitingCount                    *TriageViewWaitingCountLoaderFactory
	TriageRuleMonthMatches                    *TriageRuleMonthMatchesLoaderFactory
	TriageSampleByMessageID                   *TriageSampleByMessageIDLoaderFactory
	InboundMailboxByID                        *InboundMailboxByIDLoaderFactory
	ShipmentSummaryByID                       *ShipmentSummaryByIDLoaderFactory
	InboundAttachmentsByMessageID             *InboundAttachmentsByMessageIDLoaderFactory
	InboundAttachmentByID                     *InboundAttachmentByIDLoaderFactory
	InboundReply                              *InboundReplyLoaderFactory
	InboundDocumentReading                    *InboundDocumentReadingLoaderFactory
	CaptureRecordLabel                        *CaptureRecordLabelLoaderFactory
	CarrierMonitoringEnrollmentByCarrierID    *CarrierMonitoringEnrollmentByCarrierIDLoaderFactory
	AuditEntriesByAIAuditEventID              *AuditEntriesByAIAuditEventIDLoaderFactory
	SubsetLabels                              *SubsetLabelsLoaderFactory
	PermissionEngine                          permissionports.PermissionEngine
}

type Factory struct {
	organizationByID                          *OrganizationByIDLoaderFactory
	locationByID                              *LocationByIDLoaderFactory
	orderByID                                 *OrderByIDLoaderFactory
	shipmentProfitabilityByID                 *ShipmentProfitabilityLoaderFactory
	shipmentEtaByID                           *ShipmentEtaLoaderFactory
	ediPartnerByCustomerID                    *EDIPartnerByCustomerIDLoaderFactory
	formulaTemplateStatsByID                  *FormulaTemplateStatsLoaderFactory
	agentDefinitionStatsByID                  *AgentDefinitionStatsLoaderFactory
	payProfileActiveAssignmentCount           *PayProfileActiveAssignmentCountLoaderFactory
	pTOPolicyOpenAssignmentCount              *PTOPolicyOpenAssignmentCountLoaderFactory
	shiftTemplateActiveAssignmentCount        *ShiftTemplateActiveAssignmentCountLoaderFactory
	aiProviderMonthSpend                      *AIProviderMonthSpendLoaderFactory
	workerChecklistTemplateOpenChecklistCount *WorkerChecklistTemplateOpenChecklistCountLoaderFactory
	workerCredentialTypeActiveCredentialCount *WorkerCredentialTypeActiveCredentialCountLoaderFactory
	trainingCourseOpenRecordCount             *TrainingCourseOpenRecordCountLoaderFactory
	performanceReviewTemplateOpenReviewCount  *PerformanceReviewTemplateOpenReviewCountLoaderFactory
	customerByID                              *CustomerByIDLoaderFactory
	invoiceByID                               *InvoiceByIDLoaderFactory
	gLAccountByID                             *GLAccountByIDLoaderFactory
	routingGuideWithEntriesByID               *RoutingGuideWithEntriesByIDLoaderFactory
	workerDQFVerificationsByWorkerID          *WorkerDQFVerificationsByWorkerIDLoaderFactory
	workerLeaveEntriesByCaseID                *WorkerLeaveEntriesByCaseIDLoaderFactory
	fiscalPeriodsByFiscalYearID               *FiscalPeriodsByFiscalYearIDLoaderFactory
	agentDecisionsByProposalID                *AgentDecisionsByProposalIDLoaderFactory
	agentRunByID                              *AgentRunByIDLoaderFactory
	agentRunTranscriptByID                    *AgentRunTranscriptByIDLoaderFactory
	agentMemoryByID                           *AgentMemoryByIDLoaderFactory
	agentMemoryReplacement                    *AgentMemoryReplacementLoaderFactory
	agentDefinitionByID                       *AgentDefinitionByIDLoaderFactory
	suggestedAgentByEventKind                 *SuggestedAgentByEventKindLoaderFactory
	threadAgentByID                           *ThreadAgentByIDLoaderFactory
	usableAgentByID                           *UsableAgentByIDLoaderFactory
	accessRolesByAgentID                      *AccessRolesByAgentIDLoaderFactory
	agentsByRoleID                            *AgentsByRoleIDLoaderFactory
	toolTrustByAgentID                        *ToolTrustByAgentIDLoaderFactory
	documentTemplateKindByTemplateID          *DocumentTemplateKindByTemplateIDLoaderFactory
	iFTAJurisdictionByID                      *IFTAJurisdictionByIDLoaderFactory
	iFTAReturnByID                            *IFTAReturnByIDLoaderFactory
	fuelCardByID                              *FuelCardByIDLoaderFactory
	tractorByID                               *TractorByIDLoaderFactory
	shipmentMoveJurisdictionMilesByMoveID     *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory
	userByID                                  *UserByIDLoaderFactory
	documentByID                              *DocumentByIDLoaderFactory
	usStateByID                               *UsStateByIDLoaderFactory
	invoicesByShipmentID                      *InvoicesByShipmentIDLoaderFactory
	chargeAllocationsByShipmentID             *ChargeAllocationsByShipmentIDLoaderFactory
	chargeAllocationsByOrderChargeID          *ChargeAllocationsByOrderChargeIDLoaderFactory
	customerPaymentApplicationsByInvoiceID    *CustomerPaymentApplicationsByInvoiceIDLoaderFactory
	creditMemoApplicationsByInvoiceID         *CreditMemoApplicationsByInvoiceIDLoaderFactory
	invoiceDisputesByInvoiceID                *InvoiceDisputesByInvoiceIDLoaderFactory
	payAdvanceByID                            *PayAdvanceByIDLoaderFactory
	escrowAccountByID                         *EscrowAccountByIDLoaderFactory
	settlementDisputesBySettlementID          *SettlementDisputesBySettlementIDLoaderFactory
	carrierInvoiceMatchesByMoveID             *CarrierInvoiceMatchesByMoveIDLoaderFactory
	factoringCompanyByID                      *FactoringCompanyByIDLoaderFactory
	factoringStateByInvoiceID                 *FactoringStateByInvoiceIDLoaderFactory
	factoringAgreementsByCompanyID            *FactoringAgreementsByCompanyIDLoaderFactory
	factoringSubmissionItems                  *FactoringSubmissionItemsLoaderFactory
	factoringSubmissionByID                   *FactoringSubmissionByIDLoaderFactory
	factoringLiveItemByInvoiceID              *FactoringLiveItemByInvoiceIDLoaderFactory
	lateChargeAssessmentsByInvoiceID          *LateChargeAssessmentsByInvoiceIDLoaderFactory
	invoiceEDISendPlanByInvoiceID             *InvoiceEDISendPlanByInvoiceIDLoaderFactory
	carrierIntelSnapshotByCarrierID           *CarrierIntelSnapshotByCarrierIDLoaderFactory
	carrierIntelSnapshotByCustomerID          *CarrierIntelSnapshotByCustomerIDLoaderFactory
	carrierIntelOpenEventCount                *CarrierIntelOpenEventCountLoaderFactory
	inboundAttachmentCount                    *InboundAttachmentCountLoaderFactory
	inboundThreadCount                        *InboundThreadCountLoaderFactory
	inboundActivitiesByMessageID              *InboundActivitiesByMessageIDLoaderFactory
	documentTypeNameByID                      *DocumentTypeNameByIDLoaderFactory
	triageViewWaitingCount                    *TriageViewWaitingCountLoaderFactory
	triageRuleMonthMatches                    *TriageRuleMonthMatchesLoaderFactory
	triageSampleByMessageID                   *TriageSampleByMessageIDLoaderFactory
	inboundMailboxByID                        *InboundMailboxByIDLoaderFactory
	shipmentSummaryByID                       *ShipmentSummaryByIDLoaderFactory
	inboundAttachmentsByMessageID             *InboundAttachmentsByMessageIDLoaderFactory
	inboundAttachmentByID                     *InboundAttachmentByIDLoaderFactory
	inboundReply                              *InboundReplyLoaderFactory
	inboundDocumentReading                    *InboundDocumentReadingLoaderFactory
	captureRecordLabel                        *CaptureRecordLabelLoaderFactory
	carrierMonitoringEnrollmentByCarrierID    *CarrierMonitoringEnrollmentByCarrierIDLoaderFactory
	auditEntriesByAIAuditEventID              *AuditEntriesByAIAuditEventIDLoaderFactory
	subsetLabels                              *SubsetLabelsLoaderFactory
	permissionEngine                          permissionports.PermissionEngine
}

type set struct {
	organizationByID                          *dataloadgen.Loader[string, *tenant.Organization]
	locationByID                              *dataloadgen.Loader[string, *location.Location]
	orderByID                                 *dataloadgen.Loader[string, *order.Order]
	shipmentProfitabilityByID                 *dataloadgen.Loader[string, *costingservice.ShipmentProfitabilityEstimate]
	shipmentEtaByID                           *dataloadgen.Loader[string, *shipmenttrackingports.ShipmentEta]
	ediPartnerByCustomerID                    *dataloadgen.Loader[string, *edi.EDIPartner]
	formulaTemplateStatsByID                  *dataloadgen.Loader[string, formulatemplateports.TemplateStats]
	agentDefinitionStatsByID                  *dataloadgen.Loader[string, agentdefinitionports.AgentDefinitionStats]
	payProfileActiveAssignmentCount           *dataloadgen.Loader[string, int]
	ptoPolicyOpenAssignmentCount              *dataloadgen.Loader[string, int]
	shiftTemplateActiveAssignmentCount        *dataloadgen.Loader[string, int]
	aiProviderMonthSpend                      *dataloadgen.Loader[string, decimal.Decimal]
	workerChecklistTemplateOpenChecklistCount *dataloadgen.Loader[string, int]
	workerCredentialTypeActiveCredentialCount *dataloadgen.Loader[string, int]
	trainingCourseOpenRecordCount             *dataloadgen.Loader[string, int]
	performanceReviewTemplateOpenReviewCount  *dataloadgen.Loader[string, int]
	customerByID                              *dataloadgen.Loader[string, *customer.Customer]
	invoiceByID                               *dataloadgen.Loader[string, *invoice.Invoice]
	glAccountByID                             *dataloadgen.Loader[string, *glaccount.GLAccount]
	routingGuideWithEntriesByID               *dataloadgen.Loader[string, *tender.RoutingGuide]
	workerDQFVerificationsByWorkerID          *dataloadgen.Loader[string, []*worker.WorkerEmploymentVerification]
	workerLeaveEntriesByCaseID                *dataloadgen.Loader[string, []*worker.WorkerLeaveEntry]
	fiscalPeriodsByFiscalYearID               *dataloadgen.Loader[string, []*fiscalperiod.FiscalPeriod]
	agentDecisionsByProposalID                *dataloadgen.Loader[string, []*agent.AgentDecision]
	agentRunByID                              *dataloadgen.Loader[string, *agent.AgentRun]
	agentRunTranscriptByID                    *dataloadgen.Loader[string, *agent.AgentRun]
	agentMemoryByID                           *dataloadgen.Loader[string, *agent.Memory]
	agentMemoryReplacement                    *dataloadgen.Loader[string, *agent.Memory]
	agentDefinitionByID                       *dataloadgen.Loader[string, *agentdefinition.Definition]
	suggestedAgentByEventKind                 *dataloadgen.Loader[string, *agentdefinition.Definition]
	threadAgentByID                           *dataloadgen.Loader[string, *conversation.Thread]
	usableAgentByID                           *dataloadgen.Loader[string, *agentdefinition.Definition]
	accessRolesByAgentID                      *dataloadgen.Loader[string, []*permission.Role]
	agentsByRoleID                            *dataloadgen.Loader[string, []*agentdefinition.Definition]
	toolTrustByAgentID                        *dataloadgen.Loader[string, []*agent.ToolTrust]
	documentTemplateKindByTemplateID          *dataloadgen.Loader[string, documenttemplate.Kind]
	iftaJurisdictionByID                      *dataloadgen.Loader[string, *ifta.Jurisdiction]
	iftaReturnByID                            *dataloadgen.Loader[string, *ifta.Return]
	fuelCardByID                              *dataloadgen.Loader[string, *fuelpurchase.FuelCard]
	tractorByID                               *dataloadgen.Loader[string, *tractor.Tractor]
	shipmentMoveJurisdictionMilesByMoveID     *dataloadgen.Loader[string, []*shipment.ShipmentMoveJurisdictionMile]
	userByID                                  *dataloadgen.Loader[string, *tenant.User]
	documentByID                              *dataloadgen.Loader[string, *document.Document]
	usStateByID                               *dataloadgen.Loader[string, *usstate.UsState]
	invoicesByShipmentID                      *dataloadgen.Loader[string, []*invoice.Invoice]
	chargeAllocationsByShipmentID             *dataloadgen.Loader[string, []*shipment.ChargeAllocation]
	chargeAllocationsByOrderChargeID          *dataloadgen.Loader[string, []*shipment.ChargeAllocation]
	customerPaymentApplicationsByInvoiceID    *dataloadgen.Loader[string, []*customerpayment.Application]
	creditMemoApplicationsByInvoiceID         *dataloadgen.Loader[string, []*customerpayment.CreditMemoApplication]
	invoiceDisputesByInvoiceID                *dataloadgen.Loader[string, []*invoice.InvoiceDispute]
	payAdvanceByID                            *dataloadgen.Loader[string, *driverpay.PayAdvance]
	escrowAccountByID                         *dataloadgen.Loader[string, *driverpay.EscrowAccount]
	settlementDisputesBySettlementID          *dataloadgen.Loader[string, []*driversettlement.Dispute]
	carrierInvoiceMatchesByMoveID             *dataloadgen.Loader[string, []*carriersettlement.InvoiceMatch]
	factoringCompanyByID                      *dataloadgen.Loader[string, *factoringcompany.FactoringCompany]
	factoringStateByInvoiceID                 *dataloadgen.Loader[string, *factoring.InvoiceState]
	factoringAgreementsByCompanyID            *dataloadgen.Loader[string, []*factoring.Agreement]
	factoringSubmissionItems                  *dataloadgen.Loader[string, []*factoring.SubmissionItem]
	factoringSubmissionByID                   *dataloadgen.Loader[string, *factoring.Submission]
	factoringLiveItemByInvoiceID              *dataloadgen.Loader[string, *factoringports.LiveItem]
	lateChargeAssessmentsByInvoiceID          *dataloadgen.Loader[string, []*latecharge.LateChargeAssessment]
	invoiceEDISendPlanByInvoiceID             *dataloadgen.Loader[string, *invoiceports.InvoiceEDISendPlan]
	carrierIntelSnapshotByCarrierID           *dataloadgen.Loader[string, []*carrierintel.CarrierIntelSnapshot]
	carrierIntelSnapshotByCustomerID          *dataloadgen.Loader[string, []*carrierintel.CarrierIntelSnapshot]
	carrierIntelOpenEventCount                *dataloadgen.Loader[string, int]
	inboundAttachmentCount                    *dataloadgen.Loader[string, int]
	inboundThreadCount                        *dataloadgen.Loader[string, int]
	inboundActivitiesByMessageID              *dataloadgen.Loader[string, []*triage.Activity]
	documentTypeNameByID                      *dataloadgen.Loader[string, string]
	triageViewWaitingCount                    *dataloadgen.Loader[string, int]
	triageRuleMonthMatches                    *dataloadgen.Loader[string, int]
	triageSampleByMessageID                   *dataloadgen.Loader[string, *inboundmessageports.Sample]
	inboundMailboxByID                        *dataloadgen.Loader[string, *inboundmessage.Mailbox]
	shipmentSummaryByID                       *dataloadgen.Loader[string, *shipmentports.ShipmentSummary]
	inboundAttachmentsByMessageID             *dataloadgen.Loader[string, []*inboundmessage.InboundAttachment]
	inboundAttachmentByID                     *dataloadgen.Loader[string, *inboundmessage.InboundAttachment]
	inboundLatestReplyByMessageID             *dataloadgen.Loader[string, *triage.Reply]
	inboundReplyDraftByMessageID              *dataloadgen.Loader[string, *triage.Reply]
	inboundRepliesByMessageID                 *dataloadgen.Loader[string, []*triage.Reply]
	inboundReplyByID                          *dataloadgen.Loader[string, *triage.Reply]
	inboundDocumentReadingByDocumentID        *dataloadgen.Loader[string, *inboundmessageports.DocumentReading]
	captureRecordLabel                        *dataloadgen.Loader[string, *captureports.CaptureRecordLabel]
	carrierMonitoringEnrollmentByCarrierID    *dataloadgen.Loader[string, []*carrierintel.CarrierMonitoringEnrollment]
	auditEntriesByAIAuditEventID              *dataloadgen.Loader[string, []*audit.Entry]
	// SubsetLabels names the records each record-subset field offers.
	subsetLabels *dataloadgen.Loader[string, agentports.RecordLabels]
	// FieldCeilings is the reader's sensitivity ceiling per resource, asked
	// of the permission engine once per resource per request.
	fieldCeilings aiauditports.FieldCeilings
}

func NewFactory(p FactoryParams) *Factory {
	return &Factory{
		organizationByID:                          p.OrganizationByID,
		locationByID:                              p.LocationByID,
		orderByID:                                 p.OrderByID,
		shipmentProfitabilityByID:                 p.ShipmentProfitabilityByID,
		shipmentEtaByID:                           p.ShipmentEtaByID,
		ediPartnerByCustomerID:                    p.EDIPartnerByCustomerID,
		formulaTemplateStatsByID:                  p.FormulaTemplateStatsByID,
		agentDefinitionStatsByID:                  p.AgentDefinitionStatsByID,
		payProfileActiveAssignmentCount:           p.PayProfileActiveAssignmentCount,
		pTOPolicyOpenAssignmentCount:              p.PTOPolicyOpenAssignmentCount,
		shiftTemplateActiveAssignmentCount:        p.ShiftTemplateActiveAssignmentCount,
		aiProviderMonthSpend:                      p.AIProviderMonthSpend,
		workerChecklistTemplateOpenChecklistCount: p.WorkerChecklistTemplateOpenChecklistCount,
		workerCredentialTypeActiveCredentialCount: p.WorkerCredentialTypeActiveCredentialCount,
		trainingCourseOpenRecordCount:             p.TrainingCourseOpenRecordCount,
		performanceReviewTemplateOpenReviewCount:  p.PerformanceReviewTemplateOpenReviewCount,
		customerByID:                              p.CustomerByID,
		invoiceByID:                               p.InvoiceByID,
		gLAccountByID:                             p.GLAccountByID,
		routingGuideWithEntriesByID:               p.RoutingGuideWithEntriesByID,
		workerDQFVerificationsByWorkerID:          p.WorkerDQFVerificationsByWorkerID,
		workerLeaveEntriesByCaseID:                p.WorkerLeaveEntriesByCaseID,
		fiscalPeriodsByFiscalYearID:               p.FiscalPeriodsByFiscalYearID,
		agentDecisionsByProposalID:                p.AgentDecisionsByProposalID,
		agentRunByID:                              p.AgentRunByID,
		agentRunTranscriptByID:                    p.AgentRunTranscriptByID,
		agentMemoryByID:                           p.AgentMemoryByID,
		agentMemoryReplacement:                    p.AgentMemoryReplacement,
		agentDefinitionByID:                       p.AgentDefinitionByID,
		suggestedAgentByEventKind:                 p.SuggestedAgentByEventKind,
		threadAgentByID:                           p.ThreadAgentByID,
		usableAgentByID:                           p.UsableAgentByID,
		accessRolesByAgentID:                      p.AccessRolesByAgentID,
		agentsByRoleID:                            p.AgentsByRoleID,
		toolTrustByAgentID:                        p.ToolTrustByAgentID,
		documentTemplateKindByTemplateID:          p.DocumentTemplateKindByTemplateID,
		iFTAJurisdictionByID:                      p.IFTAJurisdictionByID,
		iFTAReturnByID:                            p.IFTAReturnByID,
		fuelCardByID:                              p.FuelCardByID,
		tractorByID:                               p.TractorByID,
		shipmentMoveJurisdictionMilesByMoveID:     p.ShipmentMoveJurisdictionMilesByMoveID,
		userByID:                                  p.UserByID,
		documentByID:                              p.DocumentByID,
		usStateByID:                               p.UsStateByID,
		invoicesByShipmentID:                      p.InvoicesByShipmentID,
		chargeAllocationsByShipmentID:             p.ChargeAllocationsByShipmentID,
		chargeAllocationsByOrderChargeID:          p.ChargeAllocationsByOrderChargeID,
		customerPaymentApplicationsByInvoiceID:    p.CustomerPaymentApplicationsByInvoiceID,
		creditMemoApplicationsByInvoiceID:         p.CreditMemoApplicationsByInvoiceID,
		invoiceDisputesByInvoiceID:                p.InvoiceDisputesByInvoiceID,
		payAdvanceByID:                            p.PayAdvanceByID,
		escrowAccountByID:                         p.EscrowAccountByID,
		settlementDisputesBySettlementID:          p.SettlementDisputesBySettlementID,
		carrierInvoiceMatchesByMoveID:             p.CarrierInvoiceMatchesByMoveID,
		factoringCompanyByID:                      p.FactoringCompanyByID,
		factoringStateByInvoiceID:                 p.FactoringStateByInvoiceID,
		factoringAgreementsByCompanyID:            p.FactoringAgreementsByCompanyID,
		factoringSubmissionItems:                  p.FactoringSubmissionItems,
		factoringSubmissionByID:                   p.FactoringSubmissionByID,
		factoringLiveItemByInvoiceID:              p.FactoringLiveItemByInvoiceID,
		lateChargeAssessmentsByInvoiceID:          p.LateChargeAssessmentsByInvoiceID,
		invoiceEDISendPlanByInvoiceID:             p.InvoiceEDISendPlanByInvoiceID,
		carrierIntelSnapshotByCarrierID:           p.CarrierIntelSnapshotByCarrierID,
		carrierIntelSnapshotByCustomerID:          p.CarrierIntelSnapshotByCustomerID,
		carrierIntelOpenEventCount:                p.CarrierIntelOpenEventCount,
		inboundAttachmentCount:                    p.InboundAttachmentCount,
		inboundThreadCount:                        p.InboundThreadCount,
		inboundActivitiesByMessageID:              p.InboundActivitiesByMessageID,
		documentTypeNameByID:                      p.DocumentTypeNameByID,
		triageViewWaitingCount:                    p.TriageViewWaitingCount,
		triageRuleMonthMatches:                    p.TriageRuleMonthMatches,
		triageSampleByMessageID:                   p.TriageSampleByMessageID,
		inboundMailboxByID:                        p.InboundMailboxByID,
		shipmentSummaryByID:                       p.ShipmentSummaryByID,
		inboundAttachmentsByMessageID:             p.InboundAttachmentsByMessageID,
		inboundAttachmentByID:                     p.InboundAttachmentByID,
		inboundReply:                              p.InboundReply,
		inboundDocumentReading:                    p.InboundDocumentReading,
		captureRecordLabel:                        p.CaptureRecordLabel,
		carrierMonitoringEnrollmentByCarrierID:    p.CarrierMonitoringEnrollmentByCarrierID,
		auditEntriesByAIAuditEventID:              p.AuditEntriesByAIAuditEventID,
		subsetLabels:                              p.SubsetLabels,
		permissionEngine:                          p.PermissionEngine,
	}
}

func newSet(f *Factory, tenantInfo pagination.TenantInfo) *set {
	return &set{
		organizationByID: newOrganizationByIDLoader(f.organizationByID, tenantInfo),
		locationByID:     newLocationByIDLoader(f.locationByID, tenantInfo),
		orderByID:        newOrderByIDLoader(f.orderByID, tenantInfo),
		shipmentProfitabilityByID: newShipmentProfitabilityLoader(f.shipmentProfitabilityByID,
			tenantInfo,
		),
		shipmentEtaByID: newShipmentEtaLoader(f.shipmentEtaByID, tenantInfo),
		ediPartnerByCustomerID: newEDIPartnerByCustomerIDLoader(f.ediPartnerByCustomerID,
			tenantInfo,
		),
		formulaTemplateStatsByID: newFormulaTemplateStatsLoader(f.formulaTemplateStatsByID,
			tenantInfo,
		),
		agentDefinitionStatsByID: newAgentDefinitionStatsLoader(f.agentDefinitionStatsByID,
			tenantInfo,
		),
		payProfileActiveAssignmentCount: newPayProfileActiveAssignmentCountLoader(
			f.payProfileActiveAssignmentCount,
			tenantInfo,
		),
		ptoPolicyOpenAssignmentCount: newPTOPolicyOpenAssignmentCountLoader(
			f.pTOPolicyOpenAssignmentCount,
			tenantInfo,
		),
		shiftTemplateActiveAssignmentCount: newShiftTemplateActiveAssignmentCountLoader(
			f.shiftTemplateActiveAssignmentCount,
			tenantInfo,
		),
		aiProviderMonthSpend: newAIProviderMonthSpendLoader(f.aiProviderMonthSpend, tenantInfo),
		workerChecklistTemplateOpenChecklistCount: newWorkerChecklistTemplateOpenChecklistCountLoader(
			f.workerChecklistTemplateOpenChecklistCount,
			tenantInfo,
		),
		workerCredentialTypeActiveCredentialCount: newWorkerCredentialTypeActiveCredentialCountLoader(
			f.workerCredentialTypeActiveCredentialCount,
			tenantInfo,
		),
		trainingCourseOpenRecordCount: newTrainingCourseOpenRecordCountLoader(
			f.trainingCourseOpenRecordCount,
			tenantInfo,
		),
		performanceReviewTemplateOpenReviewCount: newPerformanceReviewTemplateOpenReviewCountLoader(
			f.performanceReviewTemplateOpenReviewCount,
			tenantInfo,
		),
		customerByID:  newCustomerByIDLoader(f.customerByID, tenantInfo),
		invoiceByID:   newInvoiceByIDLoader(f.invoiceByID, tenantInfo),
		glAccountByID: newGLAccountByIDLoader(f.gLAccountByID, tenantInfo),
		routingGuideWithEntriesByID: newRoutingGuideWithEntriesByIDLoader(
			f.routingGuideWithEntriesByID,
			tenantInfo,
		),
		workerDQFVerificationsByWorkerID: newWorkerDQFVerificationsByWorkerIDLoader(
			f.workerDQFVerificationsByWorkerID,
			tenantInfo,
		),
		workerLeaveEntriesByCaseID: newWorkerLeaveEntriesByCaseIDLoader(
			f.workerLeaveEntriesByCaseID,
			tenantInfo,
		),
		fiscalPeriodsByFiscalYearID: newFiscalPeriodsByFiscalYearIDLoader(
			f.fiscalPeriodsByFiscalYearID,
			tenantInfo,
		),
		agentDecisionsByProposalID: newAgentDecisionsByProposalIDLoader(
			f.agentDecisionsByProposalID,
			tenantInfo,
		),
		agentRunByID:    newAgentRunByIDLoader(f.agentRunByID, tenantInfo),
		agentMemoryByID: newAgentMemoryByIDLoader(f.agentMemoryByID, tenantInfo),
		agentMemoryReplacement: newAgentMemoryReplacementLoader(f.agentMemoryReplacement,
			tenantInfo,
		),
		agentRunTranscriptByID: newAgentRunTranscriptByIDLoader(f.agentRunTranscriptByID,
			tenantInfo,
		),
		agentDefinitionByID: newAgentDefinitionByIDLoader(f.agentDefinitionByID, tenantInfo),
		suggestedAgentByEventKind: newSuggestedAgentByEventKindLoader(
			f.suggestedAgentByEventKind,
			tenantInfo,
		),
		threadAgentByID:      newThreadAgentByIDLoader(f.threadAgentByID, tenantInfo),
		usableAgentByID:      newUsableAgentByIDLoader(f.usableAgentByID, tenantInfo),
		accessRolesByAgentID: newAccessRolesByAgentIDLoader(f.accessRolesByAgentID, tenantInfo),
		agentsByRoleID:       newAgentsByRoleIDLoader(f.agentsByRoleID, tenantInfo),
		toolTrustByAgentID:   newToolTrustByAgentIDLoader(f.toolTrustByAgentID, tenantInfo),
		documentTemplateKindByTemplateID: newDocumentTemplateKindByTemplateIDLoader(
			f.documentTemplateKindByTemplateID,
			tenantInfo,
		),
		iftaJurisdictionByID: newIFTAJurisdictionByIDLoader(f.iFTAJurisdictionByID, tenantInfo),
		iftaReturnByID:       newIFTAReturnByIDLoader(f.iFTAReturnByID, tenantInfo),
		fuelCardByID:         newFuelCardByIDLoader(f.fuelCardByID, tenantInfo),
		tractorByID:          newTractorByIDLoader(f.tractorByID, tenantInfo),
		shipmentMoveJurisdictionMilesByMoveID: newShipmentMoveJurisdictionMilesByMoveIDLoader(
			f.shipmentMoveJurisdictionMilesByMoveID,
			tenantInfo,
		),
		userByID:             newUserByIDLoader(f.userByID, tenantInfo),
		documentByID:         newDocumentByIDLoader(f.documentByID, tenantInfo),
		usStateByID:          newUsStateByIDLoader(f.usStateByID, tenantInfo),
		invoicesByShipmentID: newInvoicesByShipmentIDLoader(f.invoicesByShipmentID, tenantInfo),
		chargeAllocationsByShipmentID: newChargeAllocationsByShipmentIDLoader(
			f.chargeAllocationsByShipmentID,
			tenantInfo,
		),
		chargeAllocationsByOrderChargeID: newChargeAllocationsByOrderChargeIDLoader(
			f.chargeAllocationsByOrderChargeID,
			tenantInfo,
		),
		customerPaymentApplicationsByInvoiceID: newCustomerPaymentApplicationsByInvoiceIDLoader(
			f.customerPaymentApplicationsByInvoiceID,
			tenantInfo,
		),
		creditMemoApplicationsByInvoiceID: newCreditMemoApplicationsByInvoiceIDLoader(
			f.creditMemoApplicationsByInvoiceID,
			tenantInfo,
		),
		invoiceDisputesByInvoiceID: newInvoiceDisputesByInvoiceIDLoader(
			f.invoiceDisputesByInvoiceID,
			tenantInfo,
		),
		payAdvanceByID:    newPayAdvanceByIDLoader(f.payAdvanceByID, tenantInfo),
		escrowAccountByID: newEscrowAccountByIDLoader(f.escrowAccountByID, tenantInfo),
		settlementDisputesBySettlementID: newSettlementDisputesBySettlementIDLoader(
			f.settlementDisputesBySettlementID,
			tenantInfo,
		),
		carrierInvoiceMatchesByMoveID: newCarrierInvoiceMatchesByMoveIDLoader(
			f.carrierInvoiceMatchesByMoveID,
			tenantInfo,
		),
		factoringCompanyByID: newFactoringCompanyByIDLoader(
			f.factoringCompanyByID,
			tenantInfo,
		),
		factoringStateByInvoiceID: newFactoringStateByInvoiceIDLoader(
			f.factoringStateByInvoiceID,
			tenantInfo,
		),
		factoringAgreementsByCompanyID: newFactoringAgreementsByCompanyIDLoader(
			f.factoringAgreementsByCompanyID,
			tenantInfo,
		),
		factoringSubmissionItems: newFactoringSubmissionItemsLoader(
			f.factoringSubmissionItems,
			tenantInfo,
		),
		factoringSubmissionByID: newFactoringSubmissionByIDLoader(
			f.factoringSubmissionByID,
			tenantInfo,
		),
		factoringLiveItemByInvoiceID: newFactoringLiveItemByInvoiceIDLoader(
			f.factoringLiveItemByInvoiceID,
			tenantInfo,
		),
		lateChargeAssessmentsByInvoiceID: newLateChargeAssessmentsByInvoiceIDLoader(
			f.lateChargeAssessmentsByInvoiceID,
			tenantInfo,
		),
		invoiceEDISendPlanByInvoiceID: newInvoiceEDISendPlanByInvoiceIDLoader(
			f.invoiceEDISendPlanByInvoiceID,
			tenantInfo,
		),
		carrierIntelSnapshotByCarrierID: newCarrierIntelSnapshotByCarrierIDLoader(
			f.carrierIntelSnapshotByCarrierID,
			tenantInfo,
		),
		carrierIntelSnapshotByCustomerID: newCarrierIntelSnapshotByCustomerIDLoader(
			f.carrierIntelSnapshotByCustomerID,
			tenantInfo,
		),
		carrierIntelOpenEventCount: newCarrierIntelOpenEventCountLoader(
			f.carrierIntelOpenEventCount,
			tenantInfo,
		),
		inboundAttachmentCount: newInboundAttachmentCountLoader(f.inboundAttachmentCount,
			tenantInfo,
		),
		inboundThreadCount: newInboundThreadCountLoader(f.inboundThreadCount, tenantInfo),
		inboundActivitiesByMessageID: newInboundActivitiesByMessageIDLoader(
			f.inboundActivitiesByMessageID,
			tenantInfo,
		),
		documentTypeNameByID: newDocumentTypeNameByIDLoader(f.documentTypeNameByID, tenantInfo),
		triageViewWaitingCount: newTriageViewWaitingCountLoader(
			f.triageViewWaitingCount,
			tenantInfo,
		),
		triageRuleMonthMatches: newTriageRuleMonthMatchesLoader(
			f.triageRuleMonthMatches,
			tenantInfo,
		),
		triageSampleByMessageID: newTriageSampleByMessageIDLoader(
			f.triageSampleByMessageID,
			tenantInfo,
		),
		inboundMailboxByID:  newInboundMailboxByIDLoader(f.inboundMailboxByID, tenantInfo),
		shipmentSummaryByID: newShipmentSummaryByIDLoader(f.shipmentSummaryByID, tenantInfo),
		inboundAttachmentsByMessageID: newInboundAttachmentsByMessageIDLoader(
			f.inboundAttachmentsByMessageID,
			tenantInfo,
		),
		inboundAttachmentByID: newInboundAttachmentByIDLoader(f.inboundAttachmentByID, tenantInfo),
		inboundLatestReplyByMessageID: newInboundLatestReplyByMessageIDLoader(
			f.inboundReply,
			tenantInfo,
		),
		inboundReplyDraftByMessageID: newInboundReplyDraftByMessageIDLoader(
			f.inboundReply,
			tenantInfo,
		),
		inboundRepliesByMessageID: newInboundRepliesByMessageIDLoader(
			f.inboundReply,
			tenantInfo,
		),
		inboundReplyByID: newInboundReplyByIDLoader(f.inboundReply, tenantInfo),
		inboundDocumentReadingByDocumentID: newInboundDocumentReadingByDocumentIDLoader(
			f.inboundDocumentReading,
			tenantInfo,
		),
		captureRecordLabel: newCaptureRecordLabelLoader(f.captureRecordLabel, tenantInfo),
		carrierMonitoringEnrollmentByCarrierID: newCarrierMonitoringEnrollmentByCarrierIDLoader(
			f.carrierMonitoringEnrollmentByCarrierID,
			tenantInfo,
		),
		auditEntriesByAIAuditEventID: newAuditEntriesByAIAuditEventIDLoader(
			f.auditEntriesByAIAuditEventID,
			tenantInfo,
		),
		subsetLabels: newSubsetLabelsLoader(f.subsetLabels, tenantInfo),
		fieldCeilings: fieldsensitivity.NewCeilings(
			f.permissionEngine,
			tenantInfo.UserID,
			tenantInfo.OrgID,
		),
	}
}
