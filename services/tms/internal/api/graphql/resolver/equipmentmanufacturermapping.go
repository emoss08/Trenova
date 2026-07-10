package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/pkg/pagination"
)

func equipmentManufacturerColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.EquipmentManufacturerSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func equipmentManufacturerConnectionToModel(
	result *pagination.CursorListResult[*equipmentmanufacturer.EquipmentManufacturer],
) (*gqlmodel.EquipmentManufacturerConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *equipmentmanufacturer.EquipmentManufacturer, cursor string) *gqlmodel.EquipmentManufacturerEdge {
			return &gqlmodel.EquipmentManufacturerEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.EquipmentManufacturerEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EquipmentManufacturerConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
