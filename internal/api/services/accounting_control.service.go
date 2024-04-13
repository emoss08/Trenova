package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/accountingcontrol"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AccountingControlService is the service for accounting control settings.
type AccountingControlService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewAccountingControlService creates a new accessorial charge service.
func NewAccountingControlService(s *api.Server) *AccountingControlService {
	return &AccountingControlService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetAccountingControl gets the accounting control settings for an organization.
func (r *AccountingControlService) GetAccountingControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.AccountingControl, error) {
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
func (r *AccountingControlService) UpdateAccountingControl(ctx context.Context, ac *ent.AccountingControl) (*ent.AccountingControl, error) {
	updatedEntity := new(ent.AccountingControl)

	if ac == nil {
		return nil, errors.New("accounting control is nil")
	}

	if r.Client == nil {
		return nil, errors.New("client is nil")
	}

	if r.Logger == nil {
		return nil, errors.New("logger is nil")
	}

	fmt.Printf("Accounting Control %+v", ac)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
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

func (r *AccountingControlService) updateAccountingControl(ctx context.Context, tx *ent.Tx, ac *ent.AccountingControl) (*ent.AccountingControl, error) {
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
