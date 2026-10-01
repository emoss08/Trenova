package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/pkg/pagination"
)

func equipmentClassStrings(classes []equipmenttype.Class) []string {
	if len(classes) == 0 {
		return nil
	}

	values := make([]string, 0, len(classes))
	for _, class := range classes {
		values = append(values, class.String())
	}

	return values
}

func equipmentTypeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.EquipmentTypeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func equipmentTypeConnectionToModel(
	result *pagination.CursorListResult[*equipmenttype.EquipmentType],
) (*gqlmodel.EquipmentTypeConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *equipmenttype.EquipmentType, cursor string) *gqlmodel.EquipmentTypeEdge {
			return &gqlmodel.EquipmentTypeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.EquipmentTypeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EquipmentTypeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
