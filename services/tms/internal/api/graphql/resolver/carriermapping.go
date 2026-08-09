package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/pkg/pagination"
)

func carrierColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.CarrierSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func carrierConnectionToModel(
	result *pagination.CursorListResult[*carrier.Carrier],
) (*gqlmodel.CarrierConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *carrier.Carrier, cursor string) *gqlmodel.CarrierEdge {
			return &gqlmodel.CarrierEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.CarrierEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.CarrierConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
