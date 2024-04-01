package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/accountingcontrol"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// AccountingControlOps is the service for accounting control settings.
type AccountingControlOps struct {
	client *ent.Client
}

// NewAccountingControlOps creates a new accounting control service.
func NewAccountingControlOps() *AccountingControlOps {
	return &AccountingControlOps{
		client: database.GetClient(),
	}
}

// GetAccountingControl gets the accounting control settings for an organization.
func (r *AccountingControlOps) GetAccountingControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.AccountingControl, error) {
	accountingControl, err := r.client.AccountingControl.Query().Where(
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
	updatedAC, err := r.client.AccountingControl.
		UpdateOneID(ac.ID).
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
		SetNillableDefaultExpAccountID(ac.DefaultExpAccountID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedAC, nil
}
