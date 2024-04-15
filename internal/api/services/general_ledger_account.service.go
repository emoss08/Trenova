package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/generalledgeraccount"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
)

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
	Version        int                              `json:"version" validate:"omitempty"`
	TagIDs         []uuid.UUID                      `json:"tagIds,omitempty"`
}

type GeneralLedgerAccountUpdateRequest struct {
	ID uuid.UUID `json:"id,omitempty"`
	GeneralLedgerAccountRequest
}

type GeneralLedgerAccountService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewGeneralLedgerAccountService creates a new general ledger account service.
func NewGeneralLedgerAccountService(s *api.Server) *GeneralLedgerAccountService {
	return &GeneralLedgerAccountService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetGeneralLedgerAccounts gets the general ledger accounts for an organization.
func (r *GeneralLedgerAccountService) GetGeneralLedgerAccounts(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.GeneralLedgerAccount, int, error) {
	entityCount, countErr := r.Client.GeneralLedgerAccount.Query().
		Where(
			generalledgeraccount.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.GeneralLedgerAccount.Query().
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

	return entities, entityCount, nil
}

// CreateGeneralLedgerAccount creates a new general ledger account for an organization.
func (r *GeneralLedgerAccountService) CreateGeneralLedgerAccount(
	ctx context.Context, entity *GeneralLedgerAccountRequest,
) (*ent.GeneralLedgerAccount, error) {
	newEntity := new(ent.GeneralLedgerAccount)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createGeneralLedgerAccountEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return newEntity, nil
}

func (r *GeneralLedgerAccountService) createGeneralLedgerAccountEntity(
	ctx context.Context, tx *ent.Tx, entity *GeneralLedgerAccountRequest,
) (*ent.GeneralLedgerAccount, error) {
	createdEntity, err := tx.GeneralLedgerAccount.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetAccountNumber(entity.AccountNumber).
		SetAccountType(entity.AccountType).
		SetCashFlowType(entity.CashFlowType).
		SetAccountSubType(entity.AccountSubType).
		SetAccountClass(entity.AccountClass).
		SetBalance(entity.Balance).
		SetInterestRate(entity.InterestRate).
		SetNotes(entity.Notes).
		SetIsTaxRelevant(entity.IsTaxRelevant).
		SetIsReconciled(entity.IsReconciled).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// If the tags are provided, add them to the general ledger account
	if len(entity.TagIDs) > 0 {
		updateErr := createdEntity.Update().
			AddTagIDs(entity.TagIDs...).
			SaveX(ctx)
		if updateErr != nil {
			return nil, eris.Wrap(err, "failed to create entity")
		}
	}

	return createdEntity, nil
}

// UpdateGeneralLedgerAccount updates a general ledger account.
func (r *GeneralLedgerAccountService) UpdateGeneralLedgerAccount(
	ctx context.Context, entity *GeneralLedgerAccountUpdateRequest,
) (*ent.GeneralLedgerAccount, error) {
	updatedEntity := new(ent.GeneralLedgerAccount)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateGeneralLedgerAccountEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *GeneralLedgerAccountService) updateGeneralLedgerAccountEntity(
	ctx context.Context, tx *ent.Tx, entity *GeneralLedgerAccountUpdateRequest,
) (*ent.GeneralLedgerAccount, error) {
	current, err := tx.GeneralLedgerAccount.Get(ctx, entity.ID) // Get the current entity.
	if err != nil {
		return nil, eris.Wrap(err, "failed to retrieve requested entity")
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"accountNumber")
	}

	// Start building the update operation
	updateOp := tx.GeneralLedgerAccount.
		UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetAccountNumber(entity.AccountNumber).
		SetAccountType(entity.AccountType).
		SetCashFlowType(entity.CashFlowType).
		SetAccountSubType(entity.AccountSubType).
		SetAccountClass(entity.AccountClass).
		SetBalance(entity.Balance).
		SetInterestRate(entity.InterestRate).
		SetNotes(entity.Notes).
		SetIsTaxRelevant(entity.IsTaxRelevant).
		SetIsReconciled(entity.IsReconciled).
		SetVersion(entity.Version + 1) // Increment the version

	// If the tags are provided, add them to the general ledger account
	if len(entity.TagIDs) > 0 {
		updateOp = updateOp.ClearTags().
			AddTagIDs(entity.TagIDs...)
	}

	// If the tags are not provided, clear the tags
	if len(entity.TagIDs) == 0 {
		updateOp = updateOp.ClearTags()
	}

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
