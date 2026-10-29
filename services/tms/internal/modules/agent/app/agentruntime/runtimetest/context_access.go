package runtimetest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListedDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository

	ByID map[pulid.ID]*agentdefinition.Definition
}

func (s *ListedDefinitions) ListByIDs(
	_ context.Context,
	req agentdefinitionports.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	found := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, id := range req.IDs {
		if definition, ok := s.ByID[id]; ok {
			found = append(found, definition)
		}
	}

	return found, nil
}

func (s *ListedDefinitions) List(
	_ context.Context,
	_ *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	items := make([]*agentdefinition.Definition, 0, len(s.ByID))
	for _, definition := range s.ByID {
		items = append(items, definition)
	}

	return &pagination.ListResult[*agentdefinition.Definition]{
		Items:      items,
		Total:      len(items),
		TotalExact: true,
	}, nil
}

func ChatAgent(name string) *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		ID:              pulid.MustNew("agdef_"),
		Name:            name,
		AutonomyCeiling: agent.TierPropose,
		Enabled:         true,
		TriggerMode:     agentdefinition.TriggerChat,
	}
	definition.ApplyDefaults()

	return definition
}
