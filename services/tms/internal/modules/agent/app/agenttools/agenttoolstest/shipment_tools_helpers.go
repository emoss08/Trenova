package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

func ExecuteParams(params map[string]any) agentports.ToolExecuteParams {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	return agentports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &permissionports.RequestActor{
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		Params: params,
	}
}
