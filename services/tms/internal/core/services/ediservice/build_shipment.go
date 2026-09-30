package ediservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (s *Service) buildMappingPreview(
	ctx context.Context,
	partner *edi.EDIPartner,
	payload edi.LoadTenderPayload,
) (*MappingPreview, error) {
	required := payload.RequiredMappingEntityIDs
	sourceIDs := flattenRequiredIDs(required)
	items, err := s.mappingProfileRepo.GetMappingItems(ctx, repositories.GetMappingItemsRequest{
		PartnerID: partner.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: partner.OrganizationID,
			BuID:  partner.BusinessUnitID,
		},
		EntityTypes: requiredEntityTypes(required),
		SourceIDs:   sourceIDs,
	})
	if err != nil {
		return nil, err
	}

	index := mappingIndex(items)
	sourceLabels := sourceLabelIndex(&payload)

	all := make([]edi.MappingResolution, 0, len(sourceIDs))
	for _, entityType := range requiredEntityTypes(required) {
		ids := append([]pulid.ID(nil), required[entityType]...)
		slices.Sort(ids)
		for _, sourceID := range ids {
			resolution := edi.MappingResolution{
				EntityType:  entityType,
				SourceID:    sourceID,
				SourceLabel: sourceLabels[entityType][sourceID],
			}
			if item := index[entityType][sourceID]; item != nil && item.TargetID.IsNotNil() {
				resolution.SourceLabel = stringutils.FirstNonEmpty(
					item.SourceLabel,
					resolution.SourceLabel,
				)
				resolution.TargetID = item.TargetID
				resolution.TargetLabel = item.TargetLabel
				resolution.Resolved = true
			}
			all = append(all, resolution)
		}
	}

	resolved := make([]edi.MappingResolution, 0, len(all))
	unresolved := make([]edi.MappingResolution, 0)
	for _, resolution := range all {
		if resolution.Resolved {
			resolved = append(resolved, resolution)
			continue
		}
		unresolved = append(unresolved, resolution)
	}

	return &MappingPreview{
		Resolved:   resolved,
		Unresolved: unresolved,
		All:        all,
	}, nil
}

//nolint:funlen // Shipment reconstruction maps each payload section directly into the shipment aggregate.
func (s *Service) buildTargetShipment(
	transfer *edi.EDITransfer,
	businessUnitID pulid.ID,
	resolutions []edi.MappingResolution,
	approverID pulid.ID,
) (*shipment.Shipment, error) {
	mappings := resolutionIndex(resolutions)
	payload := transfer.TenderPayload

	customerID, ok := mappedID(mappings, edi.MappingEntityTypeCustomer, payload.CustomerID)
	if !ok {
		return nil, fmt.Errorf("customer mapping is missing for %s", payload.CustomerID)
	}
	serviceTypeID, ok := mappedID(mappings, edi.MappingEntityTypeServiceType, payload.ServiceTypeID)
	if !ok {
		return nil, fmt.Errorf("service type mapping is missing for %s", payload.ServiceTypeID)
	}
	formulaTemplateID, ok := mappedID(
		mappings,
		edi.MappingEntityTypeFormulaTemplate,
		payload.FormulaTemplateID,
	)
	if !ok {
		return nil, fmt.Errorf(
			"formula template mapping is missing for %s",
			payload.FormulaTemplateID,
		)
	}

	target := &shipment.Shipment{
		BusinessUnitID: businessUnitID,
		OrganizationID: transfer.TargetOrganizationID,
		ServiceTypeID:  serviceTypeID,
		ShipmentTypeID: optionalMappedID(
			mappings,
			edi.MappingEntityTypeShipmentType,
			payload.ShipmentTypeID,
		),
		CustomerID:          customerID,
		EnteredByID:         approverID,
		FormulaTemplateID:   formulaTemplateID,
		Status:              shipment.StatusNew,
		TenderStatus:        new(shipment.TenderStatusAccepted),
		EntryMethod:         shipment.EntryMethodEDI,
		BOL:                 payload.BOL,
		Pieces:              payload.Pieces,
		Weight:              payload.Weight,
		TemperatureMin:      payload.TemperatureMin,
		TemperatureMax:      payload.TemperatureMax,
		FreightChargeAmount: payload.FreightChargeAmount,
		OtherChargeAmount:   payload.OtherChargeAmount,
		BaseRate:            payload.BaseRate,
		TotalChargeAmount:   payload.TotalChargeAmount,
		RatingUnit:          payload.RatingUnit,
		Moves:               make([]*shipment.ShipmentMove, 0, len(payload.Moves)),
		Commodities:         make([]*shipment.ShipmentCommodity, 0, len(payload.Commodities)),
		AdditionalCharges:   make([]*shipment.AdditionalCharge, 0, len(payload.AdditionalCharges)),
	}

	for _, move := range payload.Moves {
		targetMove := &shipment.ShipmentMove{
			BusinessUnitID: businessUnitID,
			OrganizationID: transfer.TargetOrganizationID,
			Status:         shipment.MoveStatusNew,
			Loaded:         move.Loaded,
			Sequence:       move.Sequence,
			Distance:       move.Distance,
			Stops:          make([]*shipment.Stop, 0, len(move.Stops)),
		}
		for idx := range move.Stops {
			stop := move.Stops[idx]
			locationID, locationOK := mappedID(
				mappings,
				edi.MappingEntityTypeLocation,
				stop.LocationID,
			)
			if !locationOK {
				return nil, fmt.Errorf("location mapping is missing for %s", stop.LocationID)
			}
			targetMove.Stops = append(targetMove.Stops, &shipment.Stop{
				BusinessUnitID:       businessUnitID,
				OrganizationID:       transfer.TargetOrganizationID,
				LocationID:           locationID,
				Status:               shipment.StopStatusNew,
				Type:                 shipment.StopType(stop.Type),
				ScheduleType:         shipment.StopScheduleType(stop.ScheduleType),
				Sequence:             stop.Sequence,
				Pieces:               stop.Pieces,
				Weight:               stop.Weight,
				ScheduledWindowStart: stop.ScheduledWindowStart,
				ScheduledWindowEnd:   stop.ScheduledWindowEnd,
				AddressLine:          stop.AddressLine,
			})
		}
		target.Moves = append(target.Moves, targetMove)
	}

	for _, commodity := range payload.Commodities {
		commodityID, commodityOK := mappedID(
			mappings,
			edi.MappingEntityTypeCommodity,
			commodity.CommodityID,
		)
		if !commodityOK {
			return nil, fmt.Errorf("commodity mapping is missing for %s", commodity.CommodityID)
		}
		target.Commodities = append(target.Commodities, &shipment.ShipmentCommodity{
			BusinessUnitID: businessUnitID,
			OrganizationID: transfer.TargetOrganizationID,
			CommodityID:    commodityID,
			Weight:         commodity.Weight,
			Pieces:         commodity.Pieces,
		})
	}

	for _, charge := range payload.AdditionalCharges {
		chargeID, chargeOK := mappedID(
			mappings,
			edi.MappingEntityTypeAccessorialCharge,
			charge.AccessorialChargeID,
		)
		if !chargeOK {
			return nil, fmt.Errorf(
				"accessorial charge mapping is missing for %s",
				charge.AccessorialChargeID,
			)
		}
		target.AdditionalCharges = append(target.AdditionalCharges, &shipment.AdditionalCharge{
			BusinessUnitID:      businessUnitID,
			OrganizationID:      transfer.TargetOrganizationID,
			AccessorialChargeID: chargeID,
			Method:              accessorialcharge.Method(charge.Method),
			Amount:              charge.Amount,
			Unit:                charge.Unit,
		})
	}

	return target, nil
}
