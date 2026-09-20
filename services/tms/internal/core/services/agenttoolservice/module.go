package agenttoolservice

import "go.uber.org/fx"

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
		NewRegistry,
	),
)
