package services

import (
	"context"

	"github.com/emoss08/trenova/ent/shipmenttype"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type ShipmentTypeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewShipmentTypeOps creates a new shipment type service.
func NewShipmentTypeOps(ctx context.Context) *ShipmentTypeOps {
	return &ShipmentTypeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetShipmentTypes gets the shipment type for an organization.
func (r *ShipmentTypeOps) GetShipmentTypes(limit, offset int, orgID, buID uuid.UUID) ([]*ent.ShipmentType, int, error) {
	shipTypeCount, countErr := r.client.ShipmentType.Query().Where(
		shipmenttype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	shipmentTypes, err := r.client.ShipmentType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			shipmenttype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return shipmentTypes, shipTypeCount, nil
}

// CreateShipmentType creates a new shipment type.
func (r *ShipmentTypeOps) CreateShipmentType(newShipmentType ent.ShipmentType) (*ent.ShipmentType, error) {
	shipmentType, err := r.client.ShipmentType.Create().
		SetOrganizationID(newShipmentType.OrganizationID).
		SetBusinessUnitID(newShipmentType.BusinessUnitID).
		SetStatus(newShipmentType.Status).
		SetCode(newShipmentType.Code).
		SetDescription(newShipmentType.Description).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return shipmentType, nil
}

// UpdateShipmentType updates a shipment type.
func (r *ShipmentTypeOps) UpdateShipmentType(shipmentType ent.ShipmentType) (*ent.ShipmentType, error) {
	// Start building the update operation
	updateOp := r.client.ShipmentType.UpdateOneID(shipmentType.ID).
		SetStatus(shipmentType.Status).
		SetCode(shipmentType.Code).
		SetDescription(shipmentType.Description)

	// Execute the update operation
	updateShipmentType, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updateShipmentType, nil
}
