package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/accountingcontrol"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/tools"
	"github.com/google/uuid"
)

// AccountingControlOps is the service for accounting control settings.
type AccountingControlOps struct {
	Client *ent.Client
}

// NewAccountingControlOps creates a new accounting control service.
func NewAccountingControlOps() *AccountingControlOps {
	return &AccountingControlOps{
		Client: database.GetClient(),
	}
}

// GetAccountingControl gets the accounting control settings for an organization.
func (r *AccountingControlOps) GetAccountingControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.AccountingControl, error) {
	accountingControl, err := r.Client.AccountingControl.Query().Where(
		accountingcontrol.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(ctx)
	if err != nil {
		return nil, err
	}

	return accountingControl, nil
}

// UpdateAccountingControl updates the accounting control settings for an organization.
func (r *AccountingControlOps) UpdateAccountingControl(ctx context.Context, ac ent.AccountingControl) (*ent.AccountingControl, error) {
	var updatedEntity *ent.AccountingControl

	err := tools.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateAccountingControl(ctx, tx, ac)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *AccountingControlOps) updateAccountingControl(ctx context.Context, tx *ent.Tx, ac ent.AccountingControl) (*ent.AccountingControl, error) {
	updateOp := tx.AccountingControl.UpdateOneID(ac.ID).
		SetRecThreshold(ac.RecThreshold).
		SetRecThresholdAction(ac.RecThresholdAction).
		SetAutoCreateJournalEntries(ac.AutoCreateJournalEntries).
		SetJournalEntryCriteria(ac.JournalEntryCriteria).
		SetRestrictManualJournalEntries(ac.RestrictManualJournalEntries).
		SetRequireJournalEntryApproval(ac.RequireJournalEntryApproval).
		SetEnableRecNotifications(ac.EnableRecNotifications).
		SetHaltOnPendingRec(ac.HaltOnPendingRec).
		SetNillableCriticalProcesses(ac.CriticalProcesses).
		SetNillableDefaultRevAccountID(ac.DefaultRevAccountID).
		SetNillableDefaultExpAccountID(ac.DefaultExpAccountID)

	if ac.DefaultRevAccountID == nil {
		updateOp = updateOp.ClearDefaultRevAccountID()
	}

	if ac.DefaultExpAccountID == nil {
		updateOp = updateOp.ClearDefaultExpAccountID()
	}

	// Update the accounting control settings
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
