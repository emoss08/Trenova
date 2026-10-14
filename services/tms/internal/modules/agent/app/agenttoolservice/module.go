package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/alerttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/dispatchtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/factoringtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/insighttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/movetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reporttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicerunservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceshareservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
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
		invoicetools.NewCorrectChargeCodeTool,
		masterdatatools.NewSaveTableViewTool,
		masterdatatools.NewCreateDashboardTool,
		masterdatatools.NewAddDashboardTileTool,
		alerttools.NewScheduleReportTool,
		alerttools.NewCreateTableChangeAlertTool,
		workforcetools.NewRequestMissingDocsTool,
		provideAttachDocumentTool,
		receivabletools.NewFlagManualReviewTool,
		masterdatatools.NewAssignMoveTool,
		receivabletools.NewRaiseExceptionTool,
		carriertools.NewWaitUntilTool,
		carriertools.NewCancelWaitTool,
		shipmenttools.NewAddShipmentCommentTool,
		shipmenttools.NewPlaceShipmentHoldTool,
		shipmenttools.NewReleaseShipmentHoldTool,
		shipmenttools.NewShareCustomerTrackingLinkTool,
		shipmenttools.NewRevokeCustomerTrackingLinkTool,
		provideCancelShipmentTool,
		movetools.ProvideRecordStopActualTool,
		movetools.ProvideLogCheckCallTool,
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
		operationstools.NewEmailCustomerTool,
		provideLinkInboundMessageTool,
		provideMarkInboundMessageTool,
		workforcetools.NewReplyToInboundMessageTool,
		provideSendDetentionNoticeTool,
		provideWaiveDetentionTool,
		provideCreateShipmentTool,
		provideUpdateShipmentTool,
		provideRescheduleStopTool,
		provideAddShipmentChargeTool,
		provideCreateLocationTool,
		provideTenderToRoutingGuideTool,
		provideTenderToCarriersTool,
		dispatchtools.NewUnassignMovesTool,
		dispatchtools.NewUpdateMoveStatusTool,
		dispatchtools.ProvideAssignMoveToCarrierTool,
		dispatchtools.ProvideCancelCarrierAssignmentTool,
		dispatchtools.ProvideCancelTenderTool,
		dispatchtools.ProvideRecordTenderResponseTool,
		dispatchtools.ProvideGenerateRateConfirmationTool,
		dispatchtools.ProvideSendRateConfirmationTool,
		dispatchtools.ProvideVoidRateConfirmationTool,
		dispatchtools.ProvideRecordRateConfirmationConfirmedTool,
		masterdatatools.NewRememberTool,
		masterdatatools.NewForgetMemoryTool,
		provideDismissInsightTool,
		accountingtools.ProvideCheckAccountingConnectionTool,
		accountingtools.ProvideSetAccountingMappingTool,
		accountingtools.ProvideClearAccountingMappingTool,
		accountingtools.ProvideCreateAccountingReferenceRecordTool,
		accountingtools.ProvideRefreshAccountingReferenceDataTool,
		accountingtools.ProvideRetryAccountingSyncTool,
		accountingtools.ProvideSkipAccountingSyncTool,
		accountingtools.ProvideRedateAccountingSyncTool,
		accountingtools.ProvideApplyAccountingInboundChangeTool,
		accountingtools.ProvideIgnoreAccountingInboundChangeTool,
		accountingtools.ProvideResolveAccountingDriftTool,
		accountingtools.ProvideDismissAccountingDriftTool,
		accountingtools.ProvideCheckAccountingDriftTool,
		accountingtools.ProvidePauseAccountingSyncTool,
		accountingtools.ProvideResumeAccountingSyncTool,
		accountingtools.ProvideRequestAccountingBackfillTool,
		provideMatchBankReceiptTool,
		providePostCustomerPaymentTool,
		provideResolveBankReceiptWorkItemTool,
		provideEscalateDetentionTool,
		provideApproveDetentionTool,
		provideRequestCredentialRenewalTool,
		providePlaceWorkerDispatchHoldTool,
		provideAcknowledgeCarrierIntelEventTool,
		provideResolveCarrierIntelEventTool,
		alerttools.NewAddHomeWidgetTool,
		alerttools.NewRemoveHomeWidgetTool,
		alerttools.NewArrangeHomeLayoutTool,
	}

	providers = append(providers, ediToolProviders()...)
	providers = append(providers, ledgerToolProviders()...)
	providers = append(providers, operationsToolProviders()...)
	providers = append(providers, ratesToolProviders()...)
	providers = append(providers, workforceToolProviders()...)
	providers = append(providers, masterDataToolProviders()...)

	providers = append(providers, factoringtools.Providers()...)

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

func provideUpdateTractorStatusTool(tractors *tractorservice.Service) agentports.AgentTool {
	return masterdatatools.NewUpdateTractorStatusTool(tractors)
}

func provideUpdateTrailerStatusTool(trailers *trailerservice.Service) agentports.AgentTool {
	return masterdatatools.NewUpdateTrailerStatusTool(trailers)
}

func provideApproveWorkerPTOTool(pto workerports.WorkerPTOService) agentports.AgentTool {
	return operationstools.NewApproveWorkerPTOTool(pto)
}

func provideRejectWorkerPTOTool(pto workerports.WorkerPTOService) agentports.AgentTool {
	return operationstools.NewRejectWorkerPTOTool(pto)
}

func provideCancelWorkerPTOTool(pto workerports.WorkerPTOService) agentports.AgentTool {
	return operationstools.NewCancelWorkerPTOTool(pto)
}

func provideCreateReportTool(reports *reporting.Service) agentports.AgentTool {
	return reporttools.NewCreateReportTool(reports)
}

func provideUpdateReportTool(reports *reporting.Service) agentports.AgentTool {
	return reporttools.NewUpdateReportTool(reports)
}

func provideForkReportTool(reports *reporting.Service) agentports.AgentTool {
	return reporttools.NewForkReportTool(reports)
}

func provideEvaluateServiceFailuresTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentTool {
	return operationstools.NewEvaluateServiceFailuresTool(failures)
}

func provideResolveServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentTool {
	return operationstools.NewResolveServiceFailureTool(failures)
}

func provideNotifyDriverTool(drivers *drivernotificationservice.Service) agentports.AgentTool {
	return operationstools.NewNotifyDriverTool(drivers)
}

func provideSendDetentionNoticeTool(detention *detentionservice.Service) agentports.AgentTool {
	return operationstools.NewSendDetentionNoticeTool(detention)
}

func provideWaiveDetentionTool(detention *detentionservice.Service) agentports.AgentTool {
	return operationstools.NewWaiveDetentionTool(detention)
}

type intakeToolParams struct {
	fx.In

	Logger    *zap.Logger
	Shipments shipmentports.ShipmentService
	Locations locationports.LocationRepository
	Imports   shipmentports.ShipmentImportAssistantService `optional:"true"`
}

func provideCreateShipmentTool(p intakeToolParams) agentports.AgentTool {
	deps := operationstools.CreateShipmentDeps{
		Shipments: p.Shipments,
		Contracts: p.Shipments,
		Locations: p.Locations,
		Logger:    p.Logger,
	}
	if p.Imports != nil {
		deps.Imports = p.Imports
	}

	return operationstools.NewCreateShipmentTool(deps)
}

func provideCreateLocationTool(
	locations *locationservice.Service,
	states usstateports.UsStateRepository,
	categories locationcategoryports.LocationCategoryRepository,
	places kernelports.AutoCompleteService,
) agentports.AgentTool {
	return masterdatatools.NewCreateLocationTool(masterdatatools.CreateLocationToolParams{
		Locations:  locations,
		States:     states,
		Categories: categories,
		Places:     places,
	})
}

func provideUpdateShipmentTool(
	shipments shipmentports.ShipmentService,
	partners *ediservice.Service,
) agentports.AgentTool {
	return operationstools.NewUpdateShipmentTool(shipments, partners)
}

func provideRescheduleStopTool(
	shipments shipmentports.ShipmentService,
	partners *ediservice.Service,
) agentports.AgentTool {
	return shipmenttools.NewRescheduleStopTool(shipments, partners)
}

func provideAddShipmentChargeTool(
	shipments shipmentports.ShipmentService,
	accessorials accessorialchargeports.AccessorialChargeRepository,
) agentports.AgentTool {
	return shipmenttools.NewAddShipmentChargeTool(shipments, accessorials)
}

func provideCancelShipmentTool(
	shipments shipmentports.ShipmentService,
	partners *ediservice.Service,
	tenders *tenderservice.Service,
) agentports.AgentTool {
	return shipmenttools.NewCancelShipmentTool(shipments, partners, tenders)
}

func provideTenderToRoutingGuideTool(tenders *tenderservice.Service) agentports.AgentTool {
	return tendertools.NewTenderToRoutingGuideTool(tenders)
}

func provideTenderToCarriersTool(tenders *tenderservice.Service) agentports.AgentTool {
	return tendertools.NewTenderToCarriersTool(tenders)
}

func provideDismissInsightTool(insights *insightservice.Service) agentports.AgentTool {
	return insighttools.NewDismissInsightTool(insights)
}

func provideMatchBankReceiptTool(receipts *bankreceiptservice.Service) agentports.AgentTool {
	return invoicetools.NewMatchBankReceiptTool(receipts)
}

func providePostCustomerPaymentTool(
	payments customerpaymentports.CustomerPaymentService,
	receipts *bankreceiptservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentTool {
	return invoicetools.NewPostCustomerPaymentTool(payments, receipts, permissions)
}

func provideResolveBankReceiptWorkItemTool(
	items *bankreceiptworkitemservice.Service,
) agentports.AgentTool {
	return invoicetools.NewResolveBankReceiptWorkItemTool(items)
}

func provideEscalateDetentionTool(detention *detentionservice.Service) agentports.AgentTool {
	return masterdatatools.NewEscalateDetentionTool(detention)
}

func provideTransferToBillingTool(
	shipments shipmentports.ShipmentService,
	runs *billingtransferservice.Service,
) agentports.AgentTool {
	return billingtools.NewTransferToBillingTool(shipments, runs)
}

func provideApproveBillingQueueItemTool(
	billing billingqueueports.BillingQueueService,
	invoices *invoiceservice.Service,
) agentports.AgentTool {
	return billingtools.NewApproveBillingQueueItemTool(billing, invoices)
}

func provideSendBackToOpsTool(billing billingqueueports.BillingQueueService) agentports.AgentTool {
	return billingtools.NewSendBackToOpsTool(billing)
}

func provideMoveToExceptionTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return billingtools.NewMoveToExceptionTool(billing)
}

func provideHoldBillingQueueItemTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return billingtools.NewHoldBillingQueueItemTool(billing)
}

func provideCancelBillingQueueItemTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return billingtools.NewCancelBillingQueueItemTool(billing)
}

func provideAssignBillerTool(billing billingqueueports.BillingQueueService) agentports.AgentTool {
	return billingtools.NewAssignBillerTool(billing)
}

func provideAssignBillersTool(billing billingqueueports.BillingQueueService) agentports.AgentTool {
	return billingtools.NewAssignBillersTool(billing)
}

func provideTransitionToInReviewTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return billingtools.NewTransitionToInReviewTool(billing)
}

func provideTransitionItemsToInReviewTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return billingtools.NewTransitionItemsToInReviewTool(billing)
}

func providePostInvoiceTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return invoicetools.NewPostInvoiceTool(invoices)
}

func provideSendInvoiceTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return invoicetools.NewSendInvoiceTool(invoices)
}

func providePostInvoicesTool(
	invoices *invoiceservice.Service,
	numbers invoiceports.InvoiceRepository,
) agentports.AgentTool {
	return invoicetools.NewPostInvoicesTool(invoices, numbers)
}

func provideSendInvoicesTool(
	invoices *invoiceservice.Service,
	numbers invoiceports.InvoiceRepository,
) agentports.AgentTool {
	return invoicetools.NewSendInvoicesTool(invoices, numbers)
}

func provideApproveBillingQueueItemsTool(
	billing billingqueueports.BillingQueueService,
	invoices *invoiceservice.Service,
) agentports.AgentTool {
	return invoicetools.NewApproveBillingQueueItemsTool(billing, invoices)
}

func provideApproveDetentionTool(detention *detentionservice.Service) agentports.AgentTool {
	return masterdatatools.NewApproveDetentionTool(detention)
}

func provideRequestCredentialRenewalTool(
	credentials *workercredentialservice.Service,
) agentports.AgentTool {
	return masterdatatools.NewRequestCredentialRenewalTool(credentials)
}

func providePlaceWorkerDispatchHoldTool(workers *workerservice.Service) agentports.AgentTool {
	return masterdatatools.NewPlaceWorkerDispatchHoldTool(workers)
}

func provideAcknowledgeCarrierIntelEventTool(
	intel *carrierintelservice.Service,
) agentports.AgentTool {
	return masterdatatools.NewAcknowledgeCarrierIntelEventTool(intel)
}

func provideResolveCarrierIntelEventTool(intel *carrierintelservice.Service) agentports.AgentTool {
	return masterdatatools.NewResolveCarrierIntelEventTool(intel)
}

// The tool takes a narrow interface so it can be tested without the whole
// document stack; fx holds the concrete service, so the widening happens here.
func provideAttachDocumentTool(documents *documentservice.Service) agentports.AgentTool {
	return masterdatatools.NewAttachDocumentTool(documents)
}

func provideLinkInboundMessageTool(inbox *inboundmessageservice.Service) agentports.AgentTool {
	return workforcetools.NewLinkInboundMessageTool(inbox)
}

func provideMarkInboundMessageTool(inbox *inboundmessageservice.Service) agentports.AgentTool {
	return workforcetools.NewMarkInboundMessageTool(inbox)
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
	return editools.NewAcceptEDILoadTenderTool(edi)
}

func provideDeclineEDILoadTenderTool(edi *ediservice.Service) agentports.AgentTool {
	return editools.NewDeclineEDILoadTenderTool(edi)
}

func provideCancelEDILoadTenderTool(edi *ediservice.Service) agentports.AgentTool {
	return editools.NewCancelEDILoadTenderTool(edi)
}

func provideExpireEDILoadTenderTool(edi *ediservice.Service) agentports.AgentTool {
	return editools.NewExpireEDILoadTenderTool(edi)
}

func provideReviewEDITenderChangeTool(edi *ediservice.Service) agentports.AgentTool {
	return editools.NewReviewEDITenderChangeTool(edi)
}

func provideReviewEDITransferChangeTool(edi *ediservice.Service) agentports.AgentTool {
	return editools.NewReviewEDITransferChangeTool(edi)
}

func provideRetryEDIMessageDeliveryTool(edi *ediservice.Service) agentports.AgentTool {
	return editools.NewRetryEDIMessageDeliveryTool(edi)
}

func provideReplayEDIMessageTool(edi *ediservice.Service) agentports.AgentTool {
	return editools.NewReplayEDIMessageTool(edi)
}

func provideReprocessEDIInboundFilesTool(inbound *ediinboundservice.Service) agentports.AgentTool {
	return editools.NewReprocessEDIInboundFilesTool(inbound)
}

func provideSendEDILoadTenderTool(edi *ediservice.Service) agentports.AgentTool {
	return editools.NewSendEDILoadTenderTool(edi)
}

func provideSendEDIStatusUpdateTool(edi *ediservice.Service) agentports.AgentTool {
	return editools.NewSendEDIStatusUpdateTool(edi)
}

func provideOpenInvoiceDisputeTool(
	disputes invoiceports.InvoiceDisputeService,
) agentports.AgentTool {
	return invoicetools.NewOpenInvoiceDisputeTool(disputes)
}

func provideResolveInvoiceDisputeTool(
	disputes invoiceports.InvoiceDisputeService,
) agentports.AgentTool {
	return invoicetools.NewResolveInvoiceDisputeTool(disputes)
}

func provideWithdrawInvoiceDisputeTool(
	disputes invoiceports.InvoiceDisputeService,
) agentports.AgentTool {
	return invoicetools.NewWithdrawInvoiceDisputeTool(disputes)
}

func provideApplyCustomerPaymentTool(
	payments customerpaymentports.CustomerPaymentService,
) agentports.AgentTool {
	return receivabletools.NewApplyCustomerPaymentTool(payments)
}

func provideReverseCustomerPaymentTool(
	payments customerpaymentports.CustomerPaymentService,
) agentports.AgentTool {
	return receivabletools.NewReverseCustomerPaymentTool(payments)
}

func provideApplyCreditMemoTool(
	payments customerpaymentports.CustomerPaymentService,
) agentports.AgentTool {
	return receivabletools.NewApplyCreditMemoTool(payments)
}

func provideUnapplyCreditMemoTool(
	payments customerpaymentports.CustomerPaymentService,
) agentports.AgentTool {
	return receivabletools.NewUnapplyCreditMemoTool(payments)
}

func provideSaveInvoiceAdjustmentDraftTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agentports.AgentTool {
	return invoicetools.NewSaveInvoiceAdjustmentDraftTool(adjustments)
}

func provideSubmitInvoiceAdjustmentTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agentports.AgentTool {
	return invoicetools.NewSubmitInvoiceAdjustmentTool(adjustments)
}

func provideApproveInvoiceAdjustmentTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agentports.AgentTool {
	return invoicetools.NewApproveInvoiceAdjustmentTool(adjustments)
}

func provideRejectInvoiceAdjustmentTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agentports.AgentTool {
	return invoicetools.NewRejectInvoiceAdjustmentTool(adjustments)
}

func provideBuildInvoiceRunTool(runs *invoicerunservice.Service) agentports.AgentTool {
	return invoicetools.NewBuildInvoiceRunTool(runs)
}

func provideAdjustInvoiceRunMembershipTool(runs *invoicerunservice.Service) agentports.AgentTool {
	return invoicetools.NewAdjustInvoiceRunMembershipTool(runs)
}

func provideCommitInvoiceRunTool(runs *invoicerunservice.Service) agentports.AgentTool {
	return invoicetools.NewCommitInvoiceRunTool(runs)
}

func provideCancelInvoiceRunTool(runs *invoicerunservice.Service) agentports.AgentTool {
	return invoicetools.NewCancelInvoiceRunTool(runs)
}

func provideBillStatementNowTool(runs *invoicerunservice.Service) agentports.AgentTool {
	return invoicetools.NewBillStatementNowTool(runs)
}

func provideAssessLateChargesTool(charges latechargeports.LateChargeService) agentports.AgentTool {
	return receivabletools.NewAssessLateChargesTool(charges)
}

func provideShareInvoiceTool(shares *invoiceshareservice.Service) agentports.AgentTool {
	return receivabletools.NewShareInvoiceTool(shares)
}

func provideUpdateInvoiceDraftTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return invoicetools.NewUpdateInvoiceDraftTool(invoices)
}

func provideGenerateInvoicePDFTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return invoicetools.NewGenerateInvoicePDFTool(invoices)
}

func provideCreateInvoiceTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return invoicetools.NewCreateInvoiceTool(invoices)
}

func provideCreateInvoiceMemoTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return invoicetools.NewCreateInvoiceMemoTool(invoices)
}

func provideVoidInvoiceTool(invoices *invoiceservice.Service) agentports.AgentTool {
	return invoicetools.NewVoidInvoiceTool(invoices)
}

func provideSendInvoiceEDITool(invoices *invoiceservice.Service) agentports.AgentTool {
	return invoicetools.NewSendInvoiceEDITool(invoices)
}
