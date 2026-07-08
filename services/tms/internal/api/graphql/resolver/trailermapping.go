package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

func trailerRelationIncludes(
	ctx context.Context,
	nodePathPrefix string,
	includeEquipmentDetails *bool,
	includeFleetDetails *bool,
) repositories.TrailerRelationIncludes {
	return trailerRelationIncludesForFields(
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		nodePathPrefix,
		includeEquipmentDetails,
		includeFleetDetails,
	)
}

func trailerRelationIncludesForFields(
	fieldRequested func(string) bool,
	nodePathPrefix string,
	includeEquipmentDetails *bool,
	includeFleetDetails *bool,
) repositories.TrailerRelationIncludes {
	equipmentDetailsAllowed := includeEquipmentDetails == nil || *includeEquipmentDetails
	fleetDetailsAllowed := includeFleetDetails == nil || *includeFleetDetails
	selection := projection.Select(
		projection.TrailerSpec,
		fieldRequested,
		projection.SelectOptions{
			PathPrefix: nodePathPrefix,
			Gates: map[string]bool{
				"equipmentDetails": equipmentDetailsAllowed,
				"fleetDetails":     fleetDetailsAllowed,
			},
		},
	)
	fleetCodeSelection, fleetCodeSelected := selection.Relations["fleetCode"]

	includes := repositories.TrailerRelationIncludes{
		IncludeBusinessUnit:          selection.HasRelation("businessUnit"),
		IncludeOrganization:          selection.HasRelation("organization"),
		IncludeRegistrationState:     selection.HasRelation("registrationState"),
		IncludeEquipmentType:         selection.HasRelation("equipmentType"),
		IncludeEquipmentManufacturer: selection.HasRelation("equipmentManufacturer"),
		IncludeFleetCode:             fleetCodeSelected,
		IncludeFleetManager:          fleetCodeSelection.Selection.HasRelation("manager"),
		IncludeLastKnownLocation:     selection.HasSpecial("lastKnownLocation"),
		IncludeCustomFields:          selection.HasSpecial("customFields"),
		TrailerColumns:               selection.Columns,
	}
	if includes.IncludeEquipmentType {
		includes.EquipmentTypeColumns = selection.RelationColumns("equipmentType")
	}
	if includes.IncludeEquipmentManufacturer {
		includes.EquipmentManufacturerColumns = selection.RelationColumns("equipmentManufacturer")
	}
	if includes.IncludeFleetCode {
		includes.FleetCodeColumns = selection.RelationColumns("fleetCode")
	}

	return includes
}

func trailerFieldPath(prefix, field string) string {
	if prefix == "" {
		return field
	}

	return prefix + "." + field
}

func trailerConnectionToModel(
	result *pagination.CursorListResult[*trailer.Trailer],
) (*gqlmodel.TrailerConnection, error) {
	return trailerConnectionFromItems(
		result.Items,
		result.HasNextPage,
		result.TotalCount,
		result.CursorSort,
		result,
	)
}

func trailerConnectionFromItems(
	items []*trailer.Trailer,
	hasNextPage bool,
	totalCount *int,
	sort []pagination.CursorSortField,
	cursorValues pagination.CursorValueProvider,
) (*gqlmodel.TrailerConnection, error) {
	edges, err := entityCursorEdges(
		items,
		sort,
		cursorValues,
		func(node *trailer.Trailer, cursor string) *gqlmodel.TrailerEdge {
			return &gqlmodel.TrailerEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
	)
	if err != nil {
		return nil, err
	}
	endCursor := lastEdgeCursor(
		edges,
		func(edge *gqlmodel.TrailerEdge) string { return edge.Cursor },
	)

	return &gqlmodel.TrailerConnection{
		Edges: edges,
		PageInfo: pageInfo(
			hasNextPage,
			endCursor,
		),
		TotalCount: totalCount,
	}, nil
}

func lastKnownLocationReference(entity *trailer.Trailer) *gqlmodel.LocationReference {
	if entity.LastKnownLocationID.IsNil() {
		return nil
	}

	return &gqlmodel.LocationReference{
		ID:   entity.LastKnownLocationID.String(),
		Name: entity.LastKnownLocationName,
	}
}
