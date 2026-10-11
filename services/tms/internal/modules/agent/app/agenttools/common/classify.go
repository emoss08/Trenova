package common

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

func ClassifyCommentVisibility(params agentports.ToolExecuteParams) agentports.CallPolicy {
	switch shipment.CommentVisibility(OptionalString(params.Params, "visibility")) {
	case shipment.CommentVisibilityCustomer:
		return agentports.CallPolicy{Egress: agent.EgressCustomerVisible}
	case shipment.CommentVisibilityDriver:
		return agentports.CallPolicy{Egress: agent.EgressDriverVisible}
	case shipment.CommentVisibilityInternal,
		shipment.CommentVisibilityOperations,
		shipment.CommentVisibilityAccounting:
		return agentports.CallPolicy{Egress: agent.EgressInternal}
	default:
		return agentports.CallPolicy{Egress: agent.EgressInternal}
	}
}
