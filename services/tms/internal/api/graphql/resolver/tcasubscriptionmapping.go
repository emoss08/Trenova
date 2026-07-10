package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/pkg/pagination"
)

func tcaSubscriptionConnectionToModel(
	result *pagination.CursorListResult[*tablechangealert.TCASubscription],
) (*gqlmodel.TCASubscriptionConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *tablechangealert.TCASubscription, cursor string) *gqlmodel.TCASubscriptionEdge {
			return &gqlmodel.TCASubscriptionEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.TCASubscriptionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.TCASubscriptionConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
