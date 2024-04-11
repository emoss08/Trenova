package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/locationcategory"
	"github.com/emoss08/trenova/ent/organization"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/logger"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"
)

// LocationCategoryOps is the service for location category.
type LocationCategoryOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewLocationCategoryOps creates a new location category service.
func NewLocationCategoryOps() *LocationCategoryOps {
	return &LocationCategoryOps{
		logger: logger.GetLogger(),
		client: database.GetClient(),
	}
}

// GetLocationCategories gets the location category for an organization.
func (r *LocationCategoryOps) GetLocationCategories(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.LocationCategory, int, error) {
	entityCount, countErr := r.client.LocationCategory.Query().Where(
		locationcategory.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.LocationCategory.Query().
		Limit(limit).
		Offset(offset).
		Where(
			locationcategory.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateLocationCategory creates a new location category for an organization.
func (r *LocationCategoryOps) CreateLocationCategory(
	ctx context.Context, newEntity ent.LocationCategory,
) (*ent.LocationCategory, error) {
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

	createdEntity, err := tx.LocationCategory.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetName(newEntity.Name).
		SetDescription(newEntity.Description).
		SetColor(newEntity.Color).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create entity")
	}

	return createdEntity, nil
}

// UpdateLocationCategory updates an location category for an organization.
func (r *LocationCategoryOps) UpdateLocationCategory(
	ctx context.Context, entity ent.LocationCategory,
) (*ent.LocationCategory, error) {
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

	current, err := tx.LocationCategory.Get(ctx, entity.ID) // Get the current entity.
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

	updatedEntity, err := tx.LocationCategory.UpdateOneID(entity.ID).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetColor(entity.Color).
		SetVersion(entity.Version + 1). // Increment the version.
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
