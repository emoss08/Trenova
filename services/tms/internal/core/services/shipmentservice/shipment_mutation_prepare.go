package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *service) normalizeAdditionalChargeSystemGenerationForCreate(
	entity *shipment.Shipment,
) {
	if entity == nil {
		return
	}

	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		charge.IsSystemGenerated = false
		charge.DetentionOccurrenceID = nil
		charge.RateAgreementAccessorialID = nil
		charge.FuelSurchargeProgramID = nil
	}
}

func (s *service) restoreSystemOwnedAdditionalChargeFields(
	original *shipment.Shipment,
	updated *shipment.Shipment,
) {
	if updated == nil {
		return
	}

	var originalCharges []*shipment.AdditionalCharge
	if original != nil {
		originalCharges = original.AdditionalCharges
	}

	shipment.RestoreSystemOwnedCharges(originalCharges, updated.AdditionalCharges)
}

func (s *service) restoreAssignmentsForExistingMoves(
	original *shipment.Shipment,
	updated *shipment.Shipment,
) {
	if original == nil || updated == nil {
		return
	}

	originalMoves := make(map[pulid.ID]*shipment.ShipmentMove, len(original.Moves))
	for _, move := range original.Moves {
		if move == nil || move.ID.IsNil() {
			continue
		}

		originalMoves[move.ID] = move
	}

	for _, move := range updated.Moves {
		if move == nil || move.ID.IsNil() || move.Assignment != nil {
			continue
		}

		originalMove := originalMoves[move.ID]
		if originalMove == nil || originalMove.Assignment == nil {
			continue
		}

		move.Assignment = originalMove.Assignment
	}
}

func (s *service) hydrateShipmentCommodityDetails(
	ctx context.Context,
	entity *shipment.Shipment,
) error {
	if entity == nil || len(entity.Commodities) == 0 {
		return nil
	}

	commodityIDs := uniqueShipmentCommodityIDs(entity.Commodities)
	if len(commodityIDs) == 0 || s.commodityRepo == nil {
		return nil
	}

	commodities, err := s.commodityRepo.GetByIDs(ctx, repositories.GetCommoditiesByIDsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		CommodityIDs: commodityIDs,
	})
	if err != nil {
		return err
	}

	commodityMap := make(map[pulid.ID]*commodity.Commodity, len(commodities))
	for _, item := range commodities {
		if item == nil || item.ID.IsNil() {
			continue
		}
		commodityMap[item.ID] = item
	}

	for _, shipmentCommodity := range entity.Commodities {
		if shipmentCommodity == nil || shipmentCommodity.CommodityID.IsNil() {
			continue
		}
		if loaded, ok := commodityMap[shipmentCommodity.CommodityID]; ok {
			shipmentCommodity.Commodity = mergeCommodityDetails(shipmentCommodity.Commodity, loaded)
		}
	}

	return nil
}

func mergeCommodityDetails(
	existing *commodity.Commodity,
	loaded *commodity.Commodity,
) *commodity.Commodity {
	if existing == nil {
		return loaded
	}

	merged := *loaded

	if existing.LinearFeetPerUnit != nil {
		merged.LinearFeetPerUnit = existing.LinearFeetPerUnit
	}

	if existing.HazardousMaterial != nil {
		merged.HazardousMaterial = existing.HazardousMaterial
	}

	if !existing.HazardousMaterialID.IsNil() {
		merged.HazardousMaterialID = existing.HazardousMaterialID
	}

	return &merged
}
