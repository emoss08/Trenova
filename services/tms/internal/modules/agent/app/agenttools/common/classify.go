package common

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

func ClassifyCommentVisibility(params agenttoolports.ToolExecuteParams) agenttoolports.CallPolicy {
	switch shipment.CommentVisibility(OptionalString(params.Params, "visibility")) {
	case shipment.CommentVisibilityCustomer:
		return agenttoolports.CallPolicy{Egress: agent.EgressCustomerVisible}
	case shipment.CommentVisibilityDriver:
		return agenttoolports.CallPolicy{Egress: agent.EgressDriverVisible}
	case shipment.CommentVisibilityInternal,
		shipment.CommentVisibilityOperations,
		shipment.CommentVisibilityAccounting:
		return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
	default:
		return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
	}
}
