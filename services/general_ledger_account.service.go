package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/generalledgeraccount"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// GeneralLedgerAccountOps is the service for general ledger account.
type GeneralLedgerAccountOps struct {
	client *ent.Client
}

type GeneralLedgerAccountRequest struct {
	BusinessUnitID uuid.UUID                        `json:"businessUnitId"`
	OrganizationID uuid.UUID                        `json:"organizationId"`
	Status         generalledgeraccount.Status      `json:"status" validate:"required,oneof=A I"`
	AccountNumber  string                           `json:"accountNumber" validate:"required,max=7"`
	AccountType    generalledgeraccount.AccountType `json:"accountType" validate:"required"`
	CashFlowType   string                           `json:"cashFlowType" validate:"omitempty"`
	AccountSubType string                           `json:"accountSubType" validate:"omitempty"`
	AccountClass   string                           `json:"accountClass" validate:"omitempty"`
	Balance        float64                          `json:"balance" validate:"omitempty"`
	InterestRate   float64                          `json:"interestRate" validate:"omitempty"`
	DateOpened     *pgtype.Date                     `json:"dateOpened" validate:"omitempty"`
	DateClosed     *pgtype.Date                     `json:"dateClosed" validate:"omitempty"`
	Notes          string                           `json:"notes,omitempty"`
	IsTaxRelevant  bool                             `json:"isTaxRelevant" validate:"omitempty"`
	IsReconciled   bool                             `json:"isReconciled" validate:"omitempty"`
	TagIDs         []uuid.UUID                      `json:"tagIds,omitempty"`
}

type GeneralLedgerAccountUpdateRequest struct {
	ID uuid.UUID `json:"id,omitempty"`
	GeneralLedgerAccountRequest
}

// NewGeneralLedgerAccountOps creates a new general ledger account service.
func NewGeneralLedgerAccountOps() *GeneralLedgerAccountOps {
	return &GeneralLedgerAccountOps{
		client: database.GetClient(),
	}
}

// GetGeneralLedgerAccounts gets the general ledger accounts for an organization.
func (r *GeneralLedgerAccountOps) GetGeneralLedgerAccounts(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.GeneralLedgerAccount, int, error) {
	glAccountCount, countErr := r.client.GeneralLedgerAccount.Query().
		Where(
			generalledgeraccount.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	glAccounts, err := r.client.GeneralLedgerAccount.Query().
		Limit(limit).
		WithTags().
		Offset(offset).
		Where(
			generalledgeraccount.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return glAccounts, glAccountCount, nil
}

// CreateGeneralLedgerAccount creates a new general ledger account for an organization.
func (r *GeneralLedgerAccountOps) CreateGeneralLedgerAccount(ctx context.Context, newGLAccount GeneralLedgerAccountRequest) (*ent.GeneralLedgerAccount, error) {
	glAccount, err := r.client.GeneralLedgerAccount.Create().
		SetOrganizationID(newGLAccount.OrganizationID).
		SetBusinessUnitID(newGLAccount.BusinessUnitID).
		SetStatus(newGLAccount.Status).
		SetAccountNumber(newGLAccount.AccountNumber).
		SetAccountType(newGLAccount.AccountType).
		SetCashFlowType(newGLAccount.CashFlowType).
		SetAccountSubType(newGLAccount.AccountSubType).
		SetAccountClass(newGLAccount.AccountClass).
		SetBalance(newGLAccount.Balance).
		SetInterestRate(newGLAccount.InterestRate).
		SetNotes(newGLAccount.Notes).
		SetIsTaxRelevant(newGLAccount.IsTaxRelevant).
		SetIsReconciled(newGLAccount.IsReconciled).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// If the tags are provided, add them to the general ledger account
	if len(newGLAccount.TagIDs) > 0 {
		_, updateErr := glAccount.Update().
			AddTagIDs(newGLAccount.TagIDs...).
			Save(ctx)
		if updateErr != nil {
			return nil, updateErr
		}
	}

	return glAccount, nil
}

// UpdateGeneralLedgerAccount updates a general ledger account.
func (r *GeneralLedgerAccountOps) UpdateGeneralLedgerAccount(ctx context.Context, glAccount GeneralLedgerAccountUpdateRequest) (*ent.GeneralLedgerAccount, error) {
	// Start building the update operation
	updateOp := r.client.GeneralLedgerAccount.UpdateOneID(glAccount.ID).
		SetStatus(glAccount.Status).
		SetAccountNumber(glAccount.AccountNumber).
		SetAccountType(glAccount.AccountType).
		SetCashFlowType(glAccount.CashFlowType).
		SetAccountSubType(glAccount.AccountSubType).
		SetAccountClass(glAccount.AccountClass).
		SetBalance(glAccount.Balance).
		SetInterestRate(glAccount.InterestRate).
		SetNotes(glAccount.Notes).
		SetIsTaxRelevant(glAccount.IsTaxRelevant).
		SetIsReconciled(glAccount.IsReconciled)

	// If the tags are provided, add them to the general ledger account
	if len(glAccount.TagIDs) > 0 {
		updateOp = updateOp.ClearTags().
			AddTagIDs(glAccount.TagIDs...)
	}

	// If the tags are not provided, clear the tags
	if len(glAccount.TagIDs) == 0 {
		updateOp = updateOp.ClearTags()
	}

	// Execute the update operation
	updatedGLAccount, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedGLAccount, nil
}
