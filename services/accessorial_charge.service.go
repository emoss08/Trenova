package services

import (
	"context"

	"github.com/emoss08/trenova/ent/accessorialcharge"
	tools "github.com/emoss08/trenova/util"

	logger "github.com/emoss08/trenova/util/logger"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type AccessorialChargeOps struct {
	Client *ent.Client
	Logger *logrus.Logger
}

// NewAccessorialChargeOps creates a new accessorial charge service.
func NewAccessorialChargeOps() *AccessorialChargeOps {
	return &AccessorialChargeOps{
		Client: database.GetClient(),
		Logger: logger.GetLogger(),
	}
}

// GetAccessorialCharges gets the accessorial charges for an organization.
func (r *AccessorialChargeOps) GetAccessorialCharges(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.AccessorialCharge, int, error) {
	entityCount, countErr := r.Client.AccessorialCharge.Query().Where(
		accessorialcharge.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.AccessorialCharge.Query().
		Limit(limit).
		Offset(offset).
		Where(
			accessorialcharge.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateAccessorialCharge creates a new accessorial charge.
func (r *AccessorialChargeOps) CreateAccessorialCharge(ctx context.Context, newEntity ent.AccessorialCharge) (*ent.AccessorialCharge, error) {
	// Begin a new transaction
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to start transaction")
		r.Logger.WithField("error", wrappedErr).Error("failed to start transaction")
		return nil, wrappedErr
	}

	// Ensure the transaction is either committed or rolled back
	defer func() {
		if v := recover(); v != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(rollbackErr, "failed to rollback transaction")
				r.Logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
			panic(v)
		}
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(err, "failed to rollback transaction")
				r.Logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				wrappedErr := eris.Wrap(err, "failed to commit transaction")
				r.Logger.WithField("error", wrappedErr).Error("failed to commit transaction")
			}
		}
	}()

	createdEntity, err := tx.AccessorialCharge.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetStatus(newEntity.Status).
		SetCode(newEntity.Code).
		SetDescription(newEntity.Description).
		SetIsDetention(newEntity.IsDetention).
		SetMethod(newEntity.Method).
		SetAmount(newEntity.Amount).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create accessorial charge")
	}

	return createdEntity, nil
}

// UpdateAccessorialCharge updates a accessorial charge.
func (r *AccessorialChargeOps) UpdateAccessorialCharge(ctx context.Context, entity ent.AccessorialCharge) (*ent.AccessorialCharge, error) {
	// Begin a new transaction
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to start transaction")
		r.Logger.WithField("error", wrappedErr).Error("failed to start transaction")
		return nil, wrappedErr
	}

	// Ensure the transaction is either committed or rolled back
	defer func() {
		if v := recover(); v != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(rollbackErr, "failed to rollback transaction")
				r.Logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
			panic(v)
		}
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(err, "failed to rollback transaction")
				r.Logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				wrappedErr := eris.Wrap(err, "failed to commit transaction")
				r.Logger.WithField("error", wrappedErr).Error("failed to commit transaction")
			}
		}
	}()

	current, err := tx.AccessorialCharge.Get(ctx, entity.ID)
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to retrieve requested entity")
		r.Logger.WithField("error", wrappedErr).Error("failed to retrieve requested entity")
		return nil, wrappedErr
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, tools.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.AccessorialCharge.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetIsDetention(entity.IsDetention).
		SetMethod(entity.Method).
		SetAmount(entity.Amount).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
