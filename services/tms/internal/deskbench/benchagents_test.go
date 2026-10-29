package deskbench

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/deskbench/benchagent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pagedDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository

	all      []*agentdefinition.Definition
	pageSize int
	requests []agentdefinitionports.ListAgentDefinitionRequest
}

func (p *pagedDefinitions) List(
	_ context.Context,
	req *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	seen := *req
	filter := *req.Filter
	seen.Filter = &filter
	p.requests = append(p.requests, seen)

	limit := min(req.Filter.Pagination.Limit, p.pageSize)

	return &pagination.ListResult[*agentdefinition.Definition]{
		Items:      sliceutils.Page(p.all, req.Filter.Pagination.Offset, limit),
		Total:      len(p.all),
		TotalExact: true,
	}, nil
}

func namedDefinitions(names ...string) []*agentdefinition.Definition {
	out := make([]*agentdefinition.Definition, 0, len(names))
	for _, name := range names {
		out = append(out, &agentdefinition.Definition{Name: name})
	}

	return out
}

func definitionNames(definitions []*agentdefinition.Definition) []string {
	out := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		out = append(out, definition.Name)
	}

	return out
}

func TestWithoutBenchAgents_ListsTheOrganizationsAgentsAcrossEveryPage(t *testing.T) {
	t.Parallel()

	inner := &pagedDefinitions{
		all: namedDefinitions(
			"Bench · Exceptions",
			"Billing exceptions",
			"Bench · Operations",
			"Compliance desk",
			"Dispatch desk",
			"Receivables",
		),
		pageSize: 2,
	}
	repo := withoutBenchAgents{AgentDefinitionRepository: inner}
	tenant := pagination.TenantInfo{}

	first, err := repo.List(t.Context(), &agentdefinitionports.ListAgentDefinitionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: 2},
		},
		EnabledOnly: true,
		ChatOnly:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Billing exceptions", "Compliance desk"}, definitionNames(first.Items))
	assert.Equal(t, 4, first.Total, "the total counts only the organization's agents")

	second, err := repo.List(t.Context(), &agentdefinitionports.ListAgentDefinitionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: 2, Offset: 2},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Dispatch desk", "Receivables"}, definitionNames(second.Items))

	require.NotEmpty(t, inner.requests)
	for _, request := range inner.requests {
		assert.Equal(t, pagination.MaxLimit, request.Filter.Pagination.Limit)
	}
	assert.True(t, inner.requests[0].EnabledOnly, "the caller's filters reach the repository")
	assert.True(t, inner.requests[0].ChatOnly)
}

func TestIsBenchAgent(t *testing.T) {
	t.Parallel()

	assert.True(
		t,
		benchagent.IsBenchAgent(
			&agentdefinition.Definition{Name: benchagent.BenchAgentPrefix + "Operations"},
		),
	)
	assert.False(t, benchagent.IsBenchAgent(&agentdefinition.Definition{Name: "Dispatch desk"}))
	assert.False(t, benchagent.IsBenchAgent(nil))
}
