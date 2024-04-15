package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/equipmenttype"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
)

type EquipmentTypeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewEquipmentTypeService creates a new equipment type service.
func NewEquipmentTypeService(s *api.Server) *EquipmentTypeService {
	return &EquipmentTypeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetEquipmentManufacturers gets the equipment types for an organization.
func (r *EquipmentTypeService) GetEquipmentTypes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.EquipmentType, int, error) {
	entityCount, countErr := r.Client.EquipmentType.Query().Where(
		equipmenttype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.EquipmentType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			equipmenttype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateEquipmentType creates a new equipment type.
func (r *EquipmentTypeService) CreateEquipmentType(
	ctx context.Context, entity *ent.EquipmentType,
) (*ent.EquipmentType, error) {
	updatedEntity := new(ent.EquipmentType)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createEquipmentTypeEntity(ctx, tx, entity)
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

func (r *EquipmentTypeService) createEquipmentTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.EquipmentType,
) (*ent.EquipmentType, error) {
	createdEntity, err := tx.EquipmentType.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetCostPerMile(entity.CostPerMile).
		SetEquipmentClass(entity.EquipmentClass).
		SetFixedCost(entity.FixedCost).
		SetVariableCost(entity.VariableCost).
		SetHeight(entity.Height).
		SetLength(entity.Length).
		SetWidth(entity.Width).
		SetWeight(entity.Weight).
		SetColor(entity.Color).
		SetIdlingFuelUsage(entity.IdlingFuelUsage).
		SetExemptFromTolls(entity.ExemptFromTolls).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create entity")
	}

	return createdEntity, nil
}

// UpdateEquipmentType updates a equipment type.
func (r *EquipmentTypeService) UpdateEquipmentType(
	ctx context.Context, entity *ent.EquipmentType,
) (*ent.EquipmentType, error) {
	updatedEntity := new(ent.EquipmentType)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateEquipmentTypeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *EquipmentTypeService) updateEquipmentTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.EquipmentType,
) (*ent.EquipmentType, error) {
	current, err := tx.EquipmentType.Get(ctx, entity.ID)
	if err != nil {
		return nil, eris.Wrap(err, "failed to retrieve requested entity")
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.EquipmentType.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetCostPerMile(entity.CostPerMile).
		SetEquipmentClass(entity.EquipmentClass).
		SetFixedCost(entity.FixedCost).
		SetVariableCost(entity.VariableCost).
		SetHeight(entity.Height).
		SetLength(entity.Length).
		SetWidth(entity.Width).
		SetWeight(entity.Weight).
		SetColor(entity.Color).
		SetIdlingFuelUsage(entity.IdlingFuelUsage).
		SetExemptFromTolls(entity.ExemptFromTolls).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
