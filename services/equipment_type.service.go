package services

import (
	"context"

	"github.com/emoss08/trenova/ent/equipmenttype"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/logger"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type EquipmentTypeOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewEquipmentTypeOps creates a new equipment type service.
func NewEquipmentTypeOps() *EquipmentTypeOps {
	return &EquipmentTypeOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetEquipmentTypes gets the equipment type for an organization.
func (r *EquipmentTypeOps) GetEquipmentTypes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.EquipmentType, int, error) {
	entityCount, countErr := r.client.EquipmentType.Query().Where(
		equipmenttype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.EquipmentType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			equipmenttype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateEquipmentType creates a new equipment type.
func (r *EquipmentTypeOps) CreateEquipmentType(
	ctx context.Context, newEntity ent.EquipmentType,
) (*ent.EquipmentType, error) {
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

	createdEntity, err := tx.EquipmentType.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetStatus(newEntity.Status).
		SetCode(newEntity.Code).
		SetDescription(newEntity.Description).
		SetCostPerMile(newEntity.CostPerMile).
		SetEquipmentClass(newEntity.EquipmentClass).
		SetFixedCost(newEntity.FixedCost).
		SetVariableCost(newEntity.VariableCost).
		SetHeight(newEntity.Height).
		SetLength(newEntity.Length).
		SetWidth(newEntity.Width).
		SetWeight(newEntity.Weight).
		SetColor(newEntity.Color).
		SetIdlingFuelUsage(newEntity.IdlingFuelUsage).
		SetExemptFromTolls(newEntity.ExemptFromTolls).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create entity")
	}

	return createdEntity, nil
}

// UpdateEquipmentType updates a equipment type.
func (r *EquipmentTypeOps) UpdateEquipmentType(
	ctx context.Context, entity ent.EquipmentType,
) (*ent.EquipmentType, error) {
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

	current, err := tx.EquipmentType.Get(ctx, entity.ID) // Get the current entity.
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
	updateOp := tx.EquipmentType.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetCostPerMile(entity.CostPerMile).
		SetEquipmentClass(entity.EquipmentClass).
		SetFixedCost(entity.FixedCost).
		SetVariableCost(entity.VariableCost).
		SetHeight(entity.Height).
		SetLength(entity.Length).
		SetWidth(entity.Width).
		SetWeight(entity.Weight).
		SetColor(entity.Color).
		SetIdlingFuelUsage(entity.IdlingFuelUsage).
		SetExemptFromTolls(entity.ExemptFromTolls).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
