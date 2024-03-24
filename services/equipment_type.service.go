package services

import (
	"context"

	"github.com/emoss08/trenova/ent/equipmenttype"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type EquipmentTypeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewEquipmentTypeOps creates a new equipment type service.
func NewEquipmentTypeOps(ctx context.Context) *EquipmentTypeOps {
	return &EquipmentTypeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetEquipmentTypes gets the equipment type for an organization.
func (r *EquipmentTypeOps) GetEquipmentTypes(limit, offset int, orgID, buID uuid.UUID) ([]*ent.EquipmentType, int, error) {
	equipTypeCount, countErr := r.client.EquipmentType.Query().Where(
		equipmenttype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	equipmentTypes, err := r.client.EquipmentType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			equipmenttype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return equipmentTypes, equipTypeCount, nil
}

// CreateEquipmentType creates a new equipment type.
func (r *EquipmentTypeOps) CreateEquipmentType(newEquipType ent.EquipmentType) (*ent.EquipmentType, error) {
	equipmentType, err := r.client.EquipmentType.Create().
		SetOrganizationID(newEquipType.OrganizationID).
		SetBusinessUnitID(newEquipType.BusinessUnitID).
		SetStatus(newEquipType.Status).
		SetName(newEquipType.Name).
		SetDescription(newEquipType.Description).
		SetCostPerMile(newEquipType.CostPerMile).
		SetEquipmentClass(newEquipType.EquipmentClass).
		SetFixedCost(newEquipType.FixedCost).
		SetVariableCost(newEquipType.VariableCost).
		SetHeight(newEquipType.Height).
		SetLength(newEquipType.Length).
		SetWidth(newEquipType.Width).
		SetWeight(newEquipType.Weight).
		SetIdlingFuelUsage(newEquipType.IdlingFuelUsage).
		SetExemptFromTolls(newEquipType.ExemptFromTolls).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return equipmentType, nil
}

// UpdateEquipmentType updates a equipment type.
func (r *EquipmentTypeOps) UpdateEquipmentType(equipmentType ent.EquipmentType) (*ent.EquipmentType, error) {
	// Start building the update operation
	updateOp := r.client.EquipmentType.UpdateOneID(equipmentType.ID).
		SetStatus(equipmentType.Status).
		SetName(equipmentType.Name).
		SetDescription(equipmentType.Description).
		SetCostPerMile(equipmentType.CostPerMile).
		SetEquipmentClass(equipmentType.EquipmentClass).
		SetFixedCost(equipmentType.FixedCost).
		SetVariableCost(equipmentType.VariableCost).
		SetHeight(equipmentType.Height).
		SetLength(equipmentType.Length).
		SetWidth(equipmentType.Width).
		SetWeight(equipmentType.Weight).
		SetIdlingFuelUsage(equipmentType.IdlingFuelUsage).
		SetExemptFromTolls(equipmentType.ExemptFromTolls)

	// Execute the update operation
	updatedEquipType, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedEquipType, nil
}
