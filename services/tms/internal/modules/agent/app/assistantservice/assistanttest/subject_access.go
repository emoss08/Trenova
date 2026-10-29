package assistanttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

type SubjectPermissions struct {
	permissionports.PermissionEngine

	Allowed map[string]bool
	Asked   []string
	// Granted are the agents restricted to roles that the person's roles
	// grant.
	Granted []pulid.ID
	// AgentChecks are the agents MayUseAgent was asked about.
	AgentChecks []pulid.ID
}

func (p *SubjectPermissions) usable(operation permission.Operation) *permissionports.UsableAgents {
	return &permissionports.UsableAgents{
		Assistant:  p.Allowed[permission.ResourceAssistant.String()+":"+string(operation)],
		GrantedIDs: p.Granted,
	}
}

func (p *SubjectPermissions) AgentsUsable(
	_ context.Context,
	_ *permissionports.RequestActor,
	operation permission.Operation,
) (*permissionports.UsableAgents, error) {
	p.Asked = append(p.Asked, permission.ResourceAssistant.String()+":"+string(operation))

	return p.usable(operation), nil
}

func (p *SubjectPermissions) MayUseAgent(
	_ context.Context,
	_ *permissionports.RequestActor,
	definition *agentdefinition.Definition,
) (bool, error) {
	p.AgentChecks = append(p.AgentChecks, definition.ID)

	return p.usable(permission.OpCreate).Allows(definition), nil
}

func (p *SubjectPermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	key := req.Resource + ":" + string(req.Operation)
	p.Asked = append(p.Asked, key)

	return &permissionports.PermissionCheckResult{Allowed: p.Allowed[key]}, nil
}
