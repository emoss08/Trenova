package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
)

func agentRunColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentRunSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentProposalColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentProposalSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentExceptionColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentExceptionSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentRunConnectionToModel(
	result *pagination.CursorListResult[*agent.AgentRun],
) (*gqlmodel.AgentRunConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agent.AgentRun, cursor string) *gqlmodel.AgentRunEdge {
			return &gqlmodel.AgentRunEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentRunEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentRunConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func agentProposalConnectionToModel(
	result *pagination.CursorListResult[*agent.AgentProposal],
) (*gqlmodel.AgentProposalConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agent.AgentProposal, cursor string) *gqlmodel.AgentProposalEdge {
			return &gqlmodel.AgentProposalEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentProposalEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentProposalConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func agentExceptionConnectionToModel(
	result *pagination.CursorListResult[*agent.AgentException],
) (*gqlmodel.AgentExceptionConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agent.AgentException, cursor string) *gqlmodel.AgentExceptionEdge {
			return &gqlmodel.AgentExceptionEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentExceptionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentExceptionConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
