package shipmentmapping

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
)

func ShipmentConnectionToModel(
	ctx context.Context,
	result *pagination.CursorListResult[*shipmentdomain.Shipment],
) (*gqlmodel.ShipmentConnection, error) {
	edges, endCursor, err := gqlconnection.MappedEntityCursorEdges(
		result.Items,
		result,
		func(entity *shipmentdomain.Shipment) (*gqlmodel.Shipment, error) {
			return ShipmentToModel(ctx, entity)
		},
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
		Edges:           edges,
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
		PageInfo: gqlconnection.PageInfo(
			result.HasNextPage,
			endCursor,
		),
	}, nil
}

func ShipmentCommentConnectionToModel(
	result *pagination.CursorListResult[*shipmentdomain.ShipmentComment],
) (*gqlmodel.ShipmentCommentConnection, error) {
	edges, endCursor, err := gqlconnection.MappedEntityCursorEdges(
		result.Items,
		result,
		ShipmentCommentToModel,
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
		Edges:           edges,
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
		PageInfo: gqlconnection.PageInfo(
			result.HasNextPage,
			endCursor,
		),
	}, nil
}
