package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Service) BrowserAvailability(
	ctx context.Context,
	req *agentports.BrowserAvailabilityRequest,
	actor *permissionports.RequestActor,
) (*agentports.BrowserAvailability, error) {
	definition, err := s.usableAgent(ctx, req.AgentID, actor, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return &agentports.BrowserAvailability{
		Available: s.worksInBrowser(ctx, actor, definition),
	}, nil
}

func (s *Service) worksInBrowser(
	ctx context.Context,
	actor *permissionports.RequestActor,
	definition *agentdefinition.Definition,
) bool {
	return agentextension.WorksInBrowser(s.runtime.BrowserTools(ctx, actor, definition))
}

func (s *Service) assertWorksInBrowser(
	ctx context.Context,
	actor *permissionports.RequestActor,
	definition *agentdefinition.Definition,
) error {
	if s.worksInBrowser(ctx, actor, definition) {
		return nil
	}

	return errortypes.NewValidationError("content", errortypes.ErrInvalid,
		"{0} cannot work in the browser. Portal automation has to be turned on, the agent "+
			"has to hold its portal tools, and you need permission to use them.",
		definition.Name)
}
