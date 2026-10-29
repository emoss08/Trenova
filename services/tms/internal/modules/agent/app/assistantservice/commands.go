package assistantservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Service) assertHonoursCommand(
	ctx context.Context,
	actor *permissionports.RequestActor,
	definition *agentdefinition.Definition,
	command agentextension.Command,
) error {
	if slices.Contains(s.runtime.AvailableCommands(ctx, actor, definition), command) {
		return nil
	}

	spec, _ := agentextension.CommandSpecFor(command)

	return errortypes.NewValidationError(
		"content", errortypes.ErrInvalid, spec.UnavailableReason, definition.Name,
	)
}
