package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/locationcategory"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// LocationCategoryOps is the service for location category.
type LocationCategoryOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewLocationCategoryOps creates a new location category service.
func NewLocationCategoryOps(ctx context.Context) *LocationCategoryOps {
	return &LocationCategoryOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetLocationCategories gets the location category for an organization.
func (r *LocationCategoryOps) GetLocationCategories(limit, offset int, orgID, buID uuid.UUID) ([]*ent.LocationCategory, int, error) {
	lcCount, countErr := r.client.LocationCategory.Query().Where(
		locationcategory.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	locationCategories, err := r.client.LocationCategory.Query().
		Limit(limit).
		Offset(offset).
		Where(
			locationcategory.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return locationCategories, lcCount, nil
}

// CreateLocationCategory creates a new location category for an organization.
func (r *LocationCategoryOps) CreateLocationCategory(newLocationCategory ent.LocationCategory) (*ent.LocationCategory, error) {
	locationCategory, err := r.client.LocationCategory.Create().
		SetOrganizationID(newLocationCategory.OrganizationID).
		SetBusinessUnitID(newLocationCategory.BusinessUnitID).
		SetName(newLocationCategory.Name).
		SetDescription(newLocationCategory.Description).
		SetColor(newLocationCategory.Color).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return locationCategory, nil
}

// UpdateLocationCategory updates an location category for an organization.
func (r *LocationCategoryOps) UpdateLocationCategory(locationCategory ent.LocationCategory) (*ent.LocationCategory, error) {
	updatedLC, err := r.client.LocationCategory.UpdateOneID(locationCategory.ID).
		SetName(locationCategory.Name).
		SetDescription(locationCategory.Description).
		SetColor(locationCategory.Color).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedLC, nil
}
