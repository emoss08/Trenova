package services

import (
	"context"

	"github.com/emoss08/trenova/ent/equipmentmanufactuer"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type EquipmentManufactuerOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewEquipmentManufactuerOps creates a new equipment manufacturer service.
func NewEquipmentManufactuerOps(ctx context.Context) *EquipmentManufactuerOps {
	return &EquipmentManufactuerOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetEquipmentManufacturers gets the equipment manufacturer for an organization.
func (r *EquipmentManufactuerOps) GetEquipmentManufacturers(limit, offset int, orgID, buID uuid.UUID) ([]*ent.EquipmentManufactuer, int, error) {
	equipManuCount, countErr := r.client.EquipmentManufactuer.Query().Where(
		equipmentmanufactuer.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	equipmentmanufactuers, err := r.client.EquipmentManufactuer.Query().
		Limit(limit).
		Offset(offset).
		Where(
			equipmentmanufactuer.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return equipmentmanufactuers, equipManuCount, nil
}

// CreateEquipmentManufacturer creates a new equipment manufacturer.
func (r *EquipmentManufactuerOps) CreateEquipmentManufacturer(newEquipMenu ent.EquipmentManufactuer) (*ent.EquipmentManufactuer, error) {
	equipmentManufacturer, err := r.client.EquipmentManufactuer.Create().
		SetOrganizationID(newEquipMenu.OrganizationID).
		SetBusinessUnitID(newEquipMenu.BusinessUnitID).
		SetStatus(newEquipMenu.Status).
		SetName(newEquipMenu.Name).
		SetDescription(newEquipMenu.Description).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return equipmentManufacturer, nil
}

// UpdateEquipmentManufacturer updates a equipment manufacturer.
func (r *EquipmentManufactuerOps) UpdateEquipmentManufacturer(equipmentManufacturer ent.EquipmentManufactuer) (*ent.EquipmentManufactuer, error) {
	// Start building the update operation
	updateOp := r.client.EquipmentManufactuer.UpdateOneID(equipmentManufacturer.ID).
		SetStatus(equipmentManufacturer.Status).
		SetName(equipmentManufacturer.Name).
		SetDescription(equipmentManufacturer.Description)

	// Execute the update operation
	updatedEquipManu, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedEquipManu, nil
}
