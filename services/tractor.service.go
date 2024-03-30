package services

import (
	"context"

	"github.com/emoss08/trenova/ent/tractor"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type TractorOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewTractorOps creates a new tractor service.
func NewTractorOps(ctx context.Context) *TractorOps {
	return &TractorOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetTractors gets the tractor for an organization.
func (r *TractorOps) GetTractors(limit, offset int, orgID, buID uuid.UUID) ([]*ent.Tractor, int, error) {
	entityCount, countErr := r.client.Tractor.Query().Where(
		tractor.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.Tractor.Query().
		Limit(limit).
		Offset(offset).
		WithEquipmentType().
		WithPrimaryWorker().
		WithSecondaryWorker().
		WithFleetCode().
		Where(
			tractor.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateTractor creates a new tractor.
func (r *TractorOps) CreateTractor(entity ent.Tractor) (*ent.Tractor, error) {
	newEntity, err := r.client.Tractor.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetEquipmentTypeID(entity.EquipmentTypeID).
		SetLicensePlateNumber(entity.LicensePlateNumber).
		SetVin(entity.Vin).
		SetNillableEquipmentManufacturerID(entity.EquipmentManufacturerID).
		SetModel(entity.Model).
		SetNillableYear(entity.Year).
		SetNillableStateID(entity.StateID).
		SetLeased(entity.Leased).
		SetLeasedDate(entity.LeasedDate).
		SetPrimaryWorkerID(entity.PrimaryWorkerID).
		SetNillableSecondaryWorkerID(entity.SecondaryWorkerID).
		SetFleetCodeID(entity.FleetCodeID).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return newEntity, nil
}

// UpdateTractor updates a tractor.
func (r *TractorOps) UpdateTractor(entity ent.Tractor) (*ent.Tractor, error) {
	// Start building the update operation
	updateOp := r.client.Tractor.UpdateOneID(entity.ID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetEquipmentTypeID(entity.EquipmentTypeID).
		SetLicensePlateNumber(entity.LicensePlateNumber).
		SetVin(entity.Vin).
		SetNillableEquipmentManufacturerID(entity.EquipmentManufacturerID).
		SetModel(entity.Model).
		SetNillableYear(entity.Year).
		SetNillableStateID(entity.StateID).
		SetLeased(entity.Leased).
		SetLeasedDate(entity.LeasedDate).
		SetPrimaryWorkerID(entity.PrimaryWorkerID).
		SetNillableSecondaryWorkerID(entity.SecondaryWorkerID).
		SetFleetCodeID(entity.FleetCodeID)

	// If the secondary worker ID is nil, clear the association.
	if entity.SecondaryWorkerID == nil {
		updateOp = updateOp.ClearSecondaryWorker()
	}

	// Execute the update operation
	updatedEntity, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
