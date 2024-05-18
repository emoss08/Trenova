package models

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/location"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
)

// GetLocations retrieves a list of locations for a given organization and business unit.
// It returns a slice of Location entities, the total number of location records, and an error object.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - limit int: The maximum number of records to return.
//   - offset int: The number of records to skip before starting to return records.
//   - orgID uuid.UUID: The identifier of the organization.
//   - buID uuid.UUID: The identifier of the business unit.
//
// Returns:
//   - []*ent.Location: A slice of Location entities.
//   - int: The total number of location records.
//   - error: An error object that indicates why the retrieval failed, nil if no error occurred.
func (r *QueryService) GetLocations(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.Location, int, error) {
	count, err := r.Client.Location.Query().Where(
		location.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	entities, err := r.Client.Location.Query().
		Limit(limit).
		WithLocationCategory().
		WithComments().
		WithContacts().
		WithState().
		Offset(offset).
		Order(
			location.ByName(
				sql.OrderDesc(),
			),
		).
		Where(
			location.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, count, nil
}

// CreateLocationEntity creates a location entity.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *LocationRequest: The location request containing the details of the location to be created.
//
// Returns:
//   - *ent.Location: A pointer to the newly created Location entity.
//   - error: An error object that indicates why the creation failed, nil if no error occurred.
func (r *QueryService) CreateLocationEntity(ctx context.Context, tx *ent.Tx, entity *types.LocationRequest) (*ent.Location, error) {
	createdEntity, err := tx.Location.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetNillableLocationCategoryID(entity.LocationCategoryID).
		SetName(entity.Name).
		SetAddressLine1(entity.AddressLine1).
		SetAddressLine2(entity.AddressLine2).
		SetCity(entity.City).
		SetStateID(entity.StateID).
		SetPostalCode(entity.PostalCode).
		Save(ctx)
	if err != nil {
		r.Logger.Err(err).Msg("Error creating location")
	}

	return createdEntity, err
}

// UpdateLocationEntity updates a location entity.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *LocationUpdateRequest: The location update request containing the details of the location to be updated.
//
// Returns:
//   - *ent.Location: A pointer to the updated Location entity.
//   - error: An error object that indicates why the update failed, nil if no error occurred.
func (r *QueryService) UpdateLocationEntity(ctx context.Context, tx *ent.Tx, entity *types.LocationUpdateRequest) (*ent.Location, error) {
	current, err := tx.Location.Get(ctx, entity.ID)
	if err != nil {
		return nil, err
	}

	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.Location.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetNillableLocationCategoryID(entity.LocationCategoryID).
		SetName(entity.Name).
		SetAddressLine1(entity.AddressLine1).
		SetAddressLine2(entity.AddressLine2).
		SetCity(entity.City).
		SetStateID(entity.StateID).
		SetPostalCode(entity.PostalCode).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		r.Logger.Err(err).Msg("Error updating location")
		return nil, err
	}

	return updatedEntity, nil
}
