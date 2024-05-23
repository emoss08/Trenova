package queries

import (
	"context"

	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
)

// UpdateCustomerRuleProfileEntity updates a customer rule profile entity.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerUpdateRequest: The customer update request containing the rule profile details.
//
// Returns:
//   - error: An error object that indicates why the update failed, nil if no error occurred.
func (r *QueryService) UpdateCustomerRuleProfileEntity(ctx context.Context, tx *ent.Tx, entity *types.CustomerUpdateRequest) error {
	current, err := tx.CustomerRuleProfile.Get(ctx, entity.RuleProfile.ID) // Get the current entity.
	if err != nil {
		r.Logger.Err(err).Msg("QueryService: Error getting customer rule profile")
		return err
	}

	// Check if the version matches.
	if current.Version != entity.RuleProfile.Version {
		return util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"billingCycle")
	}

	updateOp := tx.CustomerRuleProfile.UpdateOneID(entity.RuleProfile.ID).
		SetOrganizationID(entity.OrganizationID).
		SetVersion(entity.RuleProfile.Version + 1). // Increment the version
		SetBillingCycle(entity.RuleProfile.BillingCycle)

	if len(entity.RuleProfile.DocClassIDs) > 0 {
		updateOp = updateOp.ClearDocumentClassifications().
			AddDocumentClassificationIDs(entity.RuleProfile.DocClassIDs...)
	}

	// if the document classifications are not provided, clear them
	if len(entity.RuleProfile.DocClassIDs) == 0 {
		updateOp = updateOp.ClearDocumentClassifications()
	}

	return updateOp.Exec(ctx)
}

// CreateCustomerRuleProfileEntity creates a customer rule profile entity. It returns an error object.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerRequest: The customer request containing the details of the rule profile to be created.
//
// Returns:
//   - error: An error object that indicates why the creation failed, nil if no error occurred.
func (r *QueryService) CreateCustomerRuleProfileEntity(ctx context.Context, tx *ent.Tx, customerID uuid.UUID, entity *types.CustomerRequest) error {
	createdEntity := tx.CustomerRuleProfile.Create().
		SetCustomerID(customerID).
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetBillingCycle(entity.RuleProfile.BillingCycle)

	// If the document classifications are provided, add them to the customer rule profile
	if len(entity.RuleProfile.DocClassIDs) > 0 {
		err := createdEntity.
			AddDocumentClassificationIDs(entity.RuleProfile.DocClassIDs...).
			Exec(ctx)
		if err != nil {
			r.Logger.Err(err).Msg("QueryService: Error creating customer rule profile")
			return err
		}
	}

	return nil
}
