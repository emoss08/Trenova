package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/pkg/pagination"
)

func serviceFailureReasonCodeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.ServiceFailureReasonCodeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func serviceFailureReasonCodeConnectionToModel(
	result *pagination.CursorListResult[*servicefailure.ReasonCode],
) (*gqlmodel.ServiceFailureReasonCodeConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *servicefailure.ReasonCode, cursor string) *gqlmodel.ServiceFailureReasonCodeEdge {
			return &gqlmodel.ServiceFailureReasonCodeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.ServiceFailureReasonCodeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ServiceFailureReasonCodeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
