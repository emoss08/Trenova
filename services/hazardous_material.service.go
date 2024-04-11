package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/hazardousmaterial"
	"github.com/emoss08/trenova/ent/organization"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/logger"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"
)

// HazardousMaterialOps is the service for hazardous material.
type HazardousMaterialOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewHazardousMaterialOps creates a new hazardous material service.
func NewHazardousMaterialOps() *HazardousMaterialOps {
	return &HazardousMaterialOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetHazardousMaterials gets the hazardous material for an organization.
func (r *HazardousMaterialOps) GetHazardousMaterials(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.HazardousMaterial, int, error) {
	entityCount, countErr := r.client.HazardousMaterial.Query().Where(
		hazardousmaterial.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.HazardousMaterial.Query().
		Limit(limit).
		Offset(offset).
		Where(
			hazardousmaterial.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateHazardousMaterial creates a new hazardous material for an organization.
func (r *HazardousMaterialOps) CreateHazardousMaterial(
	ctx context.Context, newEntity ent.HazardousMaterial,
) (*ent.HazardousMaterial, error) {
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

	createdEntity, err := tx.HazardousMaterial.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetStatus(newEntity.Status).
		SetName(newEntity.Name).
		SetHazardClass(newEntity.HazardClass).
		SetErgNumber(newEntity.ErgNumber).
		SetDescription(newEntity.Description).
		SetPackingGroup(newEntity.PackingGroup).
		SetProperShippingName(newEntity.ProperShippingName).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create entity")
	}

	return createdEntity, nil
}

// UpdateHazardousMaterial updates an hazardous material for an organization.
func (r *HazardousMaterialOps) UpdateHazardousMaterial(
	ctx context.Context, entity ent.HazardousMaterial,
) (*ent.HazardousMaterial, error) {
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

	current, err := tx.HazardousMaterial.Get(ctx, entity.ID) // Get the current entity.
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to retrieve requested entity")
		r.logger.WithField("error", wrappedErr).Error("failed to retrieve requested entity")
		return nil, wrappedErr
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, tools.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"name")
	}

	updatedEntity, err := tx.HazardousMaterial.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetHazardClass(entity.HazardClass).
		SetErgNumber(entity.ErgNumber).
		SetDescription(entity.Description).
		SetPackingGroup(entity.PackingGroup).
		SetProperShippingName(entity.ProperShippingName).
		SetVersion(entity.Version + 1). // Increment the version
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
