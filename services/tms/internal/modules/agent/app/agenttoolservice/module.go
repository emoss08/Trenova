package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/invoiceshareservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicerunservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationcategoryports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("agent-tool-service", fx.Provide(append(grouped(), NewRegistry)...))

// ToolProviders are the constructors of every tool this package registers.
// The module provides them into the group, and the description contract test
// builds each one to read what a model is shown.
func ToolProviders() []any {
	providers := []any{
		provideTransitionToInReviewTool,
		provideTransitionItemsToInReviewTool,
		provideAssignBillersTool,
		provideTransferToBillingTool,
		provideApproveBillingQueueItemTool,
		provideSendBackToOpsTool,
		provideMoveToExceptionTool,
		provideHoldBillingQueueItemTool,
		provideCancelBillingQueueItemTool,
		provideAssignBillerTool,
		providePostInvoiceTool,
		provideSendInvoiceTool,
		providePostInvoicesTool,
		provideSendInvoicesTool,
		provideApproveBillingQueueItemsTool,
		provideOpenInvoiceDisputeTool,
		provideResolveInvoiceDisputeTool,
		provideWithdrawInvoiceDisputeTool,
		provideApplyCustomerPaymentTool,
		provideReverseCustomerPaymentTool,
		provideApplyCreditMemoTool,
		provideUnapplyCreditMemoTool,
		provideSaveInvoiceAdjustmentDraftTool,
		provideSubmitInvoiceAdjustmentTool,
		provideApproveInvoiceAdjustmentTool,
		provideRejectInvoiceAdjustmentTool,
		provideBuildInvoiceRunTool,
		provideAdjustInvoiceRunMembershipTool,
		provideCommitInvoiceRunTool,
		provideCancelInvoiceRunTool,
		provideBillStatementNowTool,
		provideAssessLateChargesTool,
		provideShareInvoiceTool,
		provideUpdateInvoiceDraftTool,
		provideGenerateInvoicePDFTool,
		provideCreateInvoiceTool,
		provideCreateInvoiceMemoTool,
		provideVoidInvoiceTool,
		provideSendInvoiceEDITool,
		newCorrectChargeCodeTool,
		newSaveTableViewTool,
		newCreateDashboardTool,
		newAddDashboardTileTool,
		newScheduleReportTool,
		newCreateTableChangeAlertTool,
		newRequestMissingDocsTool,
		provideAttachDocumentTool,
		newFlagManualReviewTool,
		newAssignMoveTool,
		newRaiseExceptionTool,
		newWaitUntilTool,
		newCancelWaitTool,
		newAddShipmentCommentTool,
		newPlaceShipmentHoldTool,
		newReleaseShipmentHoldTool,
		provideCancelShipmentTool,
		provideRecordStopActualTool,
		provideUpdateTractorStatusTool,
		provideUpdateTrailerStatusTool,
		provideApproveWorkerPTOTool,
		provideRejectWorkerPTOTool,
		provideCancelWorkerPTOTool,
		provideCreateReportTool,
		provideUpdateReportTool,
		provideForkReportTool,
		provideEvaluateServiceFailuresTool,
		provideResolveServiceFailureTool,
		provideNotifyDriverTool,
		newEmailCustomerTool,
		provideLinkInboundMessageTool,
		provideMarkInboundMessageTool,
		newReplyToInboundMessageTool,
		provideSendDetentionNoticeTool,
		provideWaiveDetentionTool,
		provideCreateShipmentTool,
		provideUpdateShipmentTool,
		provideRescheduleStopTool,
		provideAddShipmentChargeTool,
		provideCreateLocationTool,
		provideTenderToRoutingGuideTool,
		provideTenderToCarriersTool,
		newUnassignMovesTool,
		newUpdateMoveStatusTool,
		provideAssignMoveToCarrierTool,
		provideCancelCarrierAssignmentTool,
		provideCancelTenderTool,
		provideRecordTenderResponseTool,
		provideGenerateRateConfirmationTool,
		provideSendRateConfirmationTool,
		provideVoidRateConfirmationTool,
		provideRecordRateConfirmationConfirmedTool,
		newRememberTool,
		newForgetMemoryTool,
		provideDismissInsightTool,
		provideCheckAccountingConnectionTool,
		provideSetAccountingMappingTool,
		provideClearAccountingMappingTool,
		provideCreateAccountingReferenceRecordTool,
		provideRefreshAccountingReferenceDataTool,
		provideRetryAccountingSyncTool,
		provideSkipAccountingSyncTool,
		provideRedateAccountingSyncTool,
		provideApplyAccountingInboundChangeTool,
		provideIgnoreAccountingInboundChangeTool,
		provideResolveAccountingDriftTool,
		provideDismissAccountingDriftTool,
		provideCheckAccountingDriftTool,
		providePauseAccountingSyncTool,
		provideResumeAccountingSyncTool,
		provideRequestAccountingBackfillTool,
		provideMatchBankReceiptTool,
		providePostCustomerPaymentTool,
		provideResolveBankReceiptWorkItemTool,
		provideEscalateDetentionTool,
		provideApproveDetentionTool,
		provideRequestCredentialRenewalTool,
		providePlaceWorkerDispatchHoldTool,
		provideAcknowledgeCarrierIntelEventTool,
		provideResolveCarrierIntelEventTool,
		newAddHomeWidgetTool,
		newRemoveHomeWidgetTool,
		newArrangeHomeLayoutTool,
	}

	providers = append(providers, ediToolProviders()...)
	providers = append(providers, ledgerToolProviders()...)
	providers = append(providers, operationsToolProviders()...)
	providers = append(providers, ratesToolProviders()...)
	providers = append(providers, workforceToolProviders()...)
	providers = append(providers, masterDataToolProviders()...)

	return append(providers, settlementToolProviders()...)
}

func grouped() []any {
	providers := ToolProviders()
	annotated := make([]any, 0, len(providers)+1)
	for _, provider := range providers {
		annotated = append(annotated, fx.Annotate(provider, fx.ResultTags(`group:"agent_tools"`)))
	}

	return annotated
}

// The tools take the narrow interface they actually use, and the widening
// happens here. That is also what lets each tool be tested without standing up
// the audit log, the realtime bus and the PTO ledger behind it.
//
// What is asked for here has to be what the graph holds, which is not the same
// question as what the package exports. The fleet services are provided as
// concrete structs; workerptoservice is provided through fx.As, so only
// services.WorkerPTOService is in the graph and asking for the struct builds
// fine and fails at startup.

func provideUpdateTractorStatusTool(tractors *tractorservice.Service) agentports.AgentTool {
	return newUpdateTractorStatusTool(tractors)
}

func provideUpdateTrailerStatusTool(trailers *trailerservice.Service) agentports.AgentTool {
	return newUpdateTrailerStatusTool(trailers)
}

func provideApproveWorkerPTOTool(pto workerports.WorkerPTOService) agentports.AgentTool {
	return newApproveWorkerPTOTool(pto)
}

func provideRejectWorkerPTOTool(pto workerports.WorkerPTOService) agentports.AgentTool {
	return newRejectWorkerPTOTool(pto)
}

func provideCancelWorkerPTOTool(pto workerports.WorkerPTOService) agentports.AgentTool {
	return newCancelWorkerPTOTool(pto)
}

func provideCreateReportTool(reports *reporting.Service) agentports.AgentTool {
	return newCreateReportTool(reports)
}

func provideUpdateReportTool(reports *reporting.Service) agentports.AgentTool {
	return newUpdateReportTool(reports)
}

func provideForkReportTool(reports *reporting.Service) agentports.AgentTool {
	return newForkReportTool(reports)
}

func provideEvaluateServiceFailuresTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentTool {
	return newEvaluateServiceFailuresTool(failures)
}

func provideResolveServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentTool {
	return newResolveServiceFailureTool(failures)
}

func provideNotifyDriverTool(drivers *drivernotificationservice.Service) agentports.AgentTool {
	return newNotifyDriverTool(drivers)
}

func provideSendDetentionNoticeTool(detention *detentionservice.Service) agentports.AgentTool {
	return newSendDetentionNoticeTool(detention)
}

func provideWaiveDetentionTool(detention *detentionservice.Service) agentports.AgentTool {
	return newWaiveDetentionTool(detention)
}

// The intake and tender tools take the concrete services, which fx provides
// as such; the tools themselves depend on the narrow interfaces they use.

type intakeToolParams struct {
	fx.In

	Logger    *zap.Logger
	Shipments shipmentports.ShipmentService
	Locations locationports.LocationRepository
	Imports   shipmentports.ShipmentImportAssistantService `optional:"true"`
}

func provideCreateShipmentTool(p intakeToolParams) agentports.AgentTool {
	deps := createShipmentDeps{
		Shipments: p.Shipments,
		Contracts: p.Shipments,
		Locations: p.Locations,
		Logger:    p.Logger,
	}
	if p.Imports != nil {
		deps.Imports = p.Imports
	}

	return newCreateShipmentTool(deps)
}

func provideCreateLocationTool(
	locations *locationservice.Service,
	states usstateports.UsStateRepository,
	categories locationcategoryports.LocationCategoryRepository,
) agentports.AgentTool {
	return newCreateLocationTool(locations, states, categories)
}

func provideUpdateShipmentTool(
	shipments shipmentports.ShipmentService,
	partners *ediservice.Service,
) agentports.AgentTool {
	return newUpdateShipmentTool(shipments, partners)
}

func provideRescheduleStopTool(
	shipments shipmentports.ShipmentService,
	partners *ediservice.Service,
) agentports.AgentTool {
	return newRescheduleStopTool(shipments, partners)
}

func provideAddShipmentChargeTool(
	shipments shipmentports.ShipmentService,
	accessorials accessorialchargeports.AccessorialChargeRepository,
) agentports.AgentTool {
	return newAddShipmentChargeTool(shipments, accessorials)
}

func provideCancelShipmentTool(
	shipments shipmentports.ShipmentService,
	partners *ediservice.Service,
	tenders *tenderservice.Service,
) agentports.AgentTool {
	return newCancelShipmentTool(shipments, partners, tenders)
}

func provideTenderToRoutingGuideTool(tenders *tenderservice.Service) agentports.AgentTool {
	return newTenderToRoutingGuideTool(tenders)
}

func provideTenderToCarriersTool(tenders *tenderservice.Service) agentports.AgentTool {
	return newTenderToCarriersTool(tenders)
}

func provideDismissInsightTool(insights *insightservice.Service) agentports.AgentTool {
	return newDismissInsightTool(insights)
}

func provideMatchBankReceiptTool(receipts *bankreceiptservice.Service) agentports.AgentTool {
	return newMatchBankReceiptTool(receipts)
}

func providePostCustomerPaymentTool(
	payments customerpaymentports.CustomerPaymentService,
	receipts *bankreceiptservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentTool {
	return newPostCustomerPaymentTool(payments, receipts, permissions)
}

func provideResolveBankReceiptWorkItemTool(
	items *bankreceiptworkitemservice.Service,
) agentports.AgentTool {
	return newResolveBankReceiptWorkItemTool(items)
}

func provideEscalateDetentionTool(detention *detentionservice.Service) agentports.AgentTool {
	return newEscalateDetentionTool(detention)
}

func provideTransferToBillingTool(
	shipments shipmentports.ShipmentService,
	runs *billingtransferservice.Service,
) agentports.AgentTool {
	return newTransferToBillingTool(shipments, runs)
}

// The billing queue decision tools take the narrow billingQueueDecider; fx
// holds the service port, so the widening happens here.

func provideApproveBillingQueueItemTool(
	billing billingqueueports.BillingQueueService,
	invoices *invoiceservice.Service,
) agentports.AgentTool {
	return newApproveBillingQueueItemTool(billing, invoices)
}

func provideSendBackToOpsTool(billing billingqueueports.BillingQueueService) agentports.AgentTool {
	return newSendBackToOpsTool(billing)
}

func provideMoveToExceptionTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return newMoveToExceptionTool(billing)
}

func provideHoldBillingQueueItemTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return newHoldBillingQueueItemTool(billing)
}

func provideCancelBillingQueueItemTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return newCancelBillingQueueItemTool(billing)
}

func provideAssignBillerTool(billing billingqueueports.BillingQueueService) agentports.AgentTool {
	return newAssignBillerTool(billing)
}

func provideAssignBillersTool(billing billingqueueports.BillingQueueService) agentports.AgentTool {
	return newAssignBillersTool(billing)
}

func provideTransitionToInReviewTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return newTransitionToInReviewTool(billing)
}

func provideTransitionItemsToInReviewTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return newTransitionItemsToInReviewTool(billing)
}

func providePostInvoiceTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return newPostInvoiceTool(invoices)
}

func provideSendInvoiceTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return newSendInvoiceTool(invoices)
}

func providePostInvoicesTool(
	invoices *invoiceservice.Service,
	numbers invoiceports.InvoiceRepository,
) agentports.AgentTool {
	return newPostInvoicesTool(invoices, numbers)
}

func provideSendInvoicesTool(
	invoices *invoiceservice.Service,
	numbers invoiceports.InvoiceRepository,
) agentports.AgentTool {
	return newSendInvoicesTool(invoices, numbers)
}

func provideApproveBillingQueueItemsTool(
	billing billingqueueports.BillingQueueService,
	invoices *invoiceservice.Service,
) agentports.AgentTool {
	return newApproveBillingQueueItemsTool(billing, invoices)
}

func provideApproveDetentionTool(detention *detentionservice.Service) agentports.AgentTool {
	return newApproveDetentionTool(detention)
}

func provideRequestCredentialRenewalTool(
	credentials *workercredentialservice.Service,
) agentports.AgentTool {
	return newRequestCredentialRenewalTool(credentials)
}

func providePlaceWorkerDispatchHoldTool(workers *workerservice.Service) agentports.AgentTool {
	return newPlaceWorkerDispatchHoldTool(workers)
}

func provideAcknowledgeCarrierIntelEventTool(
	intel *carrierintelservice.Service,
) agentports.AgentTool {
	return newAcknowledgeCarrierIntelEventTool(intel)
}

func provideResolveCarrierIntelEventTool(intel *carrierintelservice.Service) agentports.AgentTool {
	return newResolveCarrierIntelEventTool(intel)
}

// The tool takes a narrow interface so it can be tested without the whole
// document stack; fx holds the concrete service, so the widening happens here.
func provideAttachDocumentTool(documents *documentservice.Service) agentports.AgentTool {
	return newAttachDocumentTool(documents)
}

func provideLinkInboundMessageTool(inbox *inboundmessageservice.Service) agentports.AgentTool {
	return newLinkInboundMessageTool(inbox)
}

func provideMarkInboundMessageTool(inbox *inboundmessageservice.Service) agentports.AgentTool {
	return newMarkInboundMessageTool(inbox)
}

func ediToolProviders() []any {
	return []any{
		provideAcceptEDILoadTenderTool,
		provideDeclineEDILoadTenderTool,
		provideCancelEDILoadTenderTool,
		provideExpireEDILoadTenderTool,
		provideReviewEDITenderChangeTool,
		provideReviewEDITransferChangeTool,
		provideRetryEDIMessageDeliveryTool,
		provideReplayEDIMessageTool,
		provideReprocessEDIInboundFilesTool,
		provideSendEDILoadTenderTool,
		provideSendEDIStatusUpdateTool,
	}
}

func provideAcceptEDILoadTenderTool(edi *ediservice.Service) agentports.AgentTool {
	return newAcceptEDILoadTenderTool(edi)
}

func provideDeclineEDILoadTenderTool(edi *ediservice.Service) agentports.AgentTool {
	return newDeclineEDILoadTenderTool(edi)
}

func provideCancelEDILoadTenderTool(edi *ediservice.Service) agentports.AgentTool {
	return newCancelEDILoadTenderTool(edi)
}

func provideExpireEDILoadTenderTool(edi *ediservice.Service) agentports.AgentTool {
	return newExpireEDILoadTenderTool(edi)
}

func provideReviewEDITenderChangeTool(edi *ediservice.Service) agentports.AgentTool {
	return newReviewEDITenderChangeTool(edi)
}

func provideReviewEDITransferChangeTool(edi *ediservice.Service) agentports.AgentTool {
	return newReviewEDITransferChangeTool(edi)
}

func provideRetryEDIMessageDeliveryTool(edi *ediservice.Service) agentports.AgentTool {
	return newRetryEDIMessageDeliveryTool(edi)
}

func provideReplayEDIMessageTool(edi *ediservice.Service) agentports.AgentTool {
	return newReplayEDIMessageTool(edi)
}

func provideReprocessEDIInboundFilesTool(inbound *ediinboundservice.Service) agentports.AgentTool {
	return newReprocessEDIInboundFilesTool(inbound)
}

func provideSendEDILoadTenderTool(edi *ediservice.Service) agentports.AgentTool {
	return newSendEDILoadTenderTool(edi)
}

func provideSendEDIStatusUpdateTool(edi *ediservice.Service) agentports.AgentTool {
	return newSendEDIStatusUpdateTool(edi)
}

func provideOpenInvoiceDisputeTool(
	disputes invoiceports.InvoiceDisputeService,
) agentports.AgentTool {
	return newOpenInvoiceDisputeTool(disputes)
}

func provideResolveInvoiceDisputeTool(
	disputes invoiceports.InvoiceDisputeService,
) agentports.AgentTool {
	return newResolveInvoiceDisputeTool(disputes)
}

func provideWithdrawInvoiceDisputeTool(
	disputes invoiceports.InvoiceDisputeService,
) agentports.AgentTool {
	return newWithdrawInvoiceDisputeTool(disputes)
}

func provideApplyCustomerPaymentTool(
	payments customerpaymentports.CustomerPaymentService,
) agentports.AgentTool {
	return newApplyCustomerPaymentTool(payments)
}

func provideReverseCustomerPaymentTool(
	payments customerpaymentports.CustomerPaymentService,
) agentports.AgentTool {
	return newReverseCustomerPaymentTool(payments)
}

func provideApplyCreditMemoTool(
	payments customerpaymentports.CustomerPaymentService,
) agentports.AgentTool {
	return newApplyCreditMemoTool(payments)
}

func provideUnapplyCreditMemoTool(
	payments customerpaymentports.CustomerPaymentService,
) agentports.AgentTool {
	return newUnapplyCreditMemoTool(payments)
}

func provideSaveInvoiceAdjustmentDraftTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agentports.AgentTool {
	return newSaveInvoiceAdjustmentDraftTool(adjustments)
}

func provideSubmitInvoiceAdjustmentTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agentports.AgentTool {
	return newSubmitInvoiceAdjustmentTool(adjustments)
}

func provideApproveInvoiceAdjustmentTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agentports.AgentTool {
	return newApproveInvoiceAdjustmentTool(adjustments)
}

func provideRejectInvoiceAdjustmentTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agentports.AgentTool {
	return newRejectInvoiceAdjustmentTool(adjustments)
}

func provideBuildInvoiceRunTool(runs *invoicerunservice.Service) agentports.AgentTool {
	return newBuildInvoiceRunTool(runs)
}

func provideAdjustInvoiceRunMembershipTool(runs *invoicerunservice.Service) agentports.AgentTool {
	return newAdjustInvoiceRunMembershipTool(runs)
}

func provideCommitInvoiceRunTool(runs *invoicerunservice.Service) agentports.AgentTool {
	return newCommitInvoiceRunTool(runs)
}

func provideCancelInvoiceRunTool(runs *invoicerunservice.Service) agentports.AgentTool {
	return newCancelInvoiceRunTool(runs)
}

func provideBillStatementNowTool(runs *invoicerunservice.Service) agentports.AgentTool {
	return newBillStatementNowTool(runs)
}

func provideAssessLateChargesTool(charges latechargeports.LateChargeService) agentports.AgentTool {
	return newAssessLateChargesTool(charges)
}

func provideShareInvoiceTool(shares *invoiceshareservice.Service) agentports.AgentTool {
	return newShareInvoiceTool(shares)
}

func provideUpdateInvoiceDraftTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return newUpdateInvoiceDraftTool(invoices)
}

func provideGenerateInvoicePDFTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return newGenerateInvoicePDFTool(invoices)
}

func provideCreateInvoiceTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return newCreateInvoiceTool(invoices)
}

func provideCreateInvoiceMemoTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return newCreateInvoiceMemoTool(invoices)
}

func provideVoidInvoiceTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return newVoidInvoiceTool(invoices)
}

func provideSendInvoiceEDITool(invoices *invoiceservice.Service) agentports.AgentTool {
	return newSendInvoiceEDITool(invoices)
}
