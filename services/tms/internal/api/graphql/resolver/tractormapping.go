package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

func tractorRelationIncludes(
	ctx context.Context,
	nodePathPrefix string,
	includeEquipmentDetails *bool,
	includeFleetDetails *bool,
	includeWorkerDetails *bool,
) repositories.TractorRelationIncludes {
	return tractorRelationIncludesForFields(
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		nodePathPrefix,
		includeEquipmentDetails,
		includeFleetDetails,
		includeWorkerDetails,
	)
}

func tractorRelationIncludesForFields(
	fieldRequested func(string) bool,
	nodePathPrefix string,
	includeEquipmentDetails *bool,
	includeFleetDetails *bool,
	includeWorkerDetails *bool,
) repositories.TractorRelationIncludes {
	equipmentDetailsAllowed := includeEquipmentDetails == nil || *includeEquipmentDetails
	fleetDetailsAllowed := includeFleetDetails == nil || *includeFleetDetails
	workerDetailsAllowed := includeWorkerDetails == nil || *includeWorkerDetails
	selection := projection.Select(
		projection.TractorSpec,
		fieldRequested,
		projection.SelectOptions{
			PathPrefix: nodePathPrefix,
			Gates: map[string]bool{
				"equipmentDetails": equipmentDetailsAllowed,
				"fleetDetails":     fleetDetailsAllowed,
				"workerDetails":    workerDetailsAllowed,
			},
		},
	)
	primaryWorkerSelection, primaryWorkerSelected := selection.Relations["primaryWorker"]
	secondaryWorkerSelection, secondaryWorkerSelected := selection.Relations["secondaryWorker"]

	includes := repositories.TractorRelationIncludes{
		IncludeBusinessUnit:           selection.HasRelation("businessUnit"),
		IncludeOrganization:           selection.HasRelation("organization"),
		IncludeState:                  selection.HasRelation("state"),
		IncludeEquipmentType:          selection.HasRelation("equipmentType"),
		IncludeEquipmentManufacturer:  selection.HasRelation("equipmentManufacturer"),
		IncludeFleetCode:              selection.HasRelation("fleetCode"),
		IncludePrimaryWorker:          primaryWorkerSelected,
		IncludePrimaryWorkerState:     primaryWorkerSelection.Selection.HasRelation("state"),
		IncludePrimaryWorkerFleet:     primaryWorkerSelection.Selection.HasRelation("fleetCode"),
		IncludePrimaryWorkerManager:   primaryWorkerSelection.Selection.HasRelation("manager"),
		IncludeSecondaryWorker:        secondaryWorkerSelected,
		IncludeSecondaryWorkerState:   secondaryWorkerSelection.Selection.HasRelation("state"),
		IncludeSecondaryWorkerFleet:   secondaryWorkerSelection.Selection.HasRelation("fleetCode"),
		IncludeSecondaryWorkerManager: secondaryWorkerSelection.Selection.HasRelation("manager"),
		IncludeLastKnownLocation:      selection.HasSpecial("lastKnownLocation"),
		IncludeCustomFields:           selection.HasSpecial("customFields"),
		TractorColumns:                selection.Columns,
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
	if includes.IncludePrimaryWorker {
		includes.PrimaryWorkerColumns = selection.RelationColumns("primaryWorker")
	}
	if includes.IncludeSecondaryWorker {
		includes.SecondaryWorkerColumns = selection.RelationColumns("secondaryWorker")
	}

	return includes
}

func tractorConnectionToModel(
	result *pagination.CursorListResult[*tractor.Tractor],
) (*gqlmodel.TractorConnection, error) {
	return tractorConnectionFromItems(
		result.Items,
		result.HasNextPage,
		result.TotalCount,
		result.CursorSort,
		result,
	)
}

func tractorConnectionFromItems(
	items []*tractor.Tractor,
	hasNextPage bool,
	totalCount *int,
	sort []pagination.CursorSortField,
	cursorValues pagination.CursorValueProvider,
) (*gqlmodel.TractorConnection, error) {
	edges, err := entityCursorEdges(
		items,
		sort,
		cursorValues,
		func(node *tractor.Tractor, cursor string) *gqlmodel.TractorEdge {
			return &gqlmodel.TractorEdge{
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
		func(edge *gqlmodel.TractorEdge) string { return edge.Cursor },
	)

	return &gqlmodel.TractorConnection{
		Edges: edges,
		PageInfo: pageInfo(
			hasNextPage,
			endCursor,
		),
		TotalCount: totalCount,
	}, nil
}

func tractorLastKnownLocationReference(entity *tractor.Tractor) *gqlmodel.LocationReference {
	if entity.LastKnownLocationID.IsNil() {
		return nil
	}

	return &gqlmodel.LocationReference{
		ID:   entity.LastKnownLocationID.String(),
		Name: entity.LastKnownLocationName,
	}
}
