package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/alerttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/bankreceipttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/dashboardtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/dispatchtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/documenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/edispec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/equipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/exceptiontools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/factoringtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/findingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/inboundtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/insighttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoiceadjustmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicedisputetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicelifecycletools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoiceruntools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/locationtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/memorytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/movetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/portaltools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/qualificationtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reporttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/scheduletools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/triagequeuetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/triageworktools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/webwatchtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
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
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentimportassistantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationcategoryports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
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
		provideTransitionItemsToInReviewTool,
		provideAssignBillersTool,
		provideTransferToBillingTool,
		provideSendBackToOpsTool,
		provideMoveToExceptionTool,
		provideHoldBillingQueueItemTool,
		provideCancelBillingQueueItemTool,
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
		provideCorrectChargeCodeTool,
		dashboardtools.NewSaveTableViewTool,
		dashboardtools.NewCreateDashboardTool,
		dashboardtools.NewAddDashboardTileTool,
		alerttools.NewScheduleReportTool,
		alerttools.NewCreateTableChangeAlertTool,
		qualificationtools.NewRequestMissingDocsTool,
		provideAttachDocumentTool,
		movetools.NewAssignMoveTool,
		exceptiontools.NewRaiseExceptionTool,
		carriertools.NewWaitUntilTool,
		carriertools.NewCancelWaitTool,
		scheduletools.NewCreateScheduleTool,
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
		portaltools.NewActOnPortalTool,
		webwatchtools.NewWatchWebTool,
		webwatchtools.NewStopWebWatchTool,
		provideLinkInboundMessageTool,
		provideMarkInboundMessageTool,
		inboundtools.NewReplyToInboundMessageTool,
		triagequeuetools.NewAssignInboundMessagesTool,
		triagequeuetools.NewReopenInboundMessagesTool,
		triagequeuetools.NewSnoozeInboundMessagesTool,
		triagequeuetools.NewWakeInboundMessageTool,
		triagequeuetools.NewWaitForInboundReplyTool,
		triagequeuetools.NewUnlinkInboundMessageTool,
		triagequeuetools.NewHandOffTriageTool,
		triagequeuetools.NewBlockInboundSenderTool,
		triagequeuetools.NewUnblockInboundSenderTool,
		triageworktools.NewDraftInboundReplyTool,
		triageworktools.NewDiscardInboundReplyDraftTool,
		triageworktools.NewDraftInboundLoadTool,
		triageworktools.NewCreateInboundLoadTool,
		triageworktools.NewDiscardInboundLoadTool,
		triageworktools.NewFileInboundAttachmentTool,
		triageworktools.NewUnfileInboundAttachmentTool,
		triageworktools.NewLogInboundExceptionTool,
		triageworktools.NewSettleDetentionDisputeTool,
		triageworktools.NewResolveInboundConflictTool,
		triageworktools.NewReopenInboundConflictTool,
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
		memorytools.NewRememberTool,
		memorytools.NewGuideProfileTool,
		memorytools.NewForgetMemoryTool,
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
	providers = append(providers, followUpToolProviders()...)

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

func provideUpdateTractorStatusTool(tractors *tractorservice.Service) agenttoolports.AgentTool {
	return equipmenttools.NewUpdateTractorStatusTool(tractors)
}

func provideUpdateTrailerStatusTool(trailers *trailerservice.Service) agenttoolports.AgentTool {
	return equipmenttools.NewUpdateTrailerStatusTool(trailers)
}

func provideApproveWorkerPTOTool(pto workerptoports.WorkerPTOService) agenttoolports.AgentTool {
	return operationstools.NewApproveWorkerPTOTool(pto)
}

func provideRejectWorkerPTOTool(pto workerptoports.WorkerPTOService) agenttoolports.AgentTool {
	return operationstools.NewRejectWorkerPTOTool(pto)
}

func provideCancelWorkerPTOTool(pto workerptoports.WorkerPTOService) agenttoolports.AgentTool {
	return operationstools.NewCancelWorkerPTOTool(pto)
}

func provideCreateReportTool(reports *reporting.Service) agenttoolports.AgentTool {
	return reporttools.NewCreateReportTool(reports)
}

func provideUpdateReportTool(reports *reporting.Service) agenttoolports.AgentTool {
	return reporttools.NewUpdateReportTool(reports)
}

func provideForkReportTool(reports *reporting.Service) agenttoolports.AgentTool {
	return reporttools.NewForkReportTool(reports)
}

func provideEvaluateServiceFailuresTool(
	failures servicefailureports.ServiceFailureService,
) agenttoolports.AgentTool {
	return operationstools.NewEvaluateServiceFailuresTool(failures)
}

func provideResolveServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agenttoolports.AgentTool {
	return operationstools.NewResolveServiceFailureTool(failures)
}

func provideNotifyDriverTool(drivers *drivernotificationservice.Service) agenttoolports.AgentTool {
	return operationstools.NewNotifyDriverTool(drivers)
}

func provideSendDetentionNoticeTool(detention *detentionservice.Service) agenttoolports.AgentTool {
	return operationstools.NewSendDetentionNoticeTool(detention)
}

func provideWaiveDetentionTool(detention *detentionservice.Service) agenttoolports.AgentTool {
	return operationstools.NewWaiveDetentionTool(detention)
}

type intakeToolParams struct {
	fx.In

	Logger    *zap.Logger
	Shipments shipmentserviceports.ShipmentService
	Locations locationports.LocationRepository
	Imports   shipmentimportassistantports.ShipmentImportAssistantService `optional:"true"`
}

func provideCreateShipmentTool(p intakeToolParams) agenttoolports.AgentTool {
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
) agenttoolports.AgentTool {
	return locationtools.NewCreateLocationTool(locationtools.CreateLocationToolParams{
		Locations:  locations,
		States:     states,
		Categories: categories,
		Places:     places,
	})
}

func provideUpdateShipmentTool(
	shipments shipmentserviceports.ShipmentService,
	partners *ediservice.Service,
) agenttoolports.AgentTool {
	return operationstools.NewUpdateShipmentTool(shipments, partners)
}

func provideRescheduleStopTool(
	shipments shipmentserviceports.ShipmentService,
	partners *ediservice.Service,
) agenttoolports.AgentTool {
	return shipmenttools.NewRescheduleStopTool(shipments, partners)
}

func provideAddShipmentChargeTool(
	shipments shipmentserviceports.ShipmentService,
	accessorials accessorialchargeports.AccessorialChargeRepository,
) agenttoolports.AgentTool {
	return shipmenttools.NewAddShipmentChargeTool(shipments, accessorials)
}

func provideCancelShipmentTool(
	shipments shipmentserviceports.ShipmentService,
	partners *ediservice.Service,
	tenders *tenderservice.Service,
) agenttoolports.AgentTool {
	return shipmenttools.NewCancelShipmentTool(shipments, partners, tenders)
}

func provideTenderToRoutingGuideTool(tenders *tenderservice.Service) agenttoolports.AgentTool {
	return tendertools.NewTenderToRoutingGuideTool(tenders)
}

func provideTenderToCarriersTool(tenders *tenderservice.Service) agenttoolports.AgentTool {
	return tendertools.NewTenderToCarriersTool(tenders)
}

func provideDismissInsightTool(insights *insightservice.Service) agenttoolports.AgentTool {
	return insighttools.NewDismissInsightTool(insights)
}

func provideMatchBankReceiptTool(receipts *bankreceiptservice.Service) agenttoolports.AgentTool {
	return bankreceipttools.NewMatchBankReceiptTool(receipts)
}

func providePostCustomerPaymentTool(
	payments customerpaymentports.CustomerPaymentService,
	receipts *bankreceiptservice.Service,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentTool {
	return bankreceipttools.NewPostCustomerPaymentTool(payments, receipts, permissions)
}

func provideResolveBankReceiptWorkItemTool(
	items *bankreceiptworkitemservice.Service,
) agenttoolports.AgentTool {
	return bankreceipttools.NewResolveBankReceiptWorkItemTool(items)
}

func provideEscalateDetentionTool(detention *detentionservice.Service) agenttoolports.AgentTool {
	return findingtools.NewEscalateDetentionTool(detention)
}

func provideTransferToBillingTool(
	shipments shipmentserviceports.ShipmentService,
	runs *billingtransferservice.Service,
) agenttoolports.AgentTool {
	return billingtools.NewTransferToBillingTool(shipments, runs)
}

func provideSendBackToOpsTool(
	billing billingqueueports.BillingQueueService,
) agenttoolports.AgentTool {
	return billingtools.NewSendBackToOpsTool(billing)
}

func provideMoveToExceptionTool(
	billing billingqueueports.BillingQueueService,
) agenttoolports.AgentTool {
	return billingtools.NewMoveToExceptionTool(billing)
}

func provideHoldBillingQueueItemTool(
	billing billingqueueports.BillingQueueService,
) agenttoolports.AgentTool {
	return billingtools.NewHoldBillingQueueItemTool(billing)
}

func provideCancelBillingQueueItemTool(
	billing billingqueueports.BillingQueueService,
) agenttoolports.AgentTool {
	return billingtools.NewCancelBillingQueueItemTool(billing)
}

func provideAssignBillersTool(
	billing billingqueueports.BillingQueueService,
) agenttoolports.AgentTool {
	return billingtools.NewAssignBillersTool(billing)
}

func provideTransitionItemsToInReviewTool(
	billing billingqueueports.BillingQueueService,
) agenttoolports.AgentTool {
	return billingtools.NewTransitionItemsToInReviewTool(billing)
}

func providePostInvoicesTool(
	invoices *invoiceservice.Service,
	numbers invoiceports.InvoiceRepository,
) agenttoolports.AgentTool {
	return invoicetools.NewPostInvoicesTool(invoices, numbers)
}

func provideSendInvoicesTool(
	invoices *invoiceservice.Service,
	numbers invoiceports.InvoiceRepository,
) agenttoolports.AgentTool {
	return invoicetools.NewSendInvoicesTool(invoices, numbers)
}

func provideApproveBillingQueueItemsTool(
	billing billingqueueports.BillingQueueService,
	invoices *invoiceservice.Service,
) agenttoolports.AgentTool {
	return invoicetools.NewApproveBillingQueueItemsTool(billing, invoices)
}

func provideApproveDetentionTool(detention *detentionservice.Service) agenttoolports.AgentTool {
	return findingtools.NewApproveDetentionTool(detention)
}

func provideRequestCredentialRenewalTool(
	credentials *workercredentialservice.Service,
) agenttoolports.AgentTool {
	return findingtools.NewRequestCredentialRenewalTool(credentials)
}

func providePlaceWorkerDispatchHoldTool(workers *workerservice.Service) agenttoolports.AgentTool {
	return findingtools.NewPlaceWorkerDispatchHoldTool(workers)
}

func provideAcknowledgeCarrierIntelEventTool(
	intel *carrierintelservice.Service,
) agenttoolports.AgentTool {
	return findingtools.NewAcknowledgeCarrierIntelEventTool(intel)
}

func provideResolveCarrierIntelEventTool(
	intel *carrierintelservice.Service,
) agenttoolports.AgentTool {
	return findingtools.NewResolveCarrierIntelEventTool(intel)
}

// The tool takes a narrow interface so it can be tested without the whole
// document stack; fx holds the concrete service, so the widening happens here.
func provideAttachDocumentTool(documents *documentservice.Service) agenttoolports.AgentTool {
	return documenttools.NewAttachDocumentTool(documents)
}

func provideLinkInboundMessageTool(inbox *inboundmessageservice.Service) agenttoolports.AgentTool {
	return inboundtools.NewLinkInboundMessageTool(inbox)
}

func provideMarkInboundMessageTool(inbox *inboundmessageservice.Service) agenttoolports.AgentTool {
	return inboundtools.NewMarkInboundMessageTool(inbox)
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
		provideApplyEDIConnectionDraftTool,
	}
}

func provideAcceptEDILoadTenderTool(edi *ediservice.Service) agenttoolports.AgentTool {
	return editools.NewAcceptEDILoadTenderTool(edi)
}

func provideDeclineEDILoadTenderTool(edi *ediservice.Service) agenttoolports.AgentTool {
	return editools.NewDeclineEDILoadTenderTool(edi)
}

func provideCancelEDILoadTenderTool(edi *ediservice.Service) agenttoolports.AgentTool {
	return editools.NewCancelEDILoadTenderTool(edi)
}

func provideExpireEDILoadTenderTool(edi *ediservice.Service) agenttoolports.AgentTool {
	return editools.NewExpireEDILoadTenderTool(edi)
}

func provideReviewEDITenderChangeTool(edi *ediservice.Service) agenttoolports.AgentTool {
	return editools.NewReviewEDITenderChangeTool(edi)
}

func provideReviewEDITransferChangeTool(edi *ediservice.Service) agenttoolports.AgentTool {
	return editools.NewReviewEDITransferChangeTool(edi)
}

func provideRetryEDIMessageDeliveryTool(edi *ediservice.Service) agenttoolports.AgentTool {
	return editools.NewRetryEDIMessageDeliveryTool(edi)
}

func provideReplayEDIMessageTool(edi *ediservice.Service) agenttoolports.AgentTool {
	return editools.NewReplayEDIMessageTool(edi)
}

func provideReprocessEDIInboundFilesTool(
	inbound *ediinboundservice.Service,
) agenttoolports.AgentTool {
	return editools.NewReprocessEDIInboundFilesTool(inbound)
}

func provideSendEDILoadTenderTool(edi *ediservice.Service) agenttoolports.AgentTool {
	return editools.NewSendEDILoadTenderTool(edi)
}

func provideSendEDIStatusUpdateTool(edi *ediservice.Service) agenttoolports.AgentTool {
	return editools.NewSendEDIStatusUpdateTool(edi)
}

func provideApplyEDIConnectionDraftTool(
	drafts ediconnectiondraftports.EDIConnectionDraftService,
	documents documentports.DocumentRepository,
	contents documentports.DocumentContentService,
) agenttoolports.AgentTool {
	return editools.NewApplyConnectionDraftTool(
		drafts,
		edispec.NewDocumentReader(documents, contents),
	)
}

func provideOpenInvoiceDisputeTool(
	disputes invoiceports.InvoiceDisputeService,
) agenttoolports.AgentTool {
	return invoicedisputetools.NewOpenInvoiceDisputeTool(disputes)
}

func provideResolveInvoiceDisputeTool(
	disputes invoiceports.InvoiceDisputeService,
) agenttoolports.AgentTool {
	return invoicedisputetools.NewResolveInvoiceDisputeTool(disputes)
}

func provideWithdrawInvoiceDisputeTool(
	disputes invoiceports.InvoiceDisputeService,
) agenttoolports.AgentTool {
	return invoicedisputetools.NewWithdrawInvoiceDisputeTool(disputes)
}

func provideApplyCustomerPaymentTool(
	payments customerpaymentports.CustomerPaymentService,
) agenttoolports.AgentTool {
	return receivabletools.NewApplyCustomerPaymentTool(payments)
}

func provideReverseCustomerPaymentTool(
	payments customerpaymentports.CustomerPaymentService,
) agenttoolports.AgentTool {
	return receivabletools.NewReverseCustomerPaymentTool(payments)
}

func provideApplyCreditMemoTool(
	payments customerpaymentports.CustomerPaymentService,
) agenttoolports.AgentTool {
	return receivabletools.NewApplyCreditMemoTool(payments)
}

func provideUnapplyCreditMemoTool(
	payments customerpaymentports.CustomerPaymentService,
) agenttoolports.AgentTool {
	return receivabletools.NewUnapplyCreditMemoTool(payments)
}

func provideSaveInvoiceAdjustmentDraftTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agenttoolports.AgentTool {
	return invoiceadjustmenttools.NewSaveInvoiceAdjustmentDraftTool(adjustments)
}

func provideSubmitInvoiceAdjustmentTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agenttoolports.AgentTool {
	return invoiceadjustmenttools.NewSubmitInvoiceAdjustmentTool(adjustments)
}

func provideApproveInvoiceAdjustmentTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agenttoolports.AgentTool {
	return invoiceadjustmenttools.NewApproveInvoiceAdjustmentTool(adjustments)
}

func provideRejectInvoiceAdjustmentTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
) agenttoolports.AgentTool {
	return invoiceadjustmenttools.NewRejectInvoiceAdjustmentTool(adjustments)
}

func provideBuildInvoiceRunTool(runs *invoicerunservice.Service) agenttoolports.AgentTool {
	return invoiceruntools.NewBuildInvoiceRunTool(runs)
}

func provideAdjustInvoiceRunMembershipTool(
	runs *invoicerunservice.Service,
) agenttoolports.AgentTool {
	return invoiceruntools.NewAdjustInvoiceRunMembershipTool(runs)
}

func provideCommitInvoiceRunTool(runs *invoicerunservice.Service) agenttoolports.AgentTool {
	return invoiceruntools.NewCommitInvoiceRunTool(runs)
}

func provideCancelInvoiceRunTool(runs *invoicerunservice.Service) agenttoolports.AgentTool {
	return invoiceruntools.NewCancelInvoiceRunTool(runs)
}

func provideBillStatementNowTool(runs *invoicerunservice.Service) agenttoolports.AgentTool {
	return invoiceruntools.NewBillStatementNowTool(runs)
}

func provideAssessLateChargesTool(
	charges latechargeports.LateChargeService,
) agenttoolports.AgentTool {
	return receivabletools.NewAssessLateChargesTool(charges)
}

func provideShareInvoiceTool(shares *invoiceshareservice.Service) agenttoolports.AgentTool {
	return receivabletools.NewShareInvoiceTool(shares)
}

func provideUpdateInvoiceDraftTool(invoices *invoiceservice.Service) agenttoolports.AgentTool {
	return invoicelifecycletools.NewUpdateInvoiceDraftTool(invoices)
}

func provideGenerateInvoicePDFTool(invoices *invoiceservice.Service) agenttoolports.AgentTool {
	return invoicelifecycletools.NewGenerateInvoicePDFTool(invoices)
}

func provideCreateInvoiceTool(invoices *invoiceservice.Service) agenttoolports.AgentTool {
	return invoicelifecycletools.NewCreateInvoiceTool(invoices)
}

func provideCreateInvoiceMemoTool(invoices *invoiceservice.Service) agenttoolports.AgentTool {
	return invoicelifecycletools.NewCreateInvoiceMemoTool(invoices)
}

func provideVoidInvoiceTool(invoices *invoiceservice.Service) agenttoolports.AgentTool {
	return invoicelifecycletools.NewVoidInvoiceTool(invoices)
}

func provideSendInvoiceEDITool(invoices *invoiceservice.Service) agenttoolports.AgentTool {
	return invoicelifecycletools.NewSendInvoiceEDITool(invoices)
}
