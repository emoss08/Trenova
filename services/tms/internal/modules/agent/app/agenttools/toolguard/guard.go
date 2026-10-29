package toolguard

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

var ErrMissingActor = errors.New("agent tool requires an actor")

var ErrAgentCannotApprove = errors.New("agent principals cannot approve billing queue items")

var ErrTenantMismatch = errors.New("tool parameters do not match the actor tenant")

var ErrMissingIdempotencyKey = errors.New("idempotency key is required for this tool")

func GuardExecute(tool agenttoolports.AgentTool, params *agenttoolports.ToolExecuteParams) error {
	if err := GuardPreview(tool, params); err != nil {
		return err
	}

	if tool.Policy().Idempotent && strings.TrimSpace(params.IdempotencyKey) == "" {
		return ErrMissingIdempotencyKey
	}

	return nil
}

func GuardPreview(tool agenttoolports.AgentTool, params *agenttoolports.ToolExecuteParams) error {
	if params.Actor == nil {
		return ErrMissingActor
	}

	if params.Actor.IsAgent() && tool.Policy().Operation == permission.OpApprove {
		return ErrAgentCannotApprove
	}

	if params.Actor.OrganizationID != params.OrganizationID ||
		params.Actor.BusinessUnitID != params.BusinessUnitID {
		return ErrTenantMismatch
	}

	return nil
}
