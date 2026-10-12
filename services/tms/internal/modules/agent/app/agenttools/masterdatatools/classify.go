package masterdatatools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func classifyTableView(params agentports.ToolExecuteParams) agentports.CallPolicy {
	actor := params.Actor
	if common.OptionalBool(params.Params, "shared") || actor == nil ||
		actor.PrincipalType != permissionports.PrincipalTypeUser || actor.UserID.IsNil() {
		return agentports.CallPolicy{Egress: agent.EgressInternal}
	}

	return agentports.CallPolicy{Egress: agent.EgressPersonal}
}
