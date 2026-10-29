package assistantthreads

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func (s *Service) CommandAvailability(
	ctx context.Context,
	req *assistantports.CommandAvailabilityRequest,
	actor *permissionports.RequestActor,
) (*assistantports.CommandAvailability, error) {
	definition, err := s.access().UsableAgent(ctx, req.AgentID, actor, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return &assistantports.CommandAvailability{
		Commands: s.runtime.AvailableCommands(ctx, actor, definition),
	}, nil
}
