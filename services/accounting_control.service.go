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
	ctx    context.Context
	client *ent.Client
}

// NewAccountingControlOps creates a new accounting control service.
func NewAccountingControlOps(ctx context.Context) *AccountingControlOps {
	return &AccountingControlOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetAccountingControl gets the accounting control settings for an organization.
func (r *AccountingControlOps) GetAccountingControl(orgID, buID uuid.UUID) (*ent.AccountingControl, error) {
	accountingControl, err := r.client.AccountingControl.Query().Where(
		accountingcontrol.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return accountingControl, nil
}

// UpdateAccountingControl updates the accounting control settings for an organization.
func (r *AccountingControlOps) UpdateAccountingControl(ac ent.AccountingControl) (*ent.AccountingControl, error) {
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
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedAC, nil
}
