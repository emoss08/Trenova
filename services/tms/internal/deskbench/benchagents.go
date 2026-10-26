package deskbench

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
)

func IsBenchAgent(definition *agentdefinition.Definition) bool {
	return definition != nil && strings.HasPrefix(definition.Name, BenchAgentPrefix)
}

func benchAgentScope(hidden bool) fx.Option {
	if !hidden {
		return fx.Options()
	}

	return fx.Decorate(
		func(
			inner agentdefinitionports.AgentDefinitionRepository,
		) agentdefinitionports.AgentDefinitionRepository {
			return withoutBenchAgents{AgentDefinitionRepository: inner}
		},
	)
}

type withoutBenchAgents struct {
	agentdefinitionports.AgentDefinitionRepository
}

func (w withoutBenchAgents) List(
	ctx context.Context,
	req *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	kept, err := w.organizationAgents(ctx, req)
	if err != nil {
		return nil, err
	}

	var window pagination.Info
	if req.Filter != nil {
		window = req.Filter.Pagination
	}

	return &pagination.ListResult[*agentdefinition.Definition]{
		Items:      sliceutils.Page(kept, window.Offset, window.Limit),
		Total:      len(kept),
		TotalExact: true,
	}, nil
}

func (w withoutBenchAgents) organizationAgents(
	ctx context.Context,
	req *agentdefinitionports.ListAgentDefinitionRequest,
) ([]*agentdefinition.Definition, error) {
	page := *req
	filter := pagination.QueryOptions{}
	if req.Filter != nil {
		filter = *req.Filter
	}
	page.Filter = &filter

	var kept []*agentdefinition.Definition
	for offset := 0; ; {
		filter.Pagination = pagination.Info{Limit: pagination.MaxLimit, Offset: offset}
		result, err := w.AgentDefinitionRepository.List(ctx, &page)
		if err != nil {
			return nil, err
		}
		for _, definition := range result.Items {
			if !IsBenchAgent(definition) {
				kept = append(kept, definition)
			}
		}
		offset += len(result.Items)
		if len(result.Items) == 0 || (result.TotalExact && offset >= result.Total) {
			return kept, nil
		}
	}
}
