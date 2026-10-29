package assistanttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type DelegateDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository

	ByID   map[pulid.ID]*agentdefinition.Definition
	Listed int
}

func (s *DelegateDefinitions) ListByIDs(
	_ context.Context,
	req agentdefinitionports.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	s.Listed++
	found := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, id := range req.IDs {
		if definition, ok := s.ByID[id]; ok {
			found = append(found, definition)
		}
	}

	return found, nil
}

func (s *DelegateDefinitions) GetByID(
	_ context.Context,
	req agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	if definition, ok := s.ByID[req.ID]; ok {
		return definition, nil
	}

	return nil, errortypes.NewNotFoundError("AgentDefinition not found")
}
