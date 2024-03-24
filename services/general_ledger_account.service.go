package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/generalledgeraccount"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// GeneralLedgerAccountOps is the service for general ledger account.
type GeneralLedgerAccountOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewGeneralLedgerAccountOps creates a new general ledger account service.
func NewGeneralLedgerAccountOps(ctx context.Context) *GeneralLedgerAccountOps {
	return &GeneralLedgerAccountOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetGeneralLedgerAccounts gets the general ledger accounts for an organization.
func (r *GeneralLedgerAccountOps) GetGeneralLedgerAccounts(limit, offset int, orgID, buID uuid.UUID) ([]*ent.GeneralLedgerAccount, int, error) {
	glAccountCount, countErr := r.client.GeneralLedgerAccount.Query().
		Where(
			generalledgeraccount.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	glAccounts, err := r.client.GeneralLedgerAccount.Query().
		Limit(limit).
		Offset(offset).
		Where(
			generalledgeraccount.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return glAccounts, glAccountCount, nil
}

// CreateGeneralLedgerAccount creates a new general ledger account for an organization.
func (r *GeneralLedgerAccountOps) CreateGeneralLedgerAccount(newGLAccount ent.GeneralLedgerAccount) (*ent.GeneralLedgerAccount, error) {
	glAccount, err := r.client.GeneralLedgerAccount.Create().
		SetOrganizationID(newGLAccount.OrganizationID).
		SetBusinessUnitID(newGLAccount.BusinessUnitID).
		SetStatus(newGLAccount.Status).
		SetAccountNumber(newGLAccount.AccountNumber).
		SetAccountType(newGLAccount.AccountType).
		SetCashFlowType(newGLAccount.CashFlowType).
		SetAccountSubType(newGLAccount.AccountSubType).
		SetAccountClass(newGLAccount.AccountClass).
		SetNillableBalance(newGLAccount.Balance).
		SetNillableInterestRate(newGLAccount.InterestRate).
		SetDateClosed(newGLAccount.DateClosed).
		SetNotes(newGLAccount.Notes).
		SetIsTaxRelevant(newGLAccount.IsTaxRelevant).
		SetIsReconciled(newGLAccount.IsReconciled).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return glAccount, nil
}

// UpdateGeneralLedgerAccount updates a general ledger account.
func (r *GeneralLedgerAccountOps) UpdateGeneralLedgerAccount(glAccount ent.GeneralLedgerAccount) (*ent.GeneralLedgerAccount, error) {
	// Start building the update operation
	updateOp := r.client.GeneralLedgerAccount.UpdateOneID(glAccount.ID).
		SetStatus(glAccount.Status).
		SetAccountNumber(glAccount.AccountNumber).
		SetAccountType(glAccount.AccountType).
		SetCashFlowType(glAccount.CashFlowType).
		SetAccountSubType(glAccount.AccountSubType).
		SetAccountClass(glAccount.AccountClass).
		SetNillableBalance(glAccount.Balance).
		SetNillableInterestRate(glAccount.InterestRate).
		SetDateClosed(glAccount.DateClosed).
		SetNotes(glAccount.Notes).
		SetIsTaxRelevant(glAccount.IsTaxRelevant).
		SetIsReconciled(glAccount.IsReconciled)

	// Execute the update operation
	updatedGLAccount, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedGLAccount, nil
}
