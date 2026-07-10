package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/pkg/pagination"
)

func serviceFailureConnectionToModel(
	result *pagination.CursorListResult[*servicefailure.ServiceFailure],
) (*gqlmodel.ServiceFailureConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *servicefailure.ServiceFailure, cursor string) *gqlmodel.ServiceFailureEdge {
			return &gqlmodel.ServiceFailureEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.ServiceFailureEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ServiceFailureConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
