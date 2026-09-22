package resolver

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
)

func agentRunEventColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentRunEventSpec,
		func(path string) bool { return graphql.FieldRequested(ctx, path) },
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentRunEventConnectionToModel(
	result *pagination.CursorListResult[*agent.AgentRunEvent],
) (*gqlmodel.AgentRunEventConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agent.AgentRunEvent, cursor string) *gqlmodel.AgentRunEventEdge {
			return &gqlmodel.AgentRunEventEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentRunEventEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, fmt.Errorf("build agent run event connection: %w", err)
	}

	return &gqlmodel.AgentRunEventConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
