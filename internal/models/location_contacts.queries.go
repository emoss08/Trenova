package models

import (
	"context"

	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/location"
	"github.com/emoss08/trenova/internal/ent/locationcontact"
	"github.com/google/uuid"
)

func (r *QueryService) CreateLocationContacts(ctx context.Context, tx *ent.Tx, locationID uuid.UUID, entity *types.LocationRequest) error {
	for _, contact := range entity.Contacts {
		if err := tx.LocationContact.Create().
			SetLocationID(locationID).
			SetBusinessUnitID(entity.BusinessUnitID).
			SetOrganizationID(entity.OrganizationID).
			SetName(contact.Name).
			SetEmailAddress(contact.EmailAddress).
			SetPhoneNumber(contact.PhoneNumber).
			Exec(ctx); err != nil {
			r.Logger.Err(err).Msg("Error creating location contact")
			return err
		}
	}

	return nil
}

// SyncLocationContacts synchronizes location contacts.
//
// Parameters:
//
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *LocationUpdateRequest: The location update request containing the contact details.
//   - updatedEntity *ent.Location: The updated Location entity.
//
// Returns:
//   - error: An error object that indicates why the synchronization failed, nil if no error occurred.
func (r *QueryService) SyncLocationContacts(
	ctx context.Context, tx *ent.Tx, entity *types.LocationUpdateRequest, updatedEntity *ent.Location,
) error {
	existingContacts, err := tx.Location.QueryContacts(updatedEntity).Where(
		locationcontact.HasLocationWith(location.IDEQ(entity.ID)),
	).All(ctx)
	if err != nil {
		return err
	}

	// Delete unmatched contacts
	if err = r.deleteUnmatchedLocationContacts(ctx, tx, entity, existingContacts); err != nil {
		return err
	}

	// Update or create new contacts
	return r.updateOrCreateLocationContacts(ctx, tx, entity)
}

// deleteUnmatchedLocationContacts deletes locations contacts that are not present in the update request.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *LocationUpdateRequest: The location update request containing the contact details.
//   - existingContacts []*ent.LocationContact: A slice of existing location contacts.
//
// Returns:
//   - error: An error object that indicates why the deletion failed, nil if no error occurred.
func (r *QueryService) deleteUnmatchedLocationContacts(
	ctx context.Context, tx *ent.Tx, entity *types.LocationUpdateRequest, existingContacts []*ent.LocationContact,
) error {
	contactPresent := make(map[uuid.UUID]bool)
	for _, contact := range entity.Contacts {
		if contact.ID != uuid.Nil {
			contactPresent[contact.ID] = true
		}
	}

	for _, existingContact := range existingContacts {
		if !contactPresent[existingContact.ID] {
			if err := tx.LocationComment.DeleteOneID(existingContact.ID).Exec(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// updateOrCreateLocationContacts updates existing location contacts or creates new ones.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *LocationUpdateRequest: The location update request containing the contact details.
//
// Returns:
//   - error: An error object that indicates why the update or creation failed, nil if no error occurred.
func (r *QueryService) updateOrCreateLocationContacts(ctx context.Context, tx *ent.Tx, entity *types.LocationUpdateRequest) error {
	for _, contact := range entity.Contacts {
		if contact.ID != uuid.Nil {
			if _, err := tx.LocationContact.UpdateOneID(contact.ID).
				SetName(contact.Name).
				SetEmailAddress(contact.EmailAddress).
				SetPhoneNumber(contact.PhoneNumber).
				Save(ctx); err != nil {
				return err
			}
		} else {
			if _, err := tx.LocationContact.Create().
				SetLocationID(entity.ID).
				SetBusinessUnitID(entity.BusinessUnitID).
				SetOrganizationID(entity.OrganizationID).
				SetName(contact.Name).
				SetEmailAddress(contact.EmailAddress).
				SetPhoneNumber(contact.PhoneNumber).
				Save(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}
