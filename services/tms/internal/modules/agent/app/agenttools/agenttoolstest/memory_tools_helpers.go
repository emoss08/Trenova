package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

func MemoryParams(params map[string]any) agentports.ToolExecuteParams {
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")

	return agentports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		RunID:  pulid.MustNew("arun_"),
		Params: params,
	}
}
