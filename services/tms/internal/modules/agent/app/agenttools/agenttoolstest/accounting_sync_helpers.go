package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func AccountingSyncParams(params map[string]any) agenttoolports.ToolExecuteParams {
	out := ExecuteParams(params)
	out.IdempotencyKey = "call-1"
	out.Actor.PrincipalType = permissionports.PrincipalTypeUser
	return out
}
