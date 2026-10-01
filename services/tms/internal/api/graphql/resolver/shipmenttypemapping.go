package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/pkg/pagination"
)

func shipmentTypeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.ShipmentTypeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func shipmentTypeConnectionToModel(
	result *pagination.CursorListResult[*shipmenttype.ShipmentType],
) (*gqlmodel.ShipmentTypeConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *shipmenttype.ShipmentType, cursor string) *gqlmodel.ShipmentTypeEdge {
			return &gqlmodel.ShipmentTypeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.ShipmentTypeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ShipmentTypeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
