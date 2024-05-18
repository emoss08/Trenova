package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/locationcategory"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

type LocationCategoryService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewLocationCategoryService creates a new location category service.
func NewLocationCategoryService(s *api.Server) *LocationCategoryService {
	return &LocationCategoryService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetLocationCategories gets the location categories for an organization.
func (r *LocationCategoryService) GetLocationCategories(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.LocationCategory, int, error) {
	entityCount, countErr := r.Client.LocationCategory.Query().Where(
		locationcategory.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.LocationCategory.Query().
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

// CreateLocationCategory creates a new location category.
func (r *LocationCategoryService) CreateLocationCategory(
	ctx context.Context, entity *ent.LocationCategory,
) (*ent.LocationCategory, error) {
	newEntity := new(ent.LocationCategory)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createLocationCategory(ctx, tx, entity)
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

func (r *LocationCategoryService) createLocationCategory(
	ctx context.Context, tx *ent.Tx, entity *ent.LocationCategory,
) (*ent.LocationCategory, error) {
	createdEntity, err := tx.LocationCategory.Create().
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

// UpdateLocationCategory updates a location category.
func (r *LocationCategoryService) UpdateLocationCategory(
	ctx context.Context, entity *ent.LocationCategory,
) (*ent.LocationCategory, error) {
	updatedEntity := new(ent.LocationCategory)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateLocationCategoryEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *LocationCategoryService) updateLocationCategoryEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.LocationCategory,
) (*ent.LocationCategory, error) {
	current, err := tx.LocationCategory.Get(ctx, entity.ID)
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
	updateOp := tx.LocationCategory.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetColor(entity.Color).
		SetVersion(entity.Version + 1) // Increment the version.

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
