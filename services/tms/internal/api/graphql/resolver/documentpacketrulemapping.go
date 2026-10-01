package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/pkg/pagination"
)

func documentPacketRuleColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.DocumentPacketRuleSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func documentPacketRuleConnectionToModel(
	result *pagination.CursorListResult[*documentpacketrule.DocumentPacketRule],
) (*gqlmodel.DocumentPacketRuleConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *documentpacketrule.DocumentPacketRule, cursor string) *gqlmodel.DocumentPacketRuleEdge {
			return &gqlmodel.DocumentPacketRuleEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.DocumentPacketRuleEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DocumentPacketRuleConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
