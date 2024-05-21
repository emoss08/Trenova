package queries

import (
	"context"

	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
)

// UpdateCustomerEmailProfileEntity updates a customer email profile entity.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerUpdateRequest: The customer update request containing the email profile details.
//
// Returns:
//   - error: An error object that indicates why the update failed, nil if no error occurred.
func (r *QueryService) UpdateCustomerEmailProfileEntity(ctx context.Context, tx *ent.Tx, entity *types.CustomerUpdateRequest) error {
	current, err := tx.CustomerEmailProfile.Get(ctx, entity.EmailProfile.ID)
	if err != nil {
		r.Logger.Err(err).Msg("QueryService: Error getting customer email profile")
		return err
	}

	// Check if the version matches.
	if current.Version != entity.EmailProfile.Version {
		return util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"emailProfile")
	}

	updateOp := tx.CustomerEmailProfile.UpdateOneID(entity.EmailProfile.ID).
		SetOrganizationID(entity.OrganizationID).
		SetVersion(entity.EmailProfile.Version + 1). // Increment the version
		SetSubject(entity.EmailProfile.Subject).
		SetNillableEmailProfileID(entity.EmailProfile.EmailProfileID).
		SetEmailRecipients(entity.EmailProfile.EmailRecipients).
		SetEmailCcRecipients(entity.EmailProfile.EmailCcRecipients).
		SetAttachmentName(entity.EmailProfile.AttachmentName).
		SetEmailFormat(entity.EmailProfile.EmailFormat)

	return updateOp.Exec(ctx)
}

// CreateCustomerEmailProfileEntity creates a customer email profile entity. It returns an error object.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *CustomerRequest: The customer request containing the details of the email profile to be created.
//
// Returns:
//   - error: An error object that indicates why the creation failed, nil if no error occurred.
func (r *QueryService) CreateCustomerEmailProfileEntity(ctx context.Context, tx *ent.Tx, customerID uuid.UUID, entity *types.CustomerRequest) error {
	createdEntity := tx.CustomerEmailProfile.Create().
		SetCustomerID(customerID).
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetSubject(entity.EmailProfile.Subject).
		SetNillableEmailProfileID(entity.EmailProfile.EmailProfileID).
		SetEmailRecipients(entity.EmailProfile.EmailRecipients).
		SetEmailCcRecipients(entity.EmailProfile.EmailCcRecipients).
		SetAttachmentName(entity.EmailProfile.AttachmentName).
		SetEmailFormat(entity.EmailProfile.EmailFormat)

	return createdEntity.Exec(ctx)
}
