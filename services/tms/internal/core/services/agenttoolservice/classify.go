package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

func classifyCommentVisibility(params serviceports.ToolExecuteParams) serviceports.CallPolicy {
	switch shipment.CommentVisibility(optionalString(params.Params, "visibility")) {
	case shipment.CommentVisibilityCustomer:
		return serviceports.CallPolicy{Egress: agent.EgressCustomerVisible}
	case shipment.CommentVisibilityDriver:
		return serviceports.CallPolicy{Egress: agent.EgressDriverVisible}
	case shipment.CommentVisibilityInternal,
		shipment.CommentVisibilityOperations,
		shipment.CommentVisibilityAccounting:
		return serviceports.CallPolicy{Egress: agent.EgressInternal}
	default:
		return serviceports.CallPolicy{Egress: agent.EgressInternal}
	}
}

func classifyTableView(params serviceports.ToolExecuteParams) serviceports.CallPolicy {
	actor := params.Actor
	if optionalBool(params.Params, "shared") || actor == nil ||
		actor.PrincipalType != serviceports.PrincipalTypeUser || actor.UserID.IsNil() {
		return serviceports.CallPolicy{Egress: agent.EgressInternal}
	}

	return serviceports.CallPolicy{Egress: agent.EgressPersonal}
}
