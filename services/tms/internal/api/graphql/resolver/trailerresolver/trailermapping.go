package trailerresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func trailerRelationIncludes(
	ctx context.Context,
	nodePathPrefix string,
	includeEquipmentDetails *bool,
	includeFleetDetails *bool,
) trailerports.TrailerRelationIncludes {
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
) trailerports.TrailerRelationIncludes {
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
	_, ownerWorkerSelected := selection.Relations["ownerWorker"]

	includes := trailerports.TrailerRelationIncludes{
		IncludeBusinessUnit:          selection.HasRelation("businessUnit"),
		IncludeOrganization:          selection.HasRelation("organization"),
		IncludeRegistrationState:     selection.HasRelation("registrationState"),
		IncludeEquipmentType:         selection.HasRelation("equipmentType"),
		IncludeEquipmentManufacturer: selection.HasRelation("equipmentManufacturer"),
		IncludeFleetCode:             fleetCodeSelected,
		IncludeFleetManager:          fleetCodeSelection.Selection.HasRelation("manager"),
		IncludeOwnerWorker:           ownerWorkerSelected,
		IncludeLastKnownLocation:     selection.HasSpecial("lastKnownLocation"),
		IncludeInProgressMove:        selection.HasSpecial("inProgressMove"),
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
	if includes.IncludeOwnerWorker {
		includes.OwnerWorkerColumns = selection.RelationColumns("ownerWorker")
	}

	return includes
}

func trailerConnectionToModel(
	result *pagination.CursorListResult[*trailer.Trailer],
) (*gqlmodel.TrailerConnection, error) {
	edges, err := gqlconnection.EntityCursorEdges(
		result.Items,
		result,
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
	endCursor := gqlconnection.LastEdgeCursor(
		edges,
		func(edge *gqlmodel.TrailerEdge) string { return edge.Cursor },
	)

	return &gqlmodel.TrailerConnection{
		Edges: edges,
		PageInfo: gqlconnection.PageInfo(
			result.HasNextPage,
			endCursor,
		),
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
	}, nil
}

func lastKnownLocationReference(entity *trailer.Trailer) *gqlmodel.LocationReference {
	if entity.LastKnownLocationID.IsNil() {
		return nil
	}

	return &gqlmodel.LocationReference{
		ID:        entity.LastKnownLocationID.String(),
		Name:      entity.LastKnownLocationName,
		Latitude:  entity.LastKnownLocationLatitude,
		Longitude: entity.LastKnownLocationLongitude,
	}
}
