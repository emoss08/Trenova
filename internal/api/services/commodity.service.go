package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/commodity"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

type CommodityService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewCommodityService creates a new commodity service.
func NewCommodityService(s *api.Server) *CommodityService {
	return &CommodityService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetCommodities gets the commodity for an organization.
func (r *CommodityService) GetCommodities(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.Commodity, int, error) {
	entityCount, countErr := r.Client.Commodity.Query().Where(
		commodity.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.Commodity.Query().
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

	return entities, entityCount, nil
}

// CreateCommodity creates a new commodity.
func (r *CommodityService) CreateCommodity(
	ctx context.Context, entity *ent.Commodity,
) (*ent.Commodity, error) {
	updatedEntity := new(ent.Commodity)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createCommodityEntity(ctx, tx, entity)
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

func (r *CommodityService) createCommodityEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Commodity,
) (*ent.Commodity, error) {
	createdEntity, err := tx.Commodity.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetIsHazmat(entity.IsHazmat).
		SetUnitOfMeasure(entity.UnitOfMeasure).
		SetMinTemp(entity.MinTemp).
		SetMaxTemp(entity.MaxTemp).
		SetNillableHazardousMaterialID(entity.HazardousMaterialID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateCommodity updates a commodity.
func (r *CommodityService) UpdateCommodity(ctx context.Context, entity *ent.Commodity) (*ent.Commodity, error) {
	updatedEntity := new(ent.Commodity)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateCommodityEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *CommodityService) updateCommodityEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Commodity,
) (*ent.Commodity, error) {
	current, err := tx.Commodity.Get(ctx, entity.ID)
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

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
