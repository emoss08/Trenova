package locationcategoryresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func locationCategoryColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.LocationCategorySpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func locationCategoryConnectionToModel(
	result *pagination.CursorListResult[*locationcategory.LocationCategory],
) (*gqlmodel.LocationCategoryConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *locationcategory.LocationCategory, cursor string) *gqlmodel.LocationCategoryEdge {
			return &gqlmodel.LocationCategoryEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.LocationCategoryEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.LocationCategoryConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func locationCategoryFromInput(
	input *gqlmodel.LocationCategoryInput,
	authCtx *authctx.AuthContext,
) *locationcategory.LocationCategory {
	entity := &locationcategory.LocationCategory{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	applyLocationCategoryInput(entity, input)

	return entity
}

func applyLocationCategoryInput(
	entity *locationcategory.LocationCategory,
	input *gqlmodel.LocationCategoryInput,
) {
	entity.Name = input.Name
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.Type = input.Type
	entity.FacilityType = ""
	if input.FacilityType != nil {
		entity.FacilityType = *input.FacilityType
	}
	entity.Color = gqlscalar.StringValue(input.Color)
	entity.HasSecureParking = input.HasSecureParking
	entity.RequiresAppointment = input.RequiresAppointment
	entity.AllowsOvernight = input.AllowsOvernight
	entity.HasRestroom = input.HasRestroom
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}
