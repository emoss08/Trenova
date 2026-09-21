package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"go.uber.org/fx"
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
