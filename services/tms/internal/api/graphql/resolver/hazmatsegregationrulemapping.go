package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/pkg/pagination"
)

func hazmatSegregationRuleColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.HazmatSegregationRuleSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func hazmatSegregationRuleConnectionToModel(
	result *pagination.CursorListResult[*hazmatsegregationrule.HazmatSegregationRule],
) (*gqlmodel.HazmatSegregationRuleConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *hazmatsegregationrule.HazmatSegregationRule, cursor string) *gqlmodel.HazmatSegregationRuleEdge {
			return &gqlmodel.HazmatSegregationRuleEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.HazmatSegregationRuleEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.HazmatSegregationRuleConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
