package assistantservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Service) CommandAvailability(
	ctx context.Context,
	req *assistantports.CommandAvailabilityRequest,
	actor *permissionports.RequestActor,
) (*assistantports.CommandAvailability, error) {
	definition, err := s.usableAgent(ctx, req.AgentID, actor, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return &assistantports.CommandAvailability{
		Commands: s.runtime.AvailableCommands(ctx, actor, definition),
	}, nil
}

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
