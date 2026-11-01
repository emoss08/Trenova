package locationresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/authctx"
)

func locationFromInput(
	input *gqlmodel.LocationInput,
	authCtx *authctx.AuthContext,
) (*location.Location, error) {
	entity := &location.Location{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	if err := applyLocationInput(entity, input); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyLocationInput(entity *location.Location, input *gqlmodel.LocationInput) error {
	categoryID, err := gqlid.RequiredID("locationCategoryId", input.LocationCategoryID)
	if err != nil {
		return err
	}
	stateID, err := gqlid.RequiredID("stateId", input.StateID)
	if err != nil {
		return err
	}

	entity.Status = input.Status
	if code := gqlscalar.StringValue(input.Code); code != "" {
		entity.Code = code
	}
	entity.Name = input.Name
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.LocationCategoryID = categoryID
	entity.StateID = stateID
	entity.AddressLine1 = input.AddressLine1
	entity.AddressLine2 = gqlscalar.StringValue(input.AddressLine2)
	entity.City = input.City
	entity.PostalCode = input.PostalCode
	entity.Timezone = gqlscalar.StringValue(input.Timezone)
	entity.PlaceID = gqlscalar.StringValue(input.PlaceID)
	entity.IsGeocoded = input.IsGeocoded
	entity.Longitude = input.Longitude
	entity.Latitude = input.Latitude
	entity.GeofenceType = input.GeofenceType
	entity.GeofenceRadiusMeters = input.GeofenceRadiusMeters
	entity.GeofenceVertices = geofenceVerticesFromInput(input.GeofenceVertices)
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return nil
}

func geofenceVerticesFromInput(input []*gqlmodel.GeofenceVertexInput) []geofence.Vertex {
	vertices := make([]geofence.Vertex, 0, len(input))
	for _, vertex := range input {
		if vertex == nil {
			continue
		}
		vertices = append(vertices, geofence.Vertex{
			Latitude:  vertex.Latitude,
			Longitude: vertex.Longitude,
		})
	}

	return vertices
}

func applyLocationPatch(entity *location.Location, input *gqlmodel.LocationPatchInput) error {
	if err := gqlscalar.ApplyRequired("status", "Status", input.Status, &entity.Status); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("name", "Name", input.Name, &entity.Name); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.Description, &entity.Description)
	if err := gqlid.ApplyRequiredID(
		"locationCategoryId",
		"Location category",
		input.LocationCategoryID,
		&entity.LocationCategoryID,
	); err != nil {
		return err
	}
	if err := gqlid.ApplyRequiredID("stateId", "State", input.StateID, &entity.StateID); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"addressLine1",
		"Address line 1",
		input.AddressLine1,
		&entity.AddressLine1,
	); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.AddressLine2, &entity.AddressLine2)
	if err := gqlscalar.ApplyRequired("city", "City", input.City, &entity.City); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"postalCode",
		"Postal code",
		input.PostalCode,
		&entity.PostalCode,
	); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.Timezone, &entity.Timezone)

	return gqlscalar.ApplyRequiredInt("version", "Version", input.Version, &entity.Version)
}
