package commodityresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func commodityColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.CommoditySpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func commodityConnectionToModel(
	result *pagination.CursorListResult[*commodity.Commodity],
) (*gqlmodel.CommodityConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *commodity.Commodity, cursor string) *gqlmodel.CommodityEdge {
			return &gqlmodel.CommodityEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.CommodityEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.CommodityConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func commodityFromInput(
	input *gqlmodel.CommodityInput,
	authCtx *authctx.AuthContext,
) (*commodity.Commodity, error) {
	entity := &commodity.Commodity{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	if err := applyCommodityInput(entity, input); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyCommodityInput(entity *commodity.Commodity, input *gqlmodel.CommodityInput) error {
	hazardousMaterialID, err := gqlid.OptionalScopedID(
		"hazardousMaterialId",
		input.HazardousMaterialID,
	)
	if err != nil {
		return err
	}

	entity.Status = input.Status
	entity.Name = input.Name
	entity.Description = input.Description
	entity.HazardousMaterialID = hazardousMaterialID
	entity.MinTemperature = input.MinTemperature
	entity.MaxTemperature = input.MaxTemperature
	entity.WeightPerUnit = input.WeightPerUnit
	entity.LinearFeetPerUnit = input.LinearFeetPerUnit
	entity.LengthPerUnit = input.LengthPerUnit
	entity.WidthPerUnit = input.WidthPerUnit
	entity.HeightPerUnit = input.HeightPerUnit
	entity.MaxQuantityPerShipment = input.MaxQuantityPerShipment
	entity.FreightClass = ""
	if input.FreightClass != nil {
		entity.FreightClass = *input.FreightClass
	}
	entity.LoadingInstructions = gqlscalar.StringValue(input.LoadingInstructions)
	entity.Stackable = input.Stackable
	entity.Fragile = input.Fragile
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return nil
}

func applyCommodityPatch(entity *commodity.Commodity, input *gqlmodel.CommodityPatchInput) error {
	if err := gqlscalar.ApplyRequired(
		"status",
		"Status",
		input.Status,
		&entity.Status,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("name", "Name", input.Name, &entity.Name); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"description",
		"Description",
		input.Description,
		&entity.Description,
	); err != nil {
		return err
	}
	if raw, ok := input.HazardousMaterialID.ValueOK(); ok {
		hazardousMaterialID, err := gqlid.OptionalScopedID("hazardousMaterialId", raw)
		if err != nil {
			return err
		}
		entity.HazardousMaterialID = hazardousMaterialID
	}
	gqlscalar.ApplyNullable(input.MinTemperature, &entity.MinTemperature)
	gqlscalar.ApplyNullable(input.MaxTemperature, &entity.MaxTemperature)
	gqlscalar.ApplyNullable(input.WeightPerUnit, &entity.WeightPerUnit)
	gqlscalar.ApplyNullable(input.LinearFeetPerUnit, &entity.LinearFeetPerUnit)
	gqlscalar.ApplyNullable(input.LengthPerUnit, &entity.LengthPerUnit)
	gqlscalar.ApplyNullable(input.WidthPerUnit, &entity.WidthPerUnit)
	gqlscalar.ApplyNullable(input.HeightPerUnit, &entity.HeightPerUnit)
	gqlscalar.ApplyNullable(input.MaxQuantityPerShipment, &entity.MaxQuantityPerShipment)
	gqlscalar.ApplyClearable(input.FreightClass, &entity.FreightClass)
	gqlscalar.ApplyClearable(input.LoadingInstructions, &entity.LoadingInstructions)
	if err := gqlscalar.ApplyRequired(
		"stackable",
		"Stackable",
		input.Stackable,
		&entity.Stackable,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"fragile",
		"Fragile",
		input.Fragile,
		&entity.Fragile,
	); err != nil {
		return err
	}

	return gqlscalar.ApplyRequiredInt("version", "Version", input.Version, &entity.Version)
}
