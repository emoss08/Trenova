package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("agent-tool-service",
	fx.Provide(
		fx.Annotate(newTransitionToInReviewTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newCorrectChargeCodeTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newRequestMissingDocsTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newAttachDocumentTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newFlagManualReviewTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newAssignMoveTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newRaiseExceptionTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newAddShipmentCommentTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newPlaceShipmentHoldTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newReleaseShipmentHoldTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newCancelShipmentTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newRecordStopActualTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideUpdateTractorStatusTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideUpdateTrailerStatusTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideApproveWorkerPTOTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideRejectWorkerPTOTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideCancelWorkerPTOTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideCreateReportTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideUpdateReportTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideForkReportTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideEvaluateServiceFailuresTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideResolveServiceFailureTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideNotifyDriverTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newEmailCustomerTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideSendDetentionNoticeTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideWaiveDetentionTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideCreateShipmentTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideUpdateShipmentTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideTenderToRoutingGuideTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideTenderToCarriersTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newRememberTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(newForgetMemoryTool, fx.ResultTags(`group:"agent_tools"`)),
		fx.Annotate(provideDismissInsightTool, fx.ResultTags(`group:"agent_tools"`)),
		NewRegistry,
	),
)

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

func provideUpdateShipmentTool(shipments services.ShipmentService) services.AgentTool {
	return newUpdateShipmentTool(shipments)
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
