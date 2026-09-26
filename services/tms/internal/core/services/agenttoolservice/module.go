package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptservice"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
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
	return []any{
		newTransitionToInReviewTool,
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
