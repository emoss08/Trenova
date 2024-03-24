package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/commodity"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type CommodityOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewCommodityOps creates a new commodity service.
func NewCommodityOps(ctx context.Context) *CommodityOps {
	return &CommodityOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetCommodities gets the commodities for an organization.
func (r *CommodityOps) GetCommodities(limit, offset int, orgID, buID uuid.UUID) ([]*ent.Commodity, int, error) {
	commodityCount, countErr := r.client.Commodity.Query().Where(
		commodity.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	commodities, err := r.client.Commodity.Query().
		Limit(limit).
		Offset(offset).
		WithHazardousMaterial().
		Where(
			commodity.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return commodities, commodityCount, nil
}

// CreateCommodity creates a new commodity.
func (r *CommodityOps) CreateCommodity(newCommodity ent.Commodity) (*ent.Commodity, error) {
	commodity, err := r.client.Commodity.Create().
		SetOrganizationID(newCommodity.OrganizationID).
		SetBusinessUnitID(newCommodity.BusinessUnitID).
		SetStatus(newCommodity.Status).
		SetName(newCommodity.Name).
		SetDescription(newCommodity.Description).
		SetIsHazmat(newCommodity.IsHazmat).
		SetUnitOfMeasure(newCommodity.UnitOfMeasure).
		SetMinTemp(newCommodity.MinTemp).
		SetMaxTemp(newCommodity.MaxTemp).
		SetNillableHazardousMaterialID(newCommodity.HazardousMaterialID).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return commodity, nil
}

func (r *CommodityOps) UpdateCommodity(commodity ent.Commodity) (*ent.Commodity, error) {
	// Start building the update operation
	updateOp := r.client.Commodity.UpdateOneID(commodity.ID).
		SetStatus(commodity.Status).
		SetName(commodity.Name).
		SetDescription(commodity.Description).
		SetIsHazmat(commodity.IsHazmat).
		SetUnitOfMeasure(commodity.UnitOfMeasure).
		SetMinTemp(commodity.MinTemp).
		SetMaxTemp(commodity.MaxTemp).
		SetNillableHazardousMaterialID(commodity.HazardousMaterialID)

	// If the hazardous material ID is nil, clear the association and set isHazmat to false
	if commodity.HazardousMaterialID == nil {
		updateOp = updateOp.ClearHazardousMaterial().SetIsHazmat(false)
	}

	// Execute the update operation
	updatedCommodity, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedCommodity, nil
}
