package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func (s *service) deckHeightFeetForShipment(
	ctx context.Context,
	entity *shipment.Shipment,
) float64 {
	if entity.TrailerTypeID.IsNil() || s.equipmentTypeRepo == nil {
		return 0
	}

	equipmentType, err := s.equipmentTypeRepo.GetByID(
		ctx,
		repositories.GetEquipmentTypeByIDRequest{
			ID: entity.TrailerTypeID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
	if err != nil {
		s.l.Warn("failed to load trailer type for envelope derivation", zap.Error(err))
		return 0
	}

	return equipmentType.DeckHeightFeet()
}

func (s *service) applyShipmentEnvelope(
	ctx context.Context,
	entity *shipment.Shipment,
) {
	if entity == nil {
		return
	}

	deckHeightFeet := s.deckHeightFeetForShipment(ctx, entity)

	entity.ApplyEnvelope(shipment.ComputeEnvelope(entity.Commodities, deckHeightFeet))
}
