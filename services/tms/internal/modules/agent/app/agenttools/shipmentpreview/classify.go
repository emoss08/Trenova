package shipmentpreview

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

func ClassifyCommentVisibility(
	params *agenttoolports.ToolExecuteParams,
) agenttoolports.CallPolicy {
	switch shipmentvocab.CommentVisibility(toolparams.OptionalString(params.Params, "visibility")) {
	case shipmentvocab.CommentVisibilityCustomer:
		return agenttoolports.CallPolicy{Egress: agent.EgressCustomerVisible}
	case shipmentvocab.CommentVisibilityDriver:
		return agenttoolports.CallPolicy{Egress: agent.EgressDriverVisible}
	case shipmentvocab.CommentVisibilityInternal,
		shipmentvocab.CommentVisibilityOperations,
		shipmentvocab.CommentVisibilityAccounting:
		return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
	default:
		return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
	}
}
