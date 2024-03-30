package services

import (
	"context"

	"github.com/emoss08/trenova/ent/tag"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type TagOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewTagOps creates a new tag service.
func NewTagOps(ctx context.Context) *TagOps {
	return &TagOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetTags gets the tags for an organization.
func (r *TagOps) GetTags(limit, offset int, orgID, buID uuid.UUID) ([]*ent.Tag, int, error) {
	entityCount, countErr := r.client.Tag.Query().Where(
		tag.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

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
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateTag creates a new tag.
func (r *TagOps) CreateTag(entity ent.Tag) (*ent.Tag, error) {
	newEntity, err := r.client.Tag.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetColor(entity.Color).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return newEntity, nil
}

// UpdateTag updates a tag.
func (r *TagOps) UpdateTag(entity ent.Tag) (*ent.Tag, error) {
	// Start building the update operation
	updateOp := r.client.Tag.UpdateOneID(entity.ID).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetColor(entity.Color)

	// Execute the update operation
	updatedEntity, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
