package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/pkg/pagination"
)

func recurringShipmentConnectionToModel(
	result *pagination.CursorListResult[*recurringshipment.RecurringShipment],
) (*gqlmodel.RecurringShipmentConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *recurringshipment.RecurringShipment, cursor string) *gqlmodel.RecurringShipmentEdge {
			return &gqlmodel.RecurringShipmentEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.RecurringShipmentEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RecurringShipmentConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
