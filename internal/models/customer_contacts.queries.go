package models

import (
	"context"

	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/ent/customercontact"
	"github.com/google/uuid"
)

// SyncCustomerContacts synchronizes customer contacts.
//
// Parameters:
//
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerUpdateRequest: The customer update request containing the contact details.
//   - updatedEntity *ent.Customer: The updated Customer entity.
//
// Returns:
//   - error: An error object that indicates why the synchronization failed, nil if no error occurred.
func (r *QueryService) SyncCustomerContacts(
	ctx context.Context, tx *ent.Tx, entity *types.CustomerUpdateRequest, updatedEntity *ent.Customer,
) error {
	existingComments, err := tx.Customer.QueryContacts(updatedEntity).Where(
		customercontact.HasCustomerWith(customer.IDEQ(entity.ID)),
	).All(ctx)
	if err != nil {
		return err
	}

	// Delete unmatched contacts
	if err = r.deleteUnmatchedCustomerContacts(ctx, tx, entity, existingComments); err != nil {
		return err
	}

	// Update or create new contacts
	return r.updateOrCreateCustomerContacts(ctx, tx, entity)
}

// deleteUnmatchedCustomerContacts deletes customer contacts that are not present in the update request.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerUpdateRequest: The customer update request containing the contact details.
//   - existingContacts []*ent.CustomerContact: A slice of existing customer contacts.
//
// Returns:
//   - error: An error object that indicates why the deletion failed, nil if no error occurred.
func (r *QueryService) deleteUnmatchedCustomerContacts(
	ctx context.Context, tx *ent.Tx, entity *types.CustomerUpdateRequest, existingContacts []*ent.CustomerContact,
) error {
	contactPresent := make(map[uuid.UUID]bool)
	for _, contact := range entity.Contacts {
		if contact.ID != uuid.Nil {
			contactPresent[contact.ID] = true
		}
	}

	for _, existingContact := range existingContacts {
		if !contactPresent[existingContact.ID] {
			if err := tx.CustomerContact.DeleteOneID(existingContact.ID).Exec(ctx); err != nil {
				r.Logger.Err(err).Msg("Error deleting customer contact")
				return err
			}
		}
	}

	return nil
}

// CreateCustomerContacts creates customer contacts in bulk.
//
// Parameters:
//
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - customerID uuid.UUID: The identifier of the customer to associate the contacts with.
//   - entity *CustomerRequest: The customer request containing the contact details.
//
// Returns:
//   - error: An error object that indicates why the creation failed, nil if no error occurred.
func (r *QueryService) CreateCustomerContacts(ctx context.Context, tx *ent.Tx, customerID uuid.UUID, entity *types.CustomerRequest) error {
	builders := make([]*ent.CustomerContactCreate, 0, len(entity.Contacts))

	for _, contact := range entity.Contacts {
		builder := tx.CustomerContact.Create().
			SetBusinessUnitID(entity.BusinessUnitID).
			SetOrganizationID(entity.OrganizationID).
			SetCustomerID(customerID).
			SetName(contact.Name).
			SetEmail(contact.Email).
			SetTitle(contact.Title).
			SetPhoneNumber(contact.PhoneNumber).
			SetIsPayableContact(contact.IsPayableContact)
		builders = append(builders, builder)
	}

	err := tx.CustomerContact.CreateBulk(builders...).Exec(ctx)
	if err != nil {
		r.Logger.Err(err).Msg("Error creating customer contacts")
		return err
	}

	return nil
}

// updateOrCreateCustomerContacts updates existing customer contacts or creates new ones.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerUpdateRequest: The customer update request containing the contact details.
//
// Returns:
//   - error: An error object that indicates why the update or creation failed, nil if no error occurred.
func (r *QueryService) updateOrCreateCustomerContacts(ctx context.Context, tx *ent.Tx, entity *types.CustomerUpdateRequest) error {
	// Builders for new contacts
	newContactBuilders := make([]*ent.CustomerContactCreate, 0, len(entity.Contacts))

	// Update existing contacts
	for _, contact := range entity.Contacts {
		if contact.ID != uuid.Nil {
			err := tx.CustomerContact.UpdateOneID(contact.ID).
				SetName(contact.Name).
				SetEmail(contact.Email).
				SetTitle(contact.Title).
				SetPhoneNumber(contact.PhoneNumber).
				SetIsPayableContact(contact.IsPayableContact).
				Exec(ctx)
			if err != nil {
				r.Logger.Err(err).Msg("Error updating customer contact")
				return err
			}
		} else {
			builder := tx.CustomerContact.Create().
				SetBusinessUnitID(entity.BusinessUnitID).
				SetOrganizationID(entity.OrganizationID).
				SetCustomerID(entity.ID).
				SetName(contact.Name).
				SetEmail(contact.Email).
				SetTitle(contact.Title).
				SetPhoneNumber(contact.PhoneNumber).
				SetIsPayableContact(contact.IsPayableContact)
			newContactBuilders = append(newContactBuilders, builder)
		}
	}

	// Create new contacts in bulk
	if len(newContactBuilders) > 0 {
		err := tx.CustomerContact.CreateBulk(newContactBuilders...).Exec(ctx)
		if err != nil {
			r.Logger.Err(err).Msg("Error creating customer contacts in bulk")
			return err
		}
	}

	return nil
}
