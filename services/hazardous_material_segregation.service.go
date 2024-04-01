package services

import (
	"context"

	"github.com/emoss08/trenova/ent/hazardousmaterialsegregation"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/logger"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type HazardousMaterialSegregationOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewHazardousMaterialSegregationOps creates a new fleet code service.
func NewHazardousMaterialSegregationOps() *HazardousMaterialSegregationOps {
	return &HazardousMaterialSegregationOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetHazmatSegRules gets the fleet code for an organization.
func (r *HazardousMaterialSegregationOps) GetHazmatSegRules(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.HazardousMaterialSegregation, int, error) {
	entityCount, countErr := r.client.HazardousMaterialSegregation.Query().Where(
		hazardousmaterialsegregation.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.HazardousMaterialSegregation.Query().
		Limit(limit).
		Offset(offset).
		Where(
			hazardousmaterialsegregation.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateHazmatSegRule creates a new accessorial charge.
func (r *HazardousMaterialSegregationOps) CreateHazmatSegRule(
	ctx context.Context, newEntity ent.HazardousMaterialSegregation,
) (*ent.HazardousMaterialSegregation, error) {
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

	createdEntity, err := tx.HazardousMaterialSegregation.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetClassA(newEntity.ClassA).
		SetClassB(newEntity.ClassB).
		SetSegregationType(newEntity.SegregationType).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create entity")
	}

	return createdEntity, nil
}

// UpdateHazmatSegRule updates a fleet code.
func (r *HazardousMaterialSegregationOps) UpdateHazmatSegRule(
	ctx context.Context, entity ent.HazardousMaterialSegregation,
) (*ent.HazardousMaterialSegregation, error) {
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

	current, err := tx.EquipmentManufactuer.Get(ctx, entity.ID) // Get the current entity.
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

	// Start building the update operation
	updateOp := tx.HazardousMaterialSegregation.UpdateOneID(entity.ID).
		SetClassA(entity.ClassA).
		SetClassB(entity.ClassB).
		SetSegregationType(entity.SegregationType).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
