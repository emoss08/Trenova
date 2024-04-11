package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/commodity"
	"github.com/emoss08/trenova/ent/organization"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/logger"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"
)

type CommodityOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewCommodityOps creates a new commodity service.
func NewCommodityOps() *CommodityOps {
	return &CommodityOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetCommodities gets the commodities for an organization.
func (r *CommodityOps) GetCommodities(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.Commodity, int, error) {
	commodityCount, countErr := r.client.Commodity.Query().Where(
		commodity.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	commodities, err := r.client.Commodity.Query().
		Limit(limit).
		Offset(offset).
		WithHazardousMaterial().
		Where(
			commodity.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return commodities, commodityCount, nil
}

// CreateCommodity creates a new commodity.
func (r *CommodityOps) CreateCommodity(ctx context.Context, newEntity ent.Commodity) (*ent.Commodity, error) {
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

	createdEntity, err := tx.Commodity.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetStatus(newEntity.Status).
		SetName(newEntity.Name).
		SetDescription(newEntity.Description).
		SetIsHazmat(newEntity.IsHazmat).
		SetUnitOfMeasure(newEntity.UnitOfMeasure).
		SetMinTemp(newEntity.MinTemp).
		SetMaxTemp(newEntity.MaxTemp).
		SetNillableHazardousMaterialID(newEntity.HazardousMaterialID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

func (r *CommodityOps) UpdateCommodity(ctx context.Context, entity ent.Commodity) (*ent.Commodity, error) {
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

	current, err := tx.Commodity.Get(ctx, entity.ID)
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
	updateOp := tx.Commodity.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetIsHazmat(entity.IsHazmat).
		SetUnitOfMeasure(entity.UnitOfMeasure).
		SetMinTemp(entity.MinTemp).
		SetMaxTemp(entity.MaxTemp).
		SetNillableHazardousMaterialID(entity.HazardousMaterialID).
		SetVersion(entity.Version + 1) // Increment the version

	// If the hazardous material ID is nil, clear the association and set isHazmat to false
	if entity.HazardousMaterialID == nil {
		updateOp = updateOp.ClearHazardousMaterial().SetIsHazmat(false)
	}

	// Execute the update operation
	updatedCommodity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to save updated entity")
	}

	return updatedCommodity, nil
}
