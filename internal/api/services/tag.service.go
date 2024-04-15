package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/tag"
	"github.com/google/uuid"
)

type TagService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewTagService creates a new tag service.
func NewTagService(s *api.Server) *TagService {
	return &TagService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetTags gets the tags for an organization.
func (r *TagService) GetTags(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.Tag, int, error) {
	entityCount, countErr := r.Client.Tag.Query().Where(
		tag.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.Tag.Query().
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
func (r *TagService) CreateTag(
	ctx context.Context, entity *ent.Tag,
) (*ent.Tag, error) {
	newEntity := new(ent.Tag)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createTagEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return newEntity, nil
}

func (r *TagService) createTagEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Tag,
) (*ent.Tag, error) {
	createdEntity, err := tx.Tag.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetColor(entity.Color).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateTag updates a tag.
func (r *TagService) UpdateTag(
	ctx context.Context, entity *ent.Tag,
) (*ent.Tag, error) {
	updatedEntity := new(ent.Tag)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateTagEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *TagService) updateTagEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Tag,
) (*ent.Tag, error) {
	current, err := tx.Tag.Get(ctx, entity.ID)
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
	updateOp := tx.Tag.UpdateOneID(entity.ID).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetColor(entity.Color).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
