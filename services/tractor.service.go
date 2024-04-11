package services

import (
	"context"

	"github.com/emoss08/trenova/ent/tractor"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/logger"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type TractorOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewTractorOps creates a new tractor service.
func NewTractorOps() *TractorOps {
	return &TractorOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetTractors gets the tractor for an organization.
func (r *TractorOps) GetTractors(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.Tractor, int, error) {
	entityCount, countErr := r.client.Tractor.Query().Where(
		tractor.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.Tractor.Query().
		Limit(limit).
		Offset(offset).
		WithEquipmentType().
		WithPrimaryWorker().
		WithSecondaryWorker().
		WithFleetCode().
		Where(
			tractor.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateTractor creates a new tractor.
func (r *TractorOps) CreateTractor(
	ctx context.Context, newEntity ent.Tractor,
) (*ent.Tractor, error) {
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

	createdEntity, err := tx.Tractor.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetCode(newEntity.Code).
		SetStatus(newEntity.Status).
		SetEquipmentTypeID(newEntity.EquipmentTypeID).
		SetLicensePlateNumber(newEntity.LicensePlateNumber).
		SetVin(newEntity.Vin).
		SetNillableEquipmentManufacturerID(newEntity.EquipmentManufacturerID).
		SetModel(newEntity.Model).
		SetNillableYear(newEntity.Year).
		SetNillableStateID(newEntity.StateID).
		SetLeased(newEntity.Leased).
		SetLeasedDate(newEntity.LeasedDate).
		SetPrimaryWorkerID(newEntity.PrimaryWorkerID).
		SetNillableSecondaryWorkerID(newEntity.SecondaryWorkerID).
		SetFleetCodeID(newEntity.FleetCodeID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateTractor updates a tractor.
func (r *TractorOps) UpdateTractor(
	ctx context.Context, entity ent.Tractor,
) (*ent.Tractor, error) {
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

	current, err := tx.Tractor.Get(ctx, entity.ID) // Get the current entity.
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
	updateOp := tx.Tractor.UpdateOneID(entity.ID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetEquipmentTypeID(entity.EquipmentTypeID).
		SetLicensePlateNumber(entity.LicensePlateNumber).
		SetVin(entity.Vin).
		SetNillableEquipmentManufacturerID(entity.EquipmentManufacturerID).
		SetModel(entity.Model).
		SetNillableYear(entity.Year).
		SetNillableStateID(entity.StateID).
		SetLeased(entity.Leased).
		SetLeasedDate(entity.LeasedDate).
		SetPrimaryWorkerID(entity.PrimaryWorkerID).
		SetNillableSecondaryWorkerID(entity.SecondaryWorkerID).
		SetFleetCodeID(entity.FleetCodeID).
		SetVersion(entity.Version + 1) // Increment the version

	// If the secondary worker ID is nil, clear the association.
	if entity.SecondaryWorkerID == nil {
		updateOp = updateOp.ClearSecondaryWorker()
	}

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
