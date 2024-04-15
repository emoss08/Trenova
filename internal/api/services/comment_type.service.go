package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/commenttype"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

type CommentTypeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewCommentTypeService creates a new comment type service.
func NewCommentTypeService(s *api.Server) *CommentTypeService {
	return &CommentTypeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetCommentTypes gets the comment type for an organization.
func (r *CommentTypeService) GetCommentTypes(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.CommentType, int, error) {
	entityCount, countErr := r.Client.CommentType.Query().Where(
		commenttype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.CommentType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			commenttype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateCommentType creates a new comment type.
func (r *CommentTypeService) CreateCommentType(
	ctx context.Context, entity *ent.CommentType,
) (*ent.CommentType, error) {
	updatedEntity := new(ent.CommentType)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createCommentTypeEntity(ctx, tx, entity)
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

func (r *CommentTypeService) createCommentTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.CommentType,
) (*ent.CommentType, error) {
	createdEntity, err := tx.CommentType.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetSeverity(entity.Severity).
		SetDescription(entity.Description).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateCommentType updates a comment type.
func (r *CommentTypeService) UpdateCommentType(ctx context.Context, entity *ent.CommentType) (*ent.CommentType, error) {
	updatedEntity := new(ent.CommentType)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateCommentTypeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *CommentTypeService) updateCommentTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.CommentType,
) (*ent.CommentType, error) {
	current, err := tx.CommentType.Get(ctx, entity.ID)
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
	updateOp := tx.CommentType.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetDescription(entity.Description).
		SetName(entity.Name).
		SetSeverity(entity.Severity).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
