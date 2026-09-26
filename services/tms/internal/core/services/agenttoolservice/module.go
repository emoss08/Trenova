package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptservice"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/internal/core/services/billingtransferservice"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
	"github.com/emoss08/trenova/internal/core/services/invoicerunservice"
	"github.com/emoss08/trenova/internal/core/services/invoiceservice"
	"github.com/emoss08/trenova/internal/core/services/invoiceshareservice"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("agent-tool-service", fx.Provide(append(grouped(), NewRegistry)...))

// ToolProviders are the constructors of every tool this package registers.
// The module provides them into the group, and the description contract test
// builds each one to read what a model is shown.
func ToolProviders() []any {
	providers := []any{
		newTransitionToInReviewTool,
		provideTransferToBillingTool,
		provideApproveBillingQueueItemTool,
		provideSendBackToOpsTool,
		provideMoveToExceptionTool,
		provideHoldBillingQueueItemTool,
		provideCancelBillingQueueItemTool,
		provideAssignBillerTool,
		providePostInvoiceTool,
		provideSendInvoiceTool,
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
		newAddShipmentCommentTool,
		newPlaceShipmentHoldTool,
		newReleaseShipmentHoldTool,
		provideCancelShipmentTool,
		newRecordStopActualTool,
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

func provideUpdateTractorStatusTool(tractors *tractorservice.Service) services.AgentTool {
	return newUpdateTractorStatusTool(tractors)
}

func provideUpdateTrailerStatusTool(trailers *trailerservice.Service) services.AgentTool {
	return newUpdateTrailerStatusTool(trailers)
}

func provideApproveWorkerPTOTool(pto services.WorkerPTOService) services.AgentTool {
	return newApproveWorkerPTOTool(pto)
}

func provideRejectWorkerPTOTool(pto services.WorkerPTOService) services.AgentTool {
	return newRejectWorkerPTOTool(pto)
}

func provideCancelWorkerPTOTool(pto services.WorkerPTOService) services.AgentTool {
	return newCancelWorkerPTOTool(pto)
}

func provideCreateReportTool(reports *reporting.Service) services.AgentTool {
	return newCreateReportTool(reports)
}

func provideUpdateReportTool(reports *reporting.Service) services.AgentTool {
	return newUpdateReportTool(reports)
}

func provideForkReportTool(reports *reporting.Service) services.AgentTool {
	return newForkReportTool(reports)
}

func provideEvaluateServiceFailuresTool(
	failures services.ServiceFailureService,
) services.AgentTool {
	return newEvaluateServiceFailuresTool(failures)
}

func provideResolveServiceFailureTool(failures services.ServiceFailureService) services.AgentTool {
	return newResolveServiceFailureTool(failures)
}

func provideNotifyDriverTool(drivers *drivernotificationservice.Service) services.AgentTool {
	return newNotifyDriverTool(drivers)
}

func provideSendDetentionNoticeTool(detention *detentionservice.Service) services.AgentTool {
	return newSendDetentionNoticeTool(detention)
}

func provideWaiveDetentionTool(detention *detentionservice.Service) services.AgentTool {
	return newWaiveDetentionTool(detention)
}

// The intake and tender tools take the concrete services, which fx provides
// as such; the tools themselves depend on the narrow interfaces they use.

type intakeToolParams struct {
	fx.In

	Logger    *zap.Logger
	Shipments services.ShipmentService
	Imports   services.ShipmentImportAssistantService `optional:"true"`
}

func provideCreateShipmentTool(p intakeToolParams) services.AgentTool {
	var imports importCompleter
	if p.Imports != nil {
		imports = p.Imports
	}

	return newCreateShipmentTool(p.Shipments, imports, p.Logger)
}

func provideCreateLocationTool(
	locations *locationservice.Service,
	states repositories.UsStateRepository,
	categories repositories.LocationCategoryRepository,
) services.AgentTool {
	return newCreateLocationTool(locations, states, categories)
}

func provideUpdateShipmentTool(
	shipments services.ShipmentService,
	partners *ediservice.Service,
) services.AgentTool {
	return newUpdateShipmentTool(shipments, partners)
}

func provideCancelShipmentTool(
	shipments services.ShipmentService,
	partners *ediservice.Service,
	tenders *tenderservice.Service,
) services.AgentTool {
	return newCancelShipmentTool(shipments, partners, tenders)
}

func provideTenderToRoutingGuideTool(tenders *tenderservice.Service) services.AgentTool {
	return newTenderToRoutingGuideTool(tenders)
}

func provideTenderToCarriersTool(tenders *tenderservice.Service) services.AgentTool {
	return newTenderToCarriersTool(tenders)
}

func provideDismissInsightTool(insights *insightservice.Service) services.AgentTool {
	return newDismissInsightTool(insights)
}

func provideMatchBankReceiptTool(receipts *bankreceiptservice.Service) services.AgentTool {
	return newMatchBankReceiptTool(receipts)
}

func providePostCustomerPaymentTool(
	payments services.CustomerPaymentService,
	receipts *bankreceiptservice.Service,
	permissions services.PermissionEngine,
) services.AgentTool {
	return newPostCustomerPaymentTool(payments, receipts, permissions)
}

func provideResolveBankReceiptWorkItemTool(
	items *bankreceiptworkitemservice.Service,
) services.AgentTool {
	return newResolveBankReceiptWorkItemTool(items)
}

func provideEscalateDetentionTool(detention *detentionservice.Service) services.AgentTool {
	return newEscalateDetentionTool(detention)
}

func provideTransferToBillingTool(
	shipments services.ShipmentService,
	runs *billingtransferservice.Service,
) services.AgentTool {
	return newTransferToBillingTool(shipments, runs)
}

// The billing queue decision tools take the narrow billingQueueDecider; fx
// holds the service port, so the widening happens here.

func provideApproveBillingQueueItemTool(
	billing services.BillingQueueService,
	invoices *invoiceservice.Service,
) services.AgentTool {
	return newApproveBillingQueueItemTool(billing, invoices)
}

func provideSendBackToOpsTool(billing services.BillingQueueService) services.AgentTool {
	return newSendBackToOpsTool(billing)
}

func provideMoveToExceptionTool(billing services.BillingQueueService) services.AgentTool {
	return newMoveToExceptionTool(billing)
}

func provideHoldBillingQueueItemTool(billing services.BillingQueueService) services.AgentTool {
	return newHoldBillingQueueItemTool(billing)
}

func provideCancelBillingQueueItemTool(billing services.BillingQueueService) services.AgentTool {
	return newCancelBillingQueueItemTool(billing)
}

func provideAssignBillerTool(billing services.BillingQueueService) services.AgentTool {
	return newAssignBillerTool(billing)
}

func providePostInvoiceTool(invoices *invoiceservice.Service) services.AgentTool {
	return newPostInvoiceTool(invoices)
}

func provideSendInvoiceTool(invoices *invoiceservice.Service) services.AgentTool {
	return newSendInvoiceTool(invoices)
}

func provideApproveDetentionTool(detention *detentionservice.Service) services.AgentTool {
	return newApproveDetentionTool(detention)
}

func provideRequestCredentialRenewalTool(
	credentials *workercredentialservice.Service,
) services.AgentTool {
	return newRequestCredentialRenewalTool(credentials)
}

func providePlaceWorkerDispatchHoldTool(workers *workerservice.Service) services.AgentTool {
	return newPlaceWorkerDispatchHoldTool(workers)
}

func provideAcknowledgeCarrierIntelEventTool(
	intel *carrierintelservice.Service,
) services.AgentTool {
	return newAcknowledgeCarrierIntelEventTool(intel)
}

func provideResolveCarrierIntelEventTool(intel *carrierintelservice.Service) services.AgentTool {
	return newResolveCarrierIntelEventTool(intel)
}

// The tool takes a narrow interface so it can be tested without the whole
// document stack; fx holds the concrete service, so the widening happens here.
func provideAttachDocumentTool(documents *documentservice.Service) services.AgentTool {
	return newAttachDocumentTool(documents)
}

func provideLinkInboundMessageTool(inbox *inboundmessageservice.Service) services.AgentTool {
	return newLinkInboundMessageTool(inbox)
}

func provideMarkInboundMessageTool(inbox *inboundmessageservice.Service) services.AgentTool {
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

func provideAcceptEDILoadTenderTool(edi *ediservice.Service) services.AgentTool {
	return newAcceptEDILoadTenderTool(edi)
}

func provideDeclineEDILoadTenderTool(edi *ediservice.Service) services.AgentTool {
	return newDeclineEDILoadTenderTool(edi)
}

func provideCancelEDILoadTenderTool(edi *ediservice.Service) services.AgentTool {
	return newCancelEDILoadTenderTool(edi)
}

func provideExpireEDILoadTenderTool(edi *ediservice.Service) services.AgentTool {
	return newExpireEDILoadTenderTool(edi)
}

func provideReviewEDITenderChangeTool(edi *ediservice.Service) services.AgentTool {
	return newReviewEDITenderChangeTool(edi)
}

func provideReviewEDITransferChangeTool(edi *ediservice.Service) services.AgentTool {
	return newReviewEDITransferChangeTool(edi)
}

func provideRetryEDIMessageDeliveryTool(edi *ediservice.Service) services.AgentTool {
	return newRetryEDIMessageDeliveryTool(edi)
}

func provideReplayEDIMessageTool(edi *ediservice.Service) services.AgentTool {
	return newReplayEDIMessageTool(edi)
}

func provideReprocessEDIInboundFilesTool(inbound *ediinboundservice.Service) services.AgentTool {
	return newReprocessEDIInboundFilesTool(inbound)
}

func provideSendEDILoadTenderTool(edi *ediservice.Service) services.AgentTool {
	return newSendEDILoadTenderTool(edi)
}

func provideSendEDIStatusUpdateTool(edi *ediservice.Service) services.AgentTool {
	return newSendEDIStatusUpdateTool(edi)
}

func provideOpenInvoiceDisputeTool(disputes services.InvoiceDisputeService) services.AgentTool {
	return newOpenInvoiceDisputeTool(disputes)
}

func provideResolveInvoiceDisputeTool(disputes services.InvoiceDisputeService) services.AgentTool {
	return newResolveInvoiceDisputeTool(disputes)
}

func provideWithdrawInvoiceDisputeTool(disputes services.InvoiceDisputeService) services.AgentTool {
	return newWithdrawInvoiceDisputeTool(disputes)
}

func provideApplyCustomerPaymentTool(payments services.CustomerPaymentService) services.AgentTool {
	return newApplyCustomerPaymentTool(payments)
}

func provideReverseCustomerPaymentTool(
	payments services.CustomerPaymentService,
) services.AgentTool {
	return newReverseCustomerPaymentTool(payments)
}

func provideApplyCreditMemoTool(payments services.CustomerPaymentService) services.AgentTool {
	return newApplyCreditMemoTool(payments)
}

func provideUnapplyCreditMemoTool(payments services.CustomerPaymentService) services.AgentTool {
	return newUnapplyCreditMemoTool(payments)
}

func provideSaveInvoiceAdjustmentDraftTool(
	adjustments services.InvoiceAdjustmentService,
) services.AgentTool {
	return newSaveInvoiceAdjustmentDraftTool(adjustments)
}

func provideSubmitInvoiceAdjustmentTool(
	adjustments services.InvoiceAdjustmentService,
) services.AgentTool {
	return newSubmitInvoiceAdjustmentTool(adjustments)
}

func provideApproveInvoiceAdjustmentTool(
	adjustments services.InvoiceAdjustmentService,
) services.AgentTool {
	return newApproveInvoiceAdjustmentTool(adjustments)
}

func provideRejectInvoiceAdjustmentTool(
	adjustments services.InvoiceAdjustmentService,
) services.AgentTool {
	return newRejectInvoiceAdjustmentTool(adjustments)
}

func provideBuildInvoiceRunTool(runs *invoicerunservice.Service) services.AgentTool {
	return newBuildInvoiceRunTool(runs)
}

func provideAdjustInvoiceRunMembershipTool(runs *invoicerunservice.Service) services.AgentTool {
	return newAdjustInvoiceRunMembershipTool(runs)
}

func provideCommitInvoiceRunTool(runs *invoicerunservice.Service) services.AgentTool {
	return newCommitInvoiceRunTool(runs)
}

func provideCancelInvoiceRunTool(runs *invoicerunservice.Service) services.AgentTool {
	return newCancelInvoiceRunTool(runs)
}

func provideBillStatementNowTool(runs *invoicerunservice.Service) services.AgentTool {
	return newBillStatementNowTool(runs)
}

func provideAssessLateChargesTool(charges services.LateChargeService) services.AgentTool {
	return newAssessLateChargesTool(charges)
}

func provideShareInvoiceTool(shares *invoiceshareservice.Service) services.AgentTool {
	return newShareInvoiceTool(shares)
}

func provideUpdateInvoiceDraftTool(invoices *invoiceservice.Service) services.AgentTool {
	return newUpdateInvoiceDraftTool(invoices)
}

func provideGenerateInvoicePDFTool(invoices *invoiceservice.Service) services.AgentTool {
	return newGenerateInvoicePDFTool(invoices)
}

func provideCreateInvoiceTool(invoices *invoiceservice.Service) services.AgentTool {
	return newCreateInvoiceTool(invoices)
}

func provideCreateInvoiceMemoTool(invoices *invoiceservice.Service) services.AgentTool {
	return newCreateInvoiceMemoTool(invoices)
}

func provideVoidInvoiceTool(invoices *invoiceservice.Service) services.AgentTool {
	return newVoidInvoiceTool(invoices)
}

func provideSendInvoiceEDITool(invoices *invoiceservice.Service) services.AgentTool {
	return newSendInvoiceEDITool(invoices)
}
