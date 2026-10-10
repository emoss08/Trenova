package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func classifyCommentVisibility(params agentports.ToolExecuteParams) agentports.CallPolicy {
	switch shipment.CommentVisibility(optionalString(params.Params, "visibility")) {
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

func classifyTableView(params agentports.ToolExecuteParams) agentports.CallPolicy {
	actor := params.Actor
	if optionalBool(params.Params, "shared") || actor == nil ||
		actor.PrincipalType != permissionports.PrincipalTypeUser || actor.UserID.IsNil() {
		return agentports.CallPolicy{Egress: agent.EgressInternal}
	}

	return agentports.CallPolicy{Egress: agent.EgressPersonal}
}
