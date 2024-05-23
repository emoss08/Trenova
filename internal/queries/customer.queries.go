package queries

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/ent/deliveryslot"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
)

// GetCustomers retrieves a list of customers for a given organization and business unit.
// It returns a slice of Customer entities, the total number of customer records, and an error object.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - limit int: The maximum number of records to return.
//   - offset int: The number of records to skip before starting to return records.
//   - orgID uuid.UUID: The identifier of the organization.
//   - buID uuid.UUID: The identifier of the business unit.
//
// Returns:
//   - []*ent.Customer: A slice of Customer entities.
//   - int: The total number of customer records.
//   - error: An error object that indicates why the retrieval failed, nil if no error occurred.
func (r *QueryService) GetCustomers(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.Customer, int, error) {
	count, err := r.Client.Customer.Query().Where(
		customer.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)
	if err != nil {
		r.Logger.Err(err).Msg("Error getting customer count")
		return nil, 0, err
	}

	entities, err := r.Client.Customer.Query().
		Limit(limit).
		Offset(offset).
		WithContacts().
		WithDeliverySlots(func(q *ent.DeliverySlotQuery) {
			q.Order(
				ent.Desc(deliveryslot.FieldDayOfWeek),
			)
		}).
		WithDetentionPolicies().
		WithEmailProfile().
		WithRuleProfile(func(q *ent.CustomerRuleProfileQuery) {
			q.WithDocumentClassifications()
		}).
		WithState().
		Where(
			customer.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).
		Order(
			customer.ByName(
				sql.OrderDesc(),
			),
		).All(ctx)
	if err != nil {
		r.Logger.Err(err).Msg("Error getting customers")
		return nil, 0, err
	}

	return entities, count, nil
}

// CreateCustomerEntity creates a customer entity.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerRequest: The customer request containing the details of the customer to be created.
//
// Returns:
//   - *ent.Customer: A pointer to the newly created Customer entity.
//   - error: An error object that indicates why the creation failed, nil if no error occurred.
func (r *QueryService) CreateCustomerEntity(ctx context.Context, tx *ent.Tx, entity *types.CustomerRequest) (*ent.Customer, error) {
	createdEntity, err := tx.Customer.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetAddressLine1(entity.AddressLine1).
		SetAddressLine2(entity.AddressLine2).
		SetCity(entity.City).
		SetStateID(entity.StateID).
		SetPostalCode(entity.PostalCode).
		SetHasCustomerPortal(entity.HasCustomerPortal).
		SetAutoMarkReadyToBill(entity.AutoMarkReadyToBill).
		Save(ctx)
	if err != nil {
		r.Logger.Err(err).Msg("QueryService: Error creating customer entity")
		return nil, err
	}

	return createdEntity, nil
}

// UpdateCustomerEntity updates a customer entity.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerUpdateRequest: The customer update request containing the details of the customer to be updated.
//
// Returns:
//   - *ent.Customer: A pointer to the updated Customer entity.
//   - error: An error object that indicates why the update failed, nil if no error occurred.
func (r *QueryService) UpdateCustomerEntity(ctx context.Context, tx *ent.Tx, entity *types.CustomerUpdateRequest) (*ent.Customer, error) {
	current, err := tx.Customer.Get(ctx, entity.ID)
	if err != nil {
		return nil, err
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.Customer.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetName(entity.Name).
		SetAddressLine1(entity.AddressLine1).
		SetAddressLine2(entity.AddressLine2).
		SetCity(entity.City).
		SetStateID(entity.StateID).
		SetPostalCode(entity.PostalCode).
		SetHasCustomerPortal(entity.HasCustomerPortal).
		SetAutoMarkReadyToBill(entity.AutoMarkReadyToBill).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		r.Logger.Err(err).Msg("QueryService: Error updating customer entity")
		return nil, err
	}

	return updatedEntity, nil
}
