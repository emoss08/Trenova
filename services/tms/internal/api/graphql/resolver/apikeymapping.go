package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/pkg/pagination"
)

func apiKeyConnectionToModel(
	result *pagination.CursorListResult[*apikey.Key],
) (*gqlmodel.APIKeyConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *apikey.Key, cursor string) *gqlmodel.APIKeyEdge {
			return &gqlmodel.APIKeyEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.APIKeyEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.APIKeyConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
