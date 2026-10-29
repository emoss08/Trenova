package querytooltest

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func PersonParams(principal kernelports.PrincipalType) *agenttoolports.QueryToolParams {
	params := TestParams(map[string]any{})
	params.Actor.PrincipalType = principal

	return params
}

func AgentParams(
	params map[string]any,
	dataAccess permission.FieldSensitivity,
) *agenttoolports.QueryToolParams {
	out := TestParams(params)
	out.Actor.PrincipalType = permissionports.PrincipalTypeAgent
	out.Actor.PrincipalID = pulid.MustNew("agdef_")
	out.Actor.UserID = pulid.Nil
	out.DataAccessCeiling = dataAccess

	return out
}

func ChatParams(
	params map[string]any,
	dataAccess permission.FieldSensitivity,
) *agenttoolports.QueryToolParams {
	out := TestParams(params)
	out.Actor.PrincipalType = permissionports.PrincipalTypeUser
	out.Actor.PrincipalID = out.Actor.UserID
	out.DataAccessCeiling = dataAccess

	return out
}

func TestParamsIn(timezone string, params map[string]any) *agenttoolports.QueryToolParams {
	out := TestParams(params)
	out.Timezone = timezone

	return out
}

func TestParams(params map[string]any) *agenttoolports.QueryToolParams {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	return &agenttoolports.QueryToolParams{
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
