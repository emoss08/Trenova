package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/divisioncode"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/logger"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"
)

type DivisionCodeOps struct {
	logger *logrus.Logger
	client *ent.Client
}

// NewDivisionCodeOps creates a new division code service.
func NewDivisionCodeOps() *DivisionCodeOps {
	return &DivisionCodeOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetDivisionCodes gets the division codes for an organization.
func (r *DivisionCodeOps) GetDivisionCodes(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.DivisionCode, int, error) {
	entityCount, countErr := r.client.DivisionCode.Query().Where(
		divisioncode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.DivisionCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			divisioncode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateDivisionCode creates a new division code.
func (r *DivisionCodeOps) CreateDivisionCode(ctx context.Context, newEntity ent.DivisionCode) (*ent.DivisionCode, error) {
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

	createdEntity, err := tx.DivisionCode.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetStatus(newEntity.Status).
		SetCode(newEntity.Code).
		SetDescription(newEntity.Description).
		SetNillableApAccountID(newEntity.ApAccountID).
		SetNillableCashAccountID(newEntity.CashAccountID).
		SetNillableExpenseAccountID(newEntity.ExpenseAccountID).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create entity")
	}

	return createdEntity, nil
}

// UpdateDivisionCode updates a division code.
func (r *DivisionCodeOps) UpdateDivisionCode(ctx context.Context, entity ent.DivisionCode) (*ent.DivisionCode, error) {
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

	current, err := tx.DivisionCode.Get(ctx, entity.ID)
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to retrieve requested entity")
		r.logger.WithField("error", wrappedErr).Error("failed to retrieve requested entity")
		return nil, wrappedErr
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, tools.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.DivisionCode.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetNillableApAccountID(entity.ApAccountID).
		SetNillableCashAccountID(entity.CashAccountID).
		SetNillableExpenseAccountID(entity.ExpenseAccountID).
		SetVersion(entity.Version + 1) // Increment the version

	// If the ap account ID is nil, clear the association
	if entity.ApAccountID == nil {
		updateOp = updateOp.ClearApAccount()
	}

	// If the cash account ID is nil, clear the association
	if entity.CashAccountID == nil {
		updateOp = updateOp.ClearCashAccount()
	}

	// If the expense account ID is nil, clear the association
	if entity.ExpenseAccountID == nil {
		updateOp = updateOp.ClearExpenseAccount()
	}

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
