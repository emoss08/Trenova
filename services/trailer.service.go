package services

import (
	"context"

	"github.com/emoss08/trenova/ent/trailer"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/logger"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type TrailerOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewTrailerOps creates a new trailer service.
func NewTrailerOps() *TrailerOps {
	return &TrailerOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetTrailers gets the trailer for an organization.
func (r *TrailerOps) GetTrailers(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.Trailer, int, error) {
	entityCount, countErr := r.client.Trailer.Query().Where(
		trailer.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.Trailer.Query().
		Limit(limit).
		Offset(offset).
		WithEquipmentManufacturer().
		WithState().
		WithRegistrationState().
		WithEquipmentType().
		WithFleetCode().
		Where(
			trailer.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateTrailer creates a new trailer.
func (r *TrailerOps) CreateTrailer(
	ctx context.Context, newEntity ent.Trailer,
) (*ent.Trailer, error) {
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

	createdEntity, err := tx.Trailer.Create().
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
		SetFleetCodeID(newEntity.FleetCodeID).
		SetLastInspectionDate(newEntity.LastInspectionDate).
		SetRegistrationNumber(newEntity.RegistrationNumber).
		SetNillableRegistrationStateID(newEntity.RegistrationStateID).
		SetRegistrationExpirationDate(newEntity.RegistrationExpirationDate).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateTrailer updates a trailer.
func (r *TrailerOps) UpdateTrailer(
	ctx context.Context, entity ent.Trailer,
) (*ent.Trailer, error) {
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

	current, err := tx.Trailer.Get(ctx, entity.ID) // Get the current entity.
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
	updateOp := tx.Trailer.UpdateOneID(entity.ID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetEquipmentTypeID(entity.EquipmentTypeID).
		SetLicensePlateNumber(entity.LicensePlateNumber).
		SetVin(entity.Vin).
		SetNillableEquipmentManufacturerID(entity.EquipmentManufacturerID).
		SetModel(entity.Model).
		SetNillableYear(entity.Year).
		SetNillableStateID(entity.StateID).
		SetFleetCodeID(entity.FleetCodeID).
		SetLastInspectionDate(entity.LastInspectionDate).
		SetRegistrationNumber(entity.RegistrationNumber).
		SetNillableRegistrationStateID(entity.RegistrationStateID).
		SetRegistrationExpirationDate(entity.RegistrationExpirationDate).
		SetVersion(entity.Version + 1) // Increment the version

	// If registration state id is nil clear the assocation.
	if entity.RegistrationStateID == nil {
		updateOp.ClearRegistrationState()
	}

	// If the equipment manufacturer id is nil clear the association.
	if entity.EquipmentManufacturerID == nil {
		updateOp.ClearEquipmentManufacturer()
	}

	// If the registration state id is nil clear the association.
	if entity.RegistrationStateID == nil {
		updateOp.ClearRegistrationState()
	}

	// If the state id is nil clear the association.
	if entity.StateID == nil {
		updateOp.ClearState()
	}

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
