package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func equipmentTypeFromInput(
	input gqlmodel.EquipmentTypeInput,
	id pulid.ID,
	authCtx *authctx.AuthContext,
) *equipmenttype.EquipmentType {
	status := domaintypes.StatusActive
	if input.Status != nil {
		status = *input.Status
	}

	entity := &equipmenttype.EquipmentType{
		ID:             id,
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		Status:         status,
		Code:           input.Code,
		Description:    stringValue(input.Description),
		Class:          input.Class,
		Color:          stringValue(input.Color),
		InteriorLength: input.InteriorLength,
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return entity
}

func applyEquipmentTypePatch(
	entity *equipmenttype.EquipmentType,
	input gqlmodel.EquipmentTypePatchInput,
) {
	if input.Status != nil {
		entity.Status = *input.Status
	}
	if input.Code != nil {
		entity.Code = *input.Code
	}
	if value, ok := input.Description.ValueOK(); ok {
		entity.Description = stringValue(value)
	}
	if input.Class != nil {
		entity.Class = *input.Class
	}
	if value, ok := input.Color.ValueOK(); ok {
		entity.Color = stringValue(value)
	}
	if value, ok := input.InteriorLength.ValueOK(); ok {
		entity.InteriorLength = value
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}

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
