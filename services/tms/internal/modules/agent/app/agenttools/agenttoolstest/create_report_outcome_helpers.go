package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func PersonParams(args map[string]any) agenttoolports.ToolExecuteParams {
	params := ExecuteParams(args)
	params.Actor.PrincipalType = permissionports.PrincipalTypeUser
	params.Actor.PrincipalID = params.Actor.UserID

	return params
}
