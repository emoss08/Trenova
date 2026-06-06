package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func trailerFromInput(
	input gqlmodel.TrailerInput,
	id pulid.ID,
	authCtx *authctx.AuthContext,
) (*trailer.Trailer, error) {
	equipmentTypeID, err := pulid.MustParse(input.EquipmentTypeID)
	if err != nil {
		return nil, err
	}
	equipmentManufacturerID, err := pulid.MustParse(input.EquipmentManufacturerID)
	if err != nil {
		return nil, err
	}
	registrationStateID, err := optionalID(input.RegistrationStateID)
	if err != nil {
		return nil, err
	}
	fleetCodeID, err := optionalID(input.FleetCodeID)
	if err != nil {
		return nil, err
	}

	status := domaintypes.EquipmentStatusAvailable
	if input.Status != nil {
		status = *input.Status
	}

	entity := &trailer.Trailer{
		ID:                      id,
		OrganizationID:          authCtx.OrganizationID,
		BusinessUnitID:          authCtx.BusinessUnitID,
		EquipmentTypeID:         equipmentTypeID,
		EquipmentManufacturerID: equipmentManufacturerID,
		RegistrationStateID:     registrationStateID,
		FleetCodeID:             fleetCodeID,
		Status:                  status,
		Code:                    input.Code,
		Model:                   stringValue(input.Model),
		Make:                    stringValue(input.Make),
		Year:                    input.Year,
		LicensePlateNumber:      stringValue(input.LicensePlateNumber),
		Vin:                     stringValue(input.Vin),
		RegistrationNumber:      stringValue(input.RegistrationNumber),
		MaxLoadWeight:           input.MaxLoadWeight,
		LastInspectionDate:      int64Ptr(input.LastInspectionDate),
		RegistrationExpiry:      int64Ptr(input.RegistrationExpiry),
		CustomFields:            input.CustomFields,
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return entity, nil
}

func applyTrailerPatch(entity *trailer.Trailer, input gqlmodel.TrailerPatchInput) error {
	if input.EquipmentTypeID != nil {
		id, err := pulid.MustParse(*input.EquipmentTypeID)
		if err != nil {
			return err
		}
		entity.EquipmentTypeID = id
	}
	if input.EquipmentManufacturerID != nil {
		id, err := pulid.MustParse(*input.EquipmentManufacturerID)
		if err != nil {
			return err
		}
		entity.EquipmentManufacturerID = id
	}
	if value, ok := input.RegistrationStateID.ValueOK(); ok {
		id, err := optionalID(value)
		if err != nil {
			return err
		}
		entity.RegistrationStateID = id
	}
	if value, ok := input.FleetCodeID.ValueOK(); ok {
		id, err := optionalID(value)
		if err != nil {
			return err
		}
		entity.FleetCodeID = id
	}
	if input.Status != nil {
		entity.Status = *input.Status
	}
	if input.Code != nil {
		entity.Code = *input.Code
	}
	if input.Model != nil {
		entity.Model = *input.Model
	}
	if input.Make != nil {
		entity.Make = *input.Make
	}
	if input.Year != nil {
		entity.Year = input.Year
	}
	if input.LicensePlateNumber != nil {
		entity.LicensePlateNumber = *input.LicensePlateNumber
	}
	if input.Vin != nil {
		entity.Vin = *input.Vin
	}
	if input.RegistrationNumber != nil {
		entity.RegistrationNumber = *input.RegistrationNumber
	}
	if input.MaxLoadWeight != nil {
		entity.MaxLoadWeight = input.MaxLoadWeight
	}
	if input.LastInspectionDate != nil {
		entity.LastInspectionDate = int64Ptr(input.LastInspectionDate)
	}
	if input.RegistrationExpiry != nil {
		entity.RegistrationExpiry = int64Ptr(input.RegistrationExpiry)
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
	if input.CustomFields != nil {
		entity.CustomFields = input.CustomFields
	}

	return nil
}

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
	return trailerConnectionFromItems(result.Items, result.HasNextPage, nil)
}

func trailerListConnectionToModel(
	result *pagination.ListResult[*trailer.Trailer],
	offset int,
) (*gqlmodel.TrailerConnection, error) {
	hasNextPage := offset+len(result.Items) < result.Total
	return trailerConnectionFromItems(result.Items, hasNextPage, &result.Total)
}

func trailerConnectionFromItems(
	items []*trailer.Trailer,
	hasNextPage bool,
	totalCount *int,
) (*gqlmodel.TrailerConnection, error) {
	edges := make([]*gqlmodel.TrailerEdge, 0, len(items))
	for _, entity := range items {
		cursor, err := pagination.EncodeCursor(pagination.Cursor{
			CreatedAt: entity.CreatedAt,
			ID:        entity.ID,
		})
		if err != nil {
			return nil, err
		}
		edges = append(edges, &gqlmodel.TrailerEdge{
			Node:   entity,
			Cursor: cursor,
		})
	}

	var endCursor *string
	if len(edges) > 0 {
		endCursor = &edges[len(edges)-1].Cursor
	}

	return &gqlmodel.TrailerConnection{
		Edges: edges,
		PageInfo: &gqlmodel.PageInfo{
			HasNextPage: hasNextPage,
			EndCursor:   endCursor,
		},
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
