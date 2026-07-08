package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
)

func shipmentConnectionToModel(
	result *pagination.CursorListResult[*shipmentdomain.Shipment],
) (*gqlmodel.ShipmentConnection, error) {
	edges, endCursor, err := mappedEntityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		shipmentToModel,
		func(node *gqlmodel.Shipment, cursor string) *gqlmodel.ShipmentEdge {
			return &gqlmodel.ShipmentEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.ShipmentConnection{
		Edges:      edges,
		TotalCount: result.TotalCount,
		PageInfo: pageInfo(
			result.HasNextPage,
			endCursor,
		),
	}, nil
}

func shipmentCommentConnectionToModel(
	result *pagination.CursorListResult[*shipmentdomain.ShipmentComment],
) (*gqlmodel.ShipmentCommentConnection, error) {
	edges, endCursor, err := mappedEntityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		shipmentCommentToModel,
		func(node *gqlmodel.ShipmentComment, cursor string) *gqlmodel.ShipmentCommentEdge {
			return &gqlmodel.ShipmentCommentEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.ShipmentCommentConnection{
		Edges:      edges,
		TotalCount: result.TotalCount,
		PageInfo: pageInfo(
			result.HasNextPage,
			endCursor,
		),
	}, nil
}
