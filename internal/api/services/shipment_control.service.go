package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/shipmentcontrol"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
)

// ShipmentControlService is the service for shipment control settings.
type ShipmentControlService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewShipmentControlService creates a new shipment control service.
func NewShipmentControlService(s *api.Server) *ShipmentControlService {
	return &ShipmentControlService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetShipmentControl creates a new shipment control settings for an organization.
func (r *ShipmentControlService) GetShipmentControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.ShipmentControl, error) {
	shipmentControl, err := r.Client.ShipmentControl.Query().Where(
		shipmentcontrol.HasOrganizationWith(
			organization.ID(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(ctx)
	if err != nil {
		return nil, err
	}

	return shipmentControl, nil
}

// UpdateShipmentControl updates the shipment control settings for an organization.
func (r *ShipmentControlService) UpdateShipmentControl(ctx context.Context, sc *ent.ShipmentControl) (*ent.ShipmentControl, error) {
	updatedEntity := new(ent.ShipmentControl)
	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateShipmentControlEntity(ctx, tx, sc)
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

func (r *ShipmentControlService) updateShipmentControlEntity(
	ctx context.Context, tx *ent.Tx, sc *ent.ShipmentControl,
) (*ent.ShipmentControl, error) {
	updateOp := tx.ShipmentControl.UpdateOneID(sc.ID).
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
		SetEnforceHazmatSegRules(sc.EnforceHazmatSegRules)

	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
