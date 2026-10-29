package usableagents

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"go.uber.org/zap"
)

// For is what the person may use of the organization's agents in
// a conversation: whether they may talk to agents at all, and which agents
// restricted to roles their roles grant. Nobody but a person talks to an
// agent, and a check that cannot be made offers nothing.
func For(
	ctx context.Context,
	permissions permissionports.PermissionEngine,
	actor *permissionports.RequestActor,
	logger *zap.Logger,
) *permissionports.UsableAgents {
	if permissions == nil || actor == nil ||
		actor.PrincipalType != permissionports.PrincipalTypeUser {
		return &permissionports.UsableAgents{}
	}

	usable, err := permissions.AgentsUsable(ctx, actor, permission.OpCreate)
	if err != nil {
		if logger != nil {
			logger.Warn("could not check which agents the person may use; "+
				"withholding them", zap.Error(err))
		}

		return &permissionports.UsableAgents{}
	}
	if usable == nil {
		return &permissionports.UsableAgents{}
	}

	return usable
}
