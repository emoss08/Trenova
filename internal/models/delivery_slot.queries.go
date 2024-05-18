package models

import (
	"context"

	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/ent/deliveryslot"
	"github.com/google/uuid"
)

// SyncDeliverySlots synchronizes delivery slots.
//
// Parameters:
//
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerUpdateRequest: The customer update request containing the delivery slot details.
//   - updatedEntity *ent.Customer: The updated Customer entity.
//
// Returns:
//   - error: An error object that indicates why the synchronization failed, nil if no error occurred.
func (r *QueryService) SyncDeliverySlots(
	ctx context.Context, tx *ent.Tx, entity *types.CustomerUpdateRequest, updatedEntity *ent.Customer,
) error {
	existingSlots, err := tx.Customer.QueryDeliverySlots(updatedEntity).Where(
		deliveryslot.HasCustomerWith(customer.IDEQ(entity.ID)),
	).All(ctx)
	if err != nil {
		r.Logger.Err(err).Msg("Error querying existing delivery slots")
		return err
	}

	// Delete unmatched delivery slots
	if err = r.deleteUnmatchedDeliverySlots(ctx, tx, entity, existingSlots); err != nil {
		return err
	}

	// Update or create new delivery slots
	return r.updateOrCreateDeliverySlots(ctx, tx, entity)
}

// CreateDeliverySlots creates delivery slots for a customer.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - customerID uuid.UUID: The identifier of the customer to associate the delivery slots with.
//   - entity *CustomerRequest: The customer request containing the delivery slot details.
//
// Returns:
//   - error: An error object that indicates why the creation failed, nil if no error occurred.
func (r *QueryService) CreateDeliverySlots(
	ctx context.Context, tx *ent.Tx, customerID uuid.UUID, entity *types.CustomerRequest,
) error {
	for _, slot := range entity.DeliverySlots {
		err := tx.DeliverySlot.Create().
			SetCustomerID(customerID).
			SetBusinessUnitID(entity.BusinessUnitID).
			SetOrganizationID(entity.OrganizationID).
			SetDayOfWeek(slot.DayOfWeek).
			SetStartTime(slot.StartTime).
			SetEndTime(slot.EndTime).
			Exec(ctx)
		if err != nil {
			r.Logger.Err(err).Msg("Error creating delivery slot")
			return err
		}
	}

	return nil
}

// updateOrCreateDeliverySlots updates existing delivery slots or creates new ones.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerUpdateRequest: The customer update request containing the delivery slot details.
//
// Returns:
//   - error: An error object that indicates why the update or creation failed, nil if no error occurred.
func (r *QueryService) updateOrCreateDeliverySlots(ctx context.Context, tx *ent.Tx, entity *types.CustomerUpdateRequest) error {
	for _, slot := range entity.DeliverySlots {
		if slot.ID != uuid.Nil {
			if err := tx.DeliverySlot.UpdateOneID(slot.ID).
				SetDayOfWeek(slot.DayOfWeek).
				SetStartTime(slot.StartTime).
				SetEndTime(slot.EndTime).
				Exec(ctx); err != nil {
				r.Logger.Err(err).Msg("Error updating delivery slot")
				return err
			}
		} else {
			if err := tx.DeliverySlot.Create().
				SetBusinessUnitID(entity.BusinessUnitID).
				SetOrganizationID(entity.OrganizationID).
				SetCustomerID(entity.ID).
				SetLocationID(slot.LocationID).
				SetDayOfWeek(slot.DayOfWeek).
				SetStartTime(slot.StartTime).
				SetEndTime(slot.EndTime).
				Exec(ctx); err != nil {
				r.Logger.Err(err).Msg("Error creating delivery slot")
				return err
			}
		}
	}

	return nil
}

// deleteUnmatchedDeliverySlots deletes delivery slots that are not present in the update request.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerUpdateRequest: The customer update request containing the contact details.
//   - existingContacts []*ent.CustomerContact: A slice of existing customer contacts.
//
// Returns:
//   - error: An error object that indicates why the deletion failed, nil if no error occurred.
func (r *QueryService) deleteUnmatchedDeliverySlots(
	ctx context.Context, tx *ent.Tx, entity *types.CustomerUpdateRequest, existingSlots []*ent.DeliverySlot,
) error {
	slotPresent := make(map[uuid.UUID]bool)
	for _, slot := range entity.DeliverySlots {
		if slot.ID != uuid.Nil {
			slotPresent[slot.ID] = true
		}
	}

	for _, existingSlot := range existingSlots {
		if !slotPresent[existingSlot.ID] {
			if err := tx.DeliverySlot.DeleteOneID(existingSlot.ID).Exec(ctx); err != nil {
				r.Logger.Err(err).Msg("Error deleting customer contact")
				return err
			}
		}
	}

	return nil
}
