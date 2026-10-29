package shipmentmapping

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/tariffmapping"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
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
	tariffInputs, err := tariffmapping.TariffInputsToModel(entity.TariffInputs)
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
		SourceDocumentID:       gqlscalar.StringPtrFromValue(entity.SourceDocumentID),
		ServiceTypeID:          entity.ServiceTypeID.String(),
		ShipmentTypeID:         entity.ShipmentTypeID.String(),
		CustomerID:             entity.CustomerID.String(),
		TractorTypeID:          gqlid.IDPtr(entity.TractorTypeID),
		TrailerTypeID:          gqlid.IDPtr(entity.TrailerTypeID),
		OwnerID:                gqlid.IDPtr(entity.OwnerID),
		EnteredByID:            gqlid.IDPtr(entity.EnteredByID),
		CanceledByID:           gqlid.IDPtr(entity.CanceledByID),
		FormulaTemplateID:      gqlid.IDPtr(entity.FormulaTemplateID),
		ConsolidationGroupID:   gqlid.IDPtr(entity.ConsolidationGroupID),
		OrderID:                gqlid.IDPtr(entity.OrderID),
		Stage:                  shipmentvocab.StageOf(entity.Status),
		Status:                 gqlmodel.ShipmentStatus(entity.Status),
		TenderStatus:           tenderStatusToModel(entity.TenderStatus),
		EntryMethod:            entryMethodToModel(entity.EntryMethod),
		ProNumber:              entity.ProNumber,
		BOL:                    gqlscalar.StringPtrFromValue(entity.BOL),
		CancelReason:           entity.CancelReason,
		OtherChargeAmount:      nullDecimalString(entity.OtherChargeAmount),
		FreightChargeAmount:    nullDecimalString(entity.FreightChargeAmount),
		TariffInputs:           tariffInputs,
		FreightLines:           tariffmapping.FreightLinesToModel(entity.FreightLines),
		TotalChargeAmount:      nullDecimalString(entity.TotalChargeAmount),
		Pieces:                 gqlscalar.IntPtr(entity.Pieces),
		Weight:                 gqlscalar.IntPtr(entity.Weight),
		TemperatureMin:         intPtrFromInt16(entity.TemperatureMin),
		TemperatureMax:         intPtrFromInt16(entity.TemperatureMax),
		ActualDeliveryDate:     gqlscalar.IntPtr(entity.ActualDeliveryDate),
		ActualShipDate:         gqlscalar.IntPtr(entity.ActualShipDate),
		CanceledAt:             gqlscalar.IntPtr(entity.CanceledAt),
		BillingTransferStatus:  gqlscalar.StringPtrFromValue(string(entity.BillingTransferStatus)),
		TransferredToBillingAt: gqlscalar.IntPtr(entity.TransferredToBillingAt),
		MarkedReadyToBillAt:    gqlscalar.IntPtr(entity.MarkedReadyToBillAt),
		BilledAt:               gqlscalar.IntPtr(entity.BilledAt),
		FuelSurchargeLocked:    entity.FuelSurchargeLocked,
		RatingDetail:           ratingDetail,
		AutoRated:              entity.AutoRated,
		AutoRatedAt:            gqlscalar.IntPtr(entity.AutoRatedAt),
		RateAgreementID:        gqlid.IDPtrFromPtr(entity.RateAgreementID),
		RateAgreementRuleID:    gqlid.IDPtrFromPtr(entity.RateAgreementRuleID),
		RateQuoteID:            gqlid.IDPtrFromPtr(entity.RateQuoteID),
		RateOverrideAmount:     gqldecimal.NullDecimalStringPtr(entity.RateOverrideAmount),
		RateOverrideReason:     gqlscalar.StringPtrFromValue(entity.RateOverrideReason),
		RateOverrideAt:         gqlscalar.IntPtr(entity.RateOverrideAt),
		RateLocked:             entity.RateLocked,
		Version:                int(entity.Version),
		CreatedAt:              int(entity.CreatedAt),
		UpdatedAt:              int(entity.UpdatedAt),
		Moves:                  moves,
		AdditionalCharges:      additionalCharges,
		Commodities:            commodities,
		Customer:               shipmentCustomerToModel(entity.Customer),
		BillToCustomer:         shipmentCustomerToModel(entity.BillToCustomer),
		BillToCustomerID:       gqlid.IDPtrFromPtr(entity.BillToCustomerID),
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
		model.FreightTerms = shipmentvocab.FreightTermsPrepaid
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
			coverageType = shipmentvocab.MoveCoverageTypeUnassigned
		}
		moves = append(moves, &gqlmodel.ShipmentMove{
			ID:                     gqlid.IDPtr(entity.ID),
			BusinessUnitID:         entity.BusinessUnitID.String(),
			OrganizationID:         entity.OrganizationID.String(),
			ShipmentID:             gqlid.IDPtr(entity.ShipmentID),
			Status:                 gqlmodel.MoveStatus(entity.Status),
			CoverageType:           coverageType,
			CarrierAssignment:      entity.CarrierAssignment,
			Loaded:                 entity.Loaded,
			Sequence:               int(entity.Sequence),
			Distance:               entity.Distance,
			DistanceSource:         gqlscalar.StringPtrFromValue(entity.DistanceSource),
			DistanceProvider:       gqlscalar.StringPtrFromValue(entity.DistanceProvider),
			DistanceCalculatedAt:   gqlscalar.IntPtr(entity.DistanceCalculatedAt),
			DistanceRouteSignature: gqlscalar.StringPtrFromValue(entity.DistanceRouteSignature),
			DistanceDataVersion:    gqlscalar.StringPtrFromValue(entity.DistanceDataVersion),
			DistanceRoutingType:    gqlscalar.StringPtrFromValue(entity.DistanceRoutingType),
			DistanceUnits:          gqlscalar.StringPtrFromValue(entity.DistanceUnits),
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
			ID:                     gqlid.IDPtr(entity.ID),
			BusinessUnitID:         entity.BusinessUnitID.String(),
			OrganizationID:         entity.OrganizationID.String(),
			ShipmentMoveID:         gqlid.IDPtr(entity.ShipmentMoveID),
			LocationID:             entity.LocationID.String(),
			Status:                 gqlmodel.StopStatus(entity.Status),
			Type:                   gqlmodel.StopType(entity.Type),
			ScheduleType:           gqlmodel.StopScheduleType(entity.ScheduleType),
			Sequence:               int(entity.Sequence),
			Pieces:                 gqlscalar.IntPtr(entity.Pieces),
			Weight:                 gqlscalar.IntPtr(entity.Weight),
			ScheduledWindowStart:   int(entity.ScheduledWindowStart),
			ScheduledWindowEnd:     gqlscalar.IntPtr(entity.ScheduledWindowEnd),
			ActualArrival:          gqlscalar.IntPtr(entity.ActualArrival),
			ActualDeparture:        gqlscalar.IntPtr(entity.ActualDeparture),
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
		ID:                gqlid.IDPtr(entity.ID),
		BusinessUnitID:    entity.BusinessUnitID.String(),
		OrganizationID:    entity.OrganizationID.String(),
		ShipmentMoveID:    gqlid.IDPtr(entity.ShipmentMoveID),
		PrimaryWorkerID:   gqlid.IDPtrFromPulidPtr(entity.PrimaryWorkerID),
		TractorID:         gqlid.IDPtrFromPulidPtr(entity.TractorID),
		TrailerID:         gqlid.IDPtrFromPulidPtr(entity.TrailerID),
		SecondaryWorkerID: gqlid.IDPtrFromPulidPtr(entity.SecondaryWorkerID),
		Status:            gqlmodel.AssignmentStatus(entity.Status),
		ArchivedAt:        gqlscalar.IntPtr(entity.ArchivedAt),
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
		ID:                     gqlid.IDPtr(entity.ID),
		BusinessUnitID:         entity.BusinessUnitID.String(),
		OrganizationID:         entity.OrganizationID.String(),
		ShipmentID:             entity.ShipmentID.String(),
		AccessorialChargeID:    entity.AccessorialChargeID.String(),
		IsSystemGenerated:      entity.IsSystemGenerated,
		Method:                 string(entity.Method),
		Amount:                 entity.Amount.String(),
		Unit:                   int(entity.Unit),
		FuelSurchargeProgramID: gqlid.IDPtrFromPulidPtr(entity.FuelSurchargeProgramID),
		FuelSurchargeDetail:    fuelSurchargeDetailToMap(entity.FuelSurchargeDetail),
		IsDetention:            entity.IsDetention,
		IsTariff:               entity.IsTariff,
		Label:                  entity.Label,
		Quantity:               gqldecimal.NullDecimalStringPtr(entity.Quantity),
		Rate:                   gqldecimal.NullDecimalStringPtr(entity.Rate),
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
			ID:             gqlid.IDPtr(entity.ID),
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
	inputs, err := tariffmapping.TariffInputsToModel(detail.Inputs)
	if err != nil {
		return nil, err
	}
	trace, err := tariffmapping.TariffTraceToModel(detail.Trace)
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
		Bill:                tariffmapping.TariffBillToModel(detail.Bill),
		Refusal:             detail.Refusal,
		Choices:             tariffmapping.TariffChoicesToModel(detail.Choices),
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
		AwaitingReview:      detail.AwaitsReview(),
		ReviewApproval:      ratingReviewApprovalToModel(detail.ReviewApproval),
	}, nil
}

func ratingReviewApprovalToModel(
	approval *shipmentdomain.RatingReviewApproval,
) *gqlmodel.ShipmentRatingReviewApproval {
	if approval == nil {
		return nil
	}
	reviews := approval.Reviews
	if reviews == nil {
		reviews = []string{}
	}
	return &gqlmodel.ShipmentRatingReviewApproval{
		Reviews:      reviews,
		ApprovedByID: approval.ApprovedByID.String(),
		ApprovedAt:   int(approval.ApprovedAt),
	}
}

func tenderStatusToModel(status *shipmentvocab.TenderStatus) *gqlmodel.ShipmentTenderStatus {
	if status == nil {
		return nil
	}
	value := gqlmodel.ShipmentTenderStatus(*status)
	return &value
}

func entryMethodToModel(method shipmentvocab.EntryMethod) *gqlmodel.ShipmentEntryMethod {
	if method == "" {
		return nil
	}
	value := gqlmodel.ShipmentEntryMethod(method)
	return &value
}

func intPtrFromInt16(value *int16) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}
