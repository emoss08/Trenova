package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/ent/shipmentcontrol"
	"github.com/google/uuid"
)

// ShipmentControlOps is the service for shipment control settings.
type ShipmentControlOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewShipmentControlOps creates a new shipment control service.
func NewShipmentControlOps(ctx context.Context) *ShipmentControlOps {
	return &ShipmentControlOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetShipmentControl creates a new shipment control settings for an organization.
func (r *ShipmentControlOps) GetShipmentControl(orgID, buID uuid.UUID) (*ent.ShipmentControl, error) {
	shipmentControl, err := r.client.ShipmentControl.Query().Where(
		shipmentcontrol.HasOrganizationWith(
			organization.ID(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return shipmentControl, nil
}

// UpdateShipmentControl updates the shipment control settings for an organization.
func (r *ShipmentControlOps) UpdateShipmentControl(sc ent.ShipmentControl) (*ent.ShipmentControl, error) {
	updatedSC, err := r.client.ShipmentControl.
		UpdateOneID(sc.ID).
		SetAutoRateShipment(sc.AutoRateShipment).
		SetCalculateDistance(sc.CalculateDistance).
		SetEnforceRevCode(sc.EnforceRevCode).
		SetEnforceVoidedComm(sc.EnforceVoidedComm).
		SetGenerateRoutes(sc.GenerateRoutes).
		SetEnforceCommodity(sc.EnforceCommodity).
		SetAutoSequenceStops(sc.AutoSequenceStops).
		SetAutoShipmentTotal(sc.AutoShipmentTotal).
		SetEnforceOriginDestination(sc.EnforceOriginDestination).
		SetCheckForDuplicateBol(sc.CheckForDuplicateBol).
		SetSendPlacardInfo(sc.SendPlacardInfo).
		SetEnforceHazmatSegRules(sc.EnforceHazmatSegRules).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedSC, nil
}
