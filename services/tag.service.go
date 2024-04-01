package services

import (
	"context"

	"github.com/emoss08/trenova/ent/tag"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/logger"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type TagOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewTagOps creates a new tag service.
func NewTagOps() *TagOps {
	return &TagOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetTags gets the tags for an organization.
func (r *TagOps) GetTags(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.Tag, int, error) {
	entityCount, countErr := r.client.Tag.Query().Where(
		tag.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.Tag.Query().
		Limit(limit).
		Offset(offset).
		Where(
			tag.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateTag creates a new tag.
func (r *TagOps) CreateTag(
	ctx context.Context, newEntity ent.Tag,
) (*ent.Tag, error) {
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

	createdEntity, err := tx.Tag.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetName(newEntity.Name).
		SetDescription(newEntity.Description).
		SetColor(newEntity.Color).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateTag updates a tag.
func (r *TagOps) UpdateTag(
	ctx context.Context, entity ent.Tag,
) (*ent.Tag, error) {
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

	current, err := tx.Tag.Get(ctx, entity.ID) // Get the current entity.
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
	updateOp := tx.Tag.UpdateOneID(entity.ID).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetColor(entity.Color).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
