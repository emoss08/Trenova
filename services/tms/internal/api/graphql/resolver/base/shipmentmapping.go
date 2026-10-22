package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func ShipmentToModel(
	ctx context.Context,
	entity *shipmentdomain.Shipment,
) (*gqlmodel.Shipment, error) {
	if entity == nil {
		return nil, nil //nolint:nilnil // a missing shipment maps to a null field
	}
	moves := shipmentMovesToModel(entity.Moves)
	additionalCharges := shipmentAdditionalChargesToModel(entity.AdditionalCharges)
	commodities, err := ShipmentCommoditiesToModel(entity.Commodities)
	if err != nil {
		return nil, err
	}
	tariffInputs, err := TariffInputsToModel(entity.TariffInputs)
	if err != nil {
		return nil, err
	}
	ratingDetail, err := ratingDetailToModel(entity.RatingDetail)
	if err != nil {
		return nil, err
	}
	model := &gqlmodel.Shipment{
		ID:                     entity.ID.String(),
		BusinessUnitID:         entity.BusinessUnitID.String(),
		OrganizationID:         entity.OrganizationID.String(),
		SourceDocumentID:       StringPtrFromValue(entity.SourceDocumentID),
		ServiceTypeID:          entity.ServiceTypeID.String(),
		ShipmentTypeID:         entity.ShipmentTypeID.String(),
		CustomerID:             entity.CustomerID.String(),
		TractorTypeID:          IDPtr(entity.TractorTypeID),
		TrailerTypeID:          IDPtr(entity.TrailerTypeID),
		OwnerID:                IDPtr(entity.OwnerID),
		EnteredByID:            IDPtr(entity.EnteredByID),
		CanceledByID:           IDPtr(entity.CanceledByID),
		FormulaTemplateID:      IDPtr(entity.FormulaTemplateID),
		ConsolidationGroupID:   IDPtr(entity.ConsolidationGroupID),
		OrderID:                IDPtr(entity.OrderID),
		Stage:                  entity.Stage(),
		Status:                 gqlmodel.ShipmentStatus(entity.Status),
		TenderStatus:           tenderStatusToModel(entity.TenderStatus),
		EntryMethod:            entryMethodToModel(entity.EntryMethod),
		ProNumber:              entity.ProNumber,
		BOL:                    StringPtrFromValue(entity.BOL),
		CancelReason:           entity.CancelReason,
		OtherChargeAmount:      nullDecimalString(entity.OtherChargeAmount),
		FreightChargeAmount:    nullDecimalString(entity.FreightChargeAmount),
		TariffInputs:           tariffInputs,
		FreightLines:           FreightLinesToModel(entity.FreightLines),
		TotalChargeAmount:      nullDecimalString(entity.TotalChargeAmount),
		Pieces:                 IntPtr(entity.Pieces),
		Weight:                 IntPtr(entity.Weight),
		TemperatureMin:         intPtrFromInt16(entity.TemperatureMin),
		TemperatureMax:         intPtrFromInt16(entity.TemperatureMax),
		ActualDeliveryDate:     IntPtr(entity.ActualDeliveryDate),
		ActualShipDate:         IntPtr(entity.ActualShipDate),
		CanceledAt:             IntPtr(entity.CanceledAt),
		BillingTransferStatus:  StringPtrFromValue(string(entity.BillingTransferStatus)),
		TransferredToBillingAt: IntPtr(entity.TransferredToBillingAt),
		MarkedReadyToBillAt:    IntPtr(entity.MarkedReadyToBillAt),
		BilledAt:               IntPtr(entity.BilledAt),
		FuelSurchargeLocked:    entity.FuelSurchargeLocked,
		RatingDetail:           ratingDetail,
		AutoRated:              entity.AutoRated,
		AutoRatedAt:            IntPtr(entity.AutoRatedAt),
		RateAgreementID:        IDPtrFromPtr(entity.RateAgreementID),
		RateAgreementRuleID:    IDPtrFromPtr(entity.RateAgreementRuleID),
		RateQuoteID:            IDPtrFromPtr(entity.RateQuoteID),
		RateOverrideAmount:     NullDecimalStringPtr(entity.RateOverrideAmount),
		RateOverrideReason:     StringPtrFromValue(entity.RateOverrideReason),
		RateOverrideAt:         IntPtr(entity.RateOverrideAt),
		RateLocked:             entity.RateLocked,
		Version:                int(entity.Version),
		CreatedAt:              int(entity.CreatedAt),
		UpdatedAt:              int(entity.UpdatedAt),
		Moves:                  moves,
		AdditionalCharges:      additionalCharges,
		Commodities:            commodities,
		Customer:               shipmentCustomerToModel(entity.Customer),
		BillToCustomer:         shipmentCustomerToModel(entity.BillToCustomer),
		BillToCustomerID:       IDPtrFromPtr(entity.BillToCustomerID),
		FreightTerms:           entity.FreightTerms,
		Owner:                  entity.Owner,
		FormulaTemplate:        shipmentFormulaTemplateToModel(entity.FormulaTemplate),
		ChargeAllocations:      chargeAllocationsOrEmpty(entity.ChargeAllocations),
	}
	model.BillingSplitSummary, err = billingSplitSummaryToModel(ctx, entity)
	if err != nil {
		return nil, err
	}
	if model.FreightTerms == "" {
		model.FreightTerms = shipmentdomain.FreightTermsPrepaid
	}
	return model, nil
}

func chargeAllocationsOrEmpty(
	rows []*shipmentdomain.ChargeAllocation,
) []*shipmentdomain.ChargeAllocation {
	if rows == nil {
		return []*shipmentdomain.ChargeAllocation{}
	}
	return rows
}

func shipmentMovesToModel(entities []*shipmentdomain.ShipmentMove) []*gqlmodel.ShipmentMove {
	moves := make([]*gqlmodel.ShipmentMove, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		stops := shipmentStopsToModel(entity.Stops)
		assignment := shipmentAssignmentToModel(entity.Assignment)
		coverageType := entity.CoverageType
		if coverageType == "" {
			coverageType = shipmentdomain.MoveCoverageTypeUnassigned
		}
		moves = append(moves, &gqlmodel.ShipmentMove{
			ID:                     IDPtr(entity.ID),
			BusinessUnitID:         entity.BusinessUnitID.String(),
			OrganizationID:         entity.OrganizationID.String(),
			ShipmentID:             IDPtr(entity.ShipmentID),
			Status:                 gqlmodel.MoveStatus(entity.Status),
			CoverageType:           coverageType,
			CarrierAssignment:      entity.CarrierAssignment,
			Loaded:                 entity.Loaded,
			Sequence:               int(entity.Sequence),
			Distance:               entity.Distance,
			DistanceSource:         StringPtrFromValue(entity.DistanceSource),
			DistanceProvider:       StringPtrFromValue(entity.DistanceProvider),
			DistanceCalculatedAt:   IntPtr(entity.DistanceCalculatedAt),
			DistanceRouteSignature: StringPtrFromValue(entity.DistanceRouteSignature),
			DistanceDataVersion:    StringPtrFromValue(entity.DistanceDataVersion),
			DistanceRoutingType:    StringPtrFromValue(entity.DistanceRoutingType),
			DistanceUnits:          StringPtrFromValue(entity.DistanceUnits),
			DistanceMetadata:       entity.DistanceMetadata,
			Version:                int(entity.Version),
			CreatedAt:              int(entity.CreatedAt),
			UpdatedAt:              int(entity.UpdatedAt),
			Stops:                  stops,
			Assignment:             assignment,
		})
	}
	return moves
}

func shipmentStopsToModel(entities []*shipmentdomain.Stop) []*gqlmodel.ShipmentStop {
	stops := make([]*gqlmodel.ShipmentStop, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		stops = append(stops, &gqlmodel.ShipmentStop{
			ID:                     IDPtr(entity.ID),
			BusinessUnitID:         entity.BusinessUnitID.String(),
			OrganizationID:         entity.OrganizationID.String(),
			ShipmentMoveID:         IDPtr(entity.ShipmentMoveID),
			LocationID:             entity.LocationID.String(),
			Status:                 gqlmodel.StopStatus(entity.Status),
			Type:                   gqlmodel.StopType(entity.Type),
			ScheduleType:           gqlmodel.StopScheduleType(entity.ScheduleType),
			Sequence:               int(entity.Sequence),
			Pieces:                 IntPtr(entity.Pieces),
			Weight:                 IntPtr(entity.Weight),
			ScheduledWindowStart:   int(entity.ScheduledWindowStart),
			ScheduledWindowEnd:     IntPtr(entity.ScheduledWindowEnd),
			ActualArrival:          IntPtr(entity.ActualArrival),
			ActualDeparture:        IntPtr(entity.ActualDeparture),
			CountLateOverride:      entity.CountLateOverride,
			CountDetentionOverride: entity.CountDetentionOverride,
			AddressLine:            entity.AddressLine,
			Version:                int(entity.Version),
			CreatedAt:              int(entity.CreatedAt),
			UpdatedAt:              int(entity.UpdatedAt),
			Location:               entity.Location,
		})
	}
	return stops
}

func shipmentAssignmentToModel(entity *shipmentdomain.Assignment) *gqlmodel.ShipmentAssignment {
	if entity == nil {
		return nil
	}
	return &gqlmodel.ShipmentAssignment{
		ID:                IDPtr(entity.ID),
		BusinessUnitID:    entity.BusinessUnitID.String(),
		OrganizationID:    entity.OrganizationID.String(),
		ShipmentMoveID:    IDPtr(entity.ShipmentMoveID),
		PrimaryWorkerID:   IDPtrFromPulidPtr(entity.PrimaryWorkerID),
		TractorID:         IDPtrFromPulidPtr(entity.TractorID),
		TrailerID:         IDPtrFromPulidPtr(entity.TrailerID),
		SecondaryWorkerID: IDPtrFromPulidPtr(entity.SecondaryWorkerID),
		Status:            gqlmodel.AssignmentStatus(entity.Status),
		ArchivedAt:        IntPtr(entity.ArchivedAt),
		Version:           int(entity.Version),
		CreatedAt:         int(entity.CreatedAt),
		UpdatedAt:         int(entity.UpdatedAt),
		Tractor:           entity.Tractor,
		Trailer:           entity.Trailer,
		PrimaryWorker:     entity.PrimaryWorker,
		SecondaryWorker:   entity.SecondaryWorker,
	}
}

func ShipmentAdditionalChargeToModel(
	entity *shipmentdomain.AdditionalCharge,
) *gqlmodel.ShipmentAdditionalCharge {
	if entity == nil {
		return nil
	}
	return &gqlmodel.ShipmentAdditionalCharge{
		ID:                     IDPtr(entity.ID),
		BusinessUnitID:         entity.BusinessUnitID.String(),
		OrganizationID:         entity.OrganizationID.String(),
		ShipmentID:             entity.ShipmentID.String(),
		AccessorialChargeID:    entity.AccessorialChargeID.String(),
		IsSystemGenerated:      entity.IsSystemGenerated,
		Method:                 string(entity.Method),
		Amount:                 entity.Amount.String(),
		Unit:                   int(entity.Unit),
		FuelSurchargeProgramID: IDPtrFromPulidPtr(entity.FuelSurchargeProgramID),
		FuelSurchargeDetail:    fuelSurchargeDetailToMap(entity.FuelSurchargeDetail),
		IsDetention:            entity.IsDetention,
		IsTariff:               entity.IsTariff,
		Label:                  entity.Label,
		Quantity:               NullDecimalStringPtr(entity.Quantity),
		Rate:                   NullDecimalStringPtr(entity.Rate),
		RateUnit:               entity.RateUnit,
		Version:                int(entity.Version),
		CreatedAt:              int(entity.CreatedAt),
		UpdatedAt:              int(entity.UpdatedAt),
		AccessorialCharge:      ShipmentAccessorialChargeToModel(entity.AccessorialCharge),
	}
}

func shipmentAdditionalChargesToModel(
	entities []*shipmentdomain.AdditionalCharge,
) []*gqlmodel.ShipmentAdditionalCharge {
	charges := make([]*gqlmodel.ShipmentAdditionalCharge, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		charges = append(charges, ShipmentAdditionalChargeToModel(entity))
	}
	return charges
}

func ShipmentCommoditiesToModel(
	entities []*shipmentdomain.ShipmentCommodity,
) ([]*gqlmodel.ShipmentCommodity, error) {
	commodities := make([]*gqlmodel.ShipmentCommodity, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		commodities = append(commodities, &gqlmodel.ShipmentCommodity{
			ID:             IDPtr(entity.ID),
			BusinessUnitID: entity.BusinessUnitID.String(),
			OrganizationID: entity.OrganizationID.String(),
			ShipmentID:     entity.ShipmentID.String(),
			CommodityID:    entity.CommodityID.String(),
			Pieces:         int(entity.Pieces),
			Weight:         int(entity.Weight),
			LengthFeet:     entity.LengthFeet,
			WidthFeet:      entity.WidthFeet,
			HeightFeet:     entity.HeightFeet,
			Version:        int(entity.Version),
			CreatedAt:      int(entity.CreatedAt),
			UpdatedAt:      int(entity.UpdatedAt),
			Commodity:      shipmentCommodityDetailToModel(entity.Commodity),
		})
	}
	return commodities, nil
}

func nullDecimalString(value decimal.NullDecimal) string {
	if !value.Valid {
		return decimal.Zero.String()
	}
	return value.Decimal.String()
}

func ratingDetailToModel(
	detail *shipmentdomain.RatingDetail,
) (*gqlmodel.ShipmentRatingDetail, error) {
	if detail == nil {
		return nil, nil //nolint:nilnil // a shipment never rated has no detail
	}
	inputs, err := TariffInputsToModel(detail.Inputs)
	if err != nil {
		return nil, err
	}
	trace, err := TariffTraceToModel(detail.Trace)
	if err != nil {
		return nil, err
	}
	skipped := make([]*gqlmodel.ShipmentRatingSkip, 0, len(detail.Skipped))
	for _, skip := range detail.Skipped {
		skipped = append(
			skipped,
			&gqlmodel.ShipmentRatingSkip{Code: skip.Code, Source: skip.Source},
		)
	}
	return &gqlmodel.ShipmentRatingDetail{
		FormulaTemplateID:   detail.FormulaTemplateID,
		FormulaTemplateName: detail.FormulaTemplateName,
		RatedAt:             int(detail.RatedAt),
		VersionNumber:       int(detail.VersionNumber),
		Bill:                TariffBillToModel(detail.Bill),
		Refusal:             detail.Refusal,
		Inputs:              inputs,
		Skipped:             skipped,
		Trace:               trace,
		RateQuoteID:         detail.RateQuoteID,
		AgreementID:         detail.AgreementID,
		AgreementName:       detail.AgreementName,
		RuleID:              detail.RuleID,
		RuleLabel:           detail.RuleLabel,
		Source:              detail.Source,
		Explanation:         detail.Explanation,
	}, nil
}

func tenderStatusToModel(status *shipmentdomain.TenderStatus) *gqlmodel.ShipmentTenderStatus {
	if status == nil {
		return nil
	}
	value := gqlmodel.ShipmentTenderStatus(*status)
	return &value
}

func entryMethodToModel(method shipmentdomain.EntryMethod) *gqlmodel.ShipmentEntryMethod {
	if method == "" {
		return nil
	}
	value := gqlmodel.ShipmentEntryMethod(method)
	return &value
}

func StringPtrFromValue(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func intPtrFromInt16(value *int16) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func IDPtrFromPulidPtr(value *pulid.ID) *string {
	if value == nil || value.IsNil() {
		return nil
	}
	converted := value.String()
	return &converted
}

func PulidPtrFromOptionalString(value *string) (*pulid.ID, error) {
	if value == nil || *value == "" {
		return nil, nil //nolint:nilnil // an omitted ID stays unset
	}
	parsed, err := pulid.MustParse(*value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
