package dashboardtools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func classifyTableView(
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) agenttoolports.CallPolicy {
	actor := params.Actor
	if toolparams.OptionalBool(params.Params, "shared") || actor == nil ||
		actor.PrincipalType != permissionports.PrincipalTypeUser || actor.UserID.IsNil() {
		return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
	}

	return agenttoolports.CallPolicy{Egress: agent.EgressPersonal}
}
