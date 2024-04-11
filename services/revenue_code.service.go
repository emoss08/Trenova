package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/ent/revenuecode"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/logger"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"
)

// RevenueCodeOps is the service for revenue code.
type RevenueCodeOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewRevenueCodeOps creates a new revenue code service.
func NewRevenueCodeOps() *RevenueCodeOps {
	return &RevenueCodeOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetRevenueCodes gets the revenue codes for an organization.
func (r *RevenueCodeOps) GetRevenueCodes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.RevenueCode, int, error) {
	entityCount, countErr := r.client.RevenueCode.Query().Where(
		revenuecode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.RevenueCode.Query().
		Limit(limit).
		Offset(offset).
		WithExpenseAccount().
		WithRevenueAccount().
		Where(
			revenuecode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateRevenueCode creates a new revenue code.
func (r *RevenueCodeOps) CreateRevenueCode(ctx context.Context, newEntity ent.RevenueCode) (*ent.RevenueCode, error) {
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

	createdEntity, err := tx.RevenueCode.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetStatus(newEntity.Status).
		SetCode(newEntity.Code).
		SetDescription(newEntity.Description).
		SetNillableExpenseAccountID(newEntity.ExpenseAccountID).
		SetNillableRevenueAccountID(newEntity.RevenueAccountID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateRevenueCode updates a revenue code.
func (r *RevenueCodeOps) UpdateRevenueCode(
	ctx context.Context, entity ent.RevenueCode,
) (*ent.RevenueCode, error) {
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

	current, err := tx.RevenueCode.Get(ctx, entity.ID) // Get the current entity.
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
	updateOp := tx.RevenueCode.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetNillableExpenseAccountID(entity.ExpenseAccountID).
		SetNillableRevenueAccountID(entity.RevenueAccountID).
		SetVersion(entity.Version + 1) // Increment the version

	// If the expense account ID is nil, clear the association
	if entity.ExpenseAccountID == nil {
		updateOp = updateOp.ClearExpenseAccount()
	}

	// If the revenue account ID is nil, clear the association
	if entity.RevenueAccountID == nil {
		updateOp = updateOp.ClearRevenueAccount()
	}

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
