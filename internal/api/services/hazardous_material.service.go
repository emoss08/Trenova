package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hazardousmaterial"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

type HazardousMaterialService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewHazardousMaterialService creates a new hazardous material service.
func NewHazardousMaterialService(s *api.Server) *HazardousMaterialService {
	return &HazardousMaterialService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetHazardousMaterials gets the hazardous materials for an organization.
func (r *HazardousMaterialService) GetHazardousMaterials(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.HazardousMaterial, int, error) {
	entityCount, countErr := r.Client.HazardousMaterial.Query().Where(
		hazardousmaterial.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.HazardousMaterial.Query().
		Limit(limit).
		Offset(offset).
		Where(
			hazardousmaterial.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateHazardousMaterial creates a new hazardous material.
func (r *HazardousMaterialService) CreateHazardousMaterial(
	ctx context.Context, entity *ent.HazardousMaterial,
) (*ent.HazardousMaterial, error) {
	newEntity := new(ent.HazardousMaterial)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createHazardousMaterialEntity(ctx, tx, entity)
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

func (r *HazardousMaterialService) createHazardousMaterialEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.HazardousMaterial,
) (*ent.HazardousMaterial, error) {
	createdEntity, err := tx.HazardousMaterial.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetHazardClass(entity.HazardClass).
		SetErgNumber(entity.ErgNumber).
		SetDescription(entity.Description).
		SetPackingGroup(entity.PackingGroup).
		SetProperShippingName(entity.ProperShippingName).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateHazardousMaterial updates a hazardous material.
func (r *HazardousMaterialService) UpdateHazardousMaterial(
	ctx context.Context, entity *ent.HazardousMaterial,
) (*ent.HazardousMaterial, error) {
	updatedEntity := new(ent.HazardousMaterial)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateHazardousMaterialEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *HazardousMaterialService) updateHazardousMaterialEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.HazardousMaterial,
) (*ent.HazardousMaterial, error) {
	current, err := tx.HazardousMaterial.Get(ctx, entity.ID)
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
	updateOp := tx.HazardousMaterial.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetHazardClass(entity.HazardClass).
		SetErgNumber(entity.ErgNumber).
		SetDescription(entity.Description).
		SetPackingGroup(entity.PackingGroup).
		SetProperShippingName(entity.ProperShippingName).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
