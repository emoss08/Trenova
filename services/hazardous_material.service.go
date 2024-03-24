package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/hazardousmaterial"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// HazardousMaterialOps is the service for hazardous material.
type HazardousMaterialOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewHazardousMaterialOps creates a new hazardous material service.
func NewHazardousMaterialOps(ctx context.Context) *HazardousMaterialOps {
	return &HazardousMaterialOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetHazardousMaterials gets the hazardous material for an organization.
func (r *HazardousMaterialOps) GetHazardousMaterials(limit, offset int, orgID, buID uuid.UUID) ([]*ent.HazardousMaterial, int, error) {
	hmCount, countErr := r.client.HazardousMaterial.Query().Where(
		hazardousmaterial.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	hazardousMaterials, err := r.client.HazardousMaterial.Query().
		Limit(limit).
		Offset(offset).
		Where(
			hazardousmaterial.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return hazardousMaterials, hmCount, nil
}

// CreateHazardousMaterial creates a new hazardous material for an organization.
func (r *HazardousMaterialOps) CreateHazardousMaterial(newHazardousMaterial ent.HazardousMaterial) (*ent.HazardousMaterial, error) {
	hazardousMaterial, err := r.client.HazardousMaterial.Create().
		SetOrganizationID(newHazardousMaterial.OrganizationID).
		SetBusinessUnitID(newHazardousMaterial.BusinessUnitID).
		SetStatus(newHazardousMaterial.Status).
		SetName(newHazardousMaterial.Name).
		SetHazardClass(newHazardousMaterial.HazardClass).
		SetErgNumber(newHazardousMaterial.ErgNumber).
		SetDescription(newHazardousMaterial.Description).
		SetPackingGroup(newHazardousMaterial.PackingGroup).
		SetProperShippingName(newHazardousMaterial.ProperShippingName).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return hazardousMaterial, nil
}

// UpdateHazardousMaterial updates an hazardous material for an organization.
func (r *HazardousMaterialOps) UpdateHazardousMaterial(hazardousMaterial ent.HazardousMaterial) (*ent.HazardousMaterial, error) {
	updatedHM, err := r.client.HazardousMaterial.UpdateOneID(hazardousMaterial.ID).
		SetStatus(hazardousMaterial.Status).
		SetName(hazardousMaterial.Name).
		SetHazardClass(hazardousMaterial.HazardClass).
		SetErgNumber(hazardousMaterial.ErgNumber).
		SetDescription(hazardousMaterial.Description).
		SetPackingGroup(hazardousMaterial.PackingGroup).
		SetProperShippingName(hazardousMaterial.ProperShippingName).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedHM, nil
}
