package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/emailprofile"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/logger"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"
)

// EmailProfileOps is the service for email profiles.
type EmailProfileOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewEmailProfileOps creates a new email profiles service.
func NewEmailProfileOps() *EmailProfileOps {
	return &EmailProfileOps{
		logger: logger.GetLogger(),
		client: database.GetClient(),
	}
}

// GetEmailProfiles gets the email profiles for an organization.
func (r *EmailProfileOps) GetEmailProfiles(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.EmailProfile, int, error) {
	entityCount, countErr := r.client.EmailProfile.Query().Where(
		emailprofile.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.EmailProfile.Query().
		Limit(limit).
		Offset(offset).
		Where(
			emailprofile.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateEmailProfile creates a new email profile for an organization.
func (r *EmailProfileOps) CreateEmailProfile(ctx context.Context, newEntity ent.EmailProfile) (*ent.EmailProfile, error) {
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

	createdEntity, err := tx.EmailProfile.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetName(newEntity.Name).
		SetEmail(newEntity.Email).
		SetProtocol(newEntity.Protocol).
		SetHost(newEntity.Host).
		SetPort(newEntity.Port).
		SetUsername(newEntity.Username).
		SetPassword(newEntity.Password).
		SetIsDefault(newEntity.IsDefault).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateEmailProfile updates an email profile for an organization.
func (r *EmailProfileOps) UpdateEmailProfile(ctx context.Context, entity ent.EmailProfile) (*ent.EmailProfile, error) {
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

	current, err := tx.EmailProfile.Get(ctx, entity.ID) // Get the current entity.
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

	updatedEmailProfile, err := tx.EmailProfile.UpdateOneID(entity.ID).
		SetName(entity.Name).
		SetEmail(entity.Email).
		SetProtocol(entity.Protocol).
		SetHost(entity.Host).
		SetPort(entity.Port).
		SetUsername(entity.Username).
		SetPassword(entity.Password).
		SetIsDefault(entity.IsDefault).
		SetVersion(entity.Version + 1). // Increment the version
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEmailProfile, nil
}
