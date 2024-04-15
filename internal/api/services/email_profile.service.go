package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/emailprofile"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// EmailProfileService is the service for email profile settings.
type EmailProfileService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewEmailProfileService creates a new email profile service.
func NewEmailProfileService(s *api.Server) *EmailProfileService {
	return &EmailProfileService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetEmailProfiles gets the email profiles for an organization.
func (r *EmailProfileService) GetEmailProfiles(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.EmailProfile, int, error) {
	entityCount, countErr := r.Client.EmailProfile.Query().Where(
		emailprofile.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.EmailProfile.Query().
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
func (r *EmailProfileService) CreateEmailProfile(
	ctx context.Context, entity *ent.EmailProfile,
) (*ent.EmailProfile, error) {
	updatedEntity := new(ent.EmailProfile)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createEmailProfileEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *EmailProfileService) createEmailProfileEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.EmailProfile,
) (*ent.EmailProfile, error) {
	createdEntity, err := tx.EmailProfile.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetName(entity.Name).
		SetEmail(entity.Email).
		SetProtocol(entity.Protocol).
		SetHost(entity.Host).
		SetPort(entity.Port).
		SetUsername(entity.Username).
		SetPassword(entity.Password).
		SetIsDefault(entity.IsDefault).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateEmailProfile updates an email profile for an organization.
func (r *EmailProfileService) UpdateEmailProfile(
	ctx context.Context, entity *ent.EmailProfile,
) (*ent.EmailProfile, error) {
	updatedEntity := new(ent.EmailProfile)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateEmailProfileEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *EmailProfileService) updateEmailProfileEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.EmailProfile,
) (*ent.EmailProfile, error) {
	current, err := tx.EmailProfile.Get(ctx, entity.ID)
	if err != nil {
		return nil, err
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"name")
	}

	// Start building the update operation
	updateOp := tx.EmailProfile.UpdateOneID(entity.ID).
		SetName(entity.Name).
		SetEmail(entity.Email).
		SetProtocol(entity.Protocol).
		SetHost(entity.Host).
		SetPort(entity.Port).
		SetUsername(entity.Username).
		SetPassword(entity.Password).
		SetIsDefault(entity.IsDefault).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
