package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/generalledgeraccount"
	"github.com/emoss08/trenova/ent/organization"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"
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

// GeneralLedgerAccountOps is the service for general ledger account.
type GeneralLedgerAccountOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewGeneralLedgerAccountOps creates a new general ledger account service.
func NewGeneralLedgerAccountOps() *GeneralLedgerAccountOps {
	return &GeneralLedgerAccountOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetGeneralLedgerAccounts gets the general ledger accounts for an organization.
func (r *GeneralLedgerAccountOps) GetGeneralLedgerAccounts(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.GeneralLedgerAccount, int, error) {
	entityCount, countErr := r.client.GeneralLedgerAccount.Query().
		Where(
			generalledgeraccount.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.GeneralLedgerAccount.Query().
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
func (r *GeneralLedgerAccountOps) CreateGeneralLedgerAccount(
	ctx context.Context, newEntity GeneralLedgerAccountRequest,
) (*ent.GeneralLedgerAccount, error) {
	// Begin a new transaction
	tx, err := r.client.Tx(ctx)
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to start transaction")
		r.logger.WithField("error", wrappedErr).Error("failed to start transaction")
		return nil, wrappedErr
	}

	// Ensure the transaction is either committed or rolled back
	defer func() {
		if v := recover(); v != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(rollbackErr, "failed to rollback transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
			panic(v)
		}
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(err, "failed to rollback transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				wrappedErr := eris.Wrap(err, "failed to commit transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to commit transaction")
			}
		}
	}()

	createdEntity, err := tx.GeneralLedgerAccount.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetStatus(newEntity.Status).
		SetAccountNumber(newEntity.AccountNumber).
		SetAccountType(newEntity.AccountType).
		SetCashFlowType(newEntity.CashFlowType).
		SetAccountSubType(newEntity.AccountSubType).
		SetAccountClass(newEntity.AccountClass).
		SetBalance(newEntity.Balance).
		SetInterestRate(newEntity.InterestRate).
		SetNotes(newEntity.Notes).
		SetIsTaxRelevant(newEntity.IsTaxRelevant).
		SetIsReconciled(newEntity.IsReconciled).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// If the tags are provided, add them to the general ledger account
	if len(newEntity.TagIDs) > 0 {
		updateErr := createdEntity.Update().
			AddTagIDs(newEntity.TagIDs...).
			SaveX(ctx)
		if updateErr != nil {
			return nil, eris.Wrap(err, "failed to create entity")
		}
	}

	return createdEntity, nil
}

// UpdateGeneralLedgerAccount updates a general ledger account.
func (r *GeneralLedgerAccountOps) UpdateGeneralLedgerAccount(
	ctx context.Context, entity GeneralLedgerAccountUpdateRequest,
) (*ent.GeneralLedgerAccount, error) {
	// Begin a new transaction
	tx, err := r.client.Tx(ctx)
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to start transaction")
		r.logger.WithField("error", wrappedErr).Error("failed to start transaction")
		return nil, wrappedErr
	}

	// Ensure the transaction is either committed or rolled back
	defer func() {
		if v := recover(); v != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(rollbackErr, "failed to rollback transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
			panic(v)
		}
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(err, "failed to rollback transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				wrappedErr := eris.Wrap(err, "failed to commit transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to commit transaction")
			}
		}
	}()

	current, err := tx.GeneralLedgerAccount.Get(ctx, entity.ID) // Get the current entity.
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to retrieve requested entity")
		r.logger.WithField("error", wrappedErr).Error("failed to retrieve requested entity")
		return nil, wrappedErr
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, tools.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"accountNumber")
	}

	// Start building the update operation
	updateOp := tx.GeneralLedgerAccount.UpdateOneID(entity.ID).
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
