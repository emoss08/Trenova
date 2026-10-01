package resolvertest

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
)

type DelegateDefinitionsRepo struct {
	repositories.AgentDefinitionRepository

	ByID  map[pulid.ID]*agentdefinition.Definition
	Reads int
}

func (r *DelegateDefinitionsRepo) ListByIDs(
	_ context.Context,
	req repositories.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	r.Reads++
	out := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, id := range req.IDs {
		if definition, ok := r.ByID[id]; ok {
			out = append(out, definition)
		}
	}

	return out, nil
}
