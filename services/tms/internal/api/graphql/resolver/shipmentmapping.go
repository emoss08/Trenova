package resolver

import (
	"context"
	"fmt"
	"math"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/order"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func shipmentFromInput(
	input gqlmodel.ShipmentInput,
	id pulid.ID,
	authCtx *authctx.AuthContext,
) (*shipmentdomain.Shipment, error) {
	serviceTypeID, err := pulid.MustParse(input.ServiceTypeID)
	if err != nil {
		return nil, err
	}
	shipmentTypeID, err := pulid.MustParse(input.ShipmentTypeID)
	if err != nil {
		return nil, err
	}
	customerID, err := pulid.MustParse(input.CustomerID)
	if err != nil {
		return nil, err
	}
	formulaTemplateID, err := pulid.MustParse(input.FormulaTemplateID)
	if err != nil {
		return nil, err
	}
	tractorTypeID, err := optionalID(input.TractorTypeID)
	if err != nil {
		return nil, err
	}
	trailerTypeID, err := optionalID(input.TrailerTypeID)
	if err != nil {
		return nil, err
	}
	ownerID, err := optionalID(input.OwnerID)
	if err != nil {
		return nil, err
	}
	enteredByID, err := optionalID(input.EnteredByID)
	if err != nil {
		return nil, err
	}
	canceledByID, err := optionalID(input.CanceledByID)
	if err != nil {
		return nil, err
	}
	consolidationGroupID, err := optionalID(input.ConsolidationGroupID)
	if err != nil {
		return nil, err
	}
	orderID, err := optionalID(input.OrderID)
	if err != nil {
		return nil, err
	}
	otherChargeAmount, err := nullDecimalFromInput(input.OtherChargeAmount)
	if err != nil {
		return nil, err
	}
	freightChargeAmount, err := nullDecimalFromInput(input.FreightChargeAmount)
	if err != nil {
		return nil, err
	}
	baseRate, err := nullDecimalFromInput(input.BaseRate)
	if err != nil {
		return nil, err
	}
	totalChargeAmount, err := nullDecimalFromInput(input.TotalChargeAmount)
	if err != nil {
		return nil, err
	}
	temperatureMin, err := int16PtrFromInput("temperatureMin", input.TemperatureMin)
	if err != nil {
		return nil, err
	}
	temperatureMax, err := int16PtrFromInput("temperatureMax", input.TemperatureMax)
	if err != nil {
		return nil, err
	}

	status := shipmentdomain.StatusNew
	if input.Status != nil {
		status = shipmentdomain.Status(*input.Status)
	}
	entryMethod := shipmentdomain.EntryMethodManual
	if input.EntryMethod != nil {
		entryMethod = shipmentdomain.EntryMethod(*input.EntryMethod)
	}

	entity := &shipmentdomain.Shipment{
		ID:                     id,
		BusinessUnitID:         authCtx.BusinessUnitID,
		OrganizationID:         authCtx.OrganizationID,
		ServiceTypeID:          serviceTypeID,
		ShipmentTypeID:         shipmentTypeID,
		CustomerID:             customerID,
		TractorTypeID:          tractorTypeID,
		TrailerTypeID:          trailerTypeID,
		OwnerID:                ownerID,
		EnteredByID:            enteredByID,
		CanceledByID:           canceledByID,
		FormulaTemplateID:      formulaTemplateID,
		ConsolidationGroupID:   consolidationGroupID,
		OrderID:                orderID,
		Status:                 status,
		EntryMethod:            entryMethod,
		ProNumber:              stringValue(input.ProNumber),
		BOL:                    stringValue(input.Bol),
		CancelReason:           stringValue(input.CancelReason),
		OtherChargeAmount:      otherChargeAmount,
		FreightChargeAmount:    freightChargeAmount,
		BaseRate:               baseRate,
		TotalChargeAmount:      totalChargeAmount,
		Pieces:                 int64Ptr(input.Pieces),
		Weight:                 int64Ptr(input.Weight),
		TemperatureMin:         temperatureMin,
		TemperatureMax:         temperatureMax,
		ActualDeliveryDate:     int64Ptr(input.ActualDeliveryDate),
		ActualShipDate:         int64Ptr(input.ActualShipDate),
		CanceledAt:             int64Ptr(input.CanceledAt),
		BillingTransferStatus:  shipmentdomain.BillingTransferStatus(stringValue(input.BillingTransferStatus)),
		TransferredToBillingAt: int64Ptr(input.TransferredToBillingAt),
		MarkedReadyToBillAt:    int64Ptr(input.MarkedReadyToBillAt),
		BilledAt:               int64Ptr(input.BilledAt),
		RatingUnit:             int64Value(input.RatingUnit),
		SourceDocumentID:       stringValue(input.SourceDocumentID),
	}
	if entity.RatingUnit == 0 {
		entity.RatingUnit = 1
	}
	if input.TenderStatus != nil {
		tenderStatus := shipmentdomain.TenderStatus(*input.TenderStatus)
		entity.TenderStatus = &tenderStatus
	}
	if input.RatingDetail != nil {
		entity.RatingDetail = &shipmentdomain.RatingDetail{
			FormulaTemplateID:   input.RatingDetail.FormulaTemplateID,
			FormulaTemplateName: input.RatingDetail.FormulaTemplateName,
			Expression:          input.RatingDetail.Expression,
			ResolvedVariables:   input.RatingDetail.ResolvedVariables,
			Result:              input.RatingDetail.Result,
			RatedAt:             int64(input.RatingDetail.RatedAt),
		}
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	moves, err := shipmentMovesFromInput(input.Moves, authCtx)
	if err != nil {
		return nil, err
	}
	additionalCharges, err := shipmentAdditionalChargesFromInput(input.AdditionalCharges, authCtx)
	if err != nil {
		return nil, err
	}
	commodities, err := shipmentCommoditiesFromInput(input.Commodities, authCtx)
	if err != nil {
		return nil, err
	}
	entity.Moves = moves
	entity.AdditionalCharges = additionalCharges
	entity.Commodities = commodities

	return entity, nil
}

func shipmentMovesFromInput(
	inputs []*gqlmodel.ShipmentMoveInput,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.ShipmentMove, error) {
	moves := make([]*shipmentdomain.ShipmentMove, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		id, err := optionalID(input.ID)
		if err != nil {
			return nil, err
		}
		shipmentID, err := optionalID(input.ShipmentID)
		if err != nil {
			return nil, err
		}
		status := shipmentdomain.MoveStatusNew
		if input.Status != nil {
			status = shipmentdomain.MoveStatus(*input.Status)
		}
		loaded := true
		if input.Loaded != nil {
			loaded = *input.Loaded
		}
		move := &shipmentdomain.ShipmentMove{
			ID:                     id,
			BusinessUnitID:         authCtx.BusinessUnitID,
			OrganizationID:         authCtx.OrganizationID,
			ShipmentID:             shipmentID,
			Status:                 status,
			Loaded:                 loaded,
			Sequence:               int64Value(input.Sequence),
			Distance:               input.Distance,
			DistanceSource:         stringValue(input.DistanceSource),
			DistanceProvider:       stringValue(input.DistanceProvider),
			DistanceCalculatedAt:   int64Ptr(input.DistanceCalculatedAt),
			DistanceRouteSignature: stringValue(input.DistanceRouteSignature),
			DistanceDataVersion:    stringValue(input.DistanceDataVersion),
			DistanceRoutingType:    stringValue(input.DistanceRoutingType),
			DistanceUnits:          stringValue(input.DistanceUnits),
			DistanceMetadata:       input.DistanceMetadata,
		}
		if input.Version != nil {
			move.Version = int64(*input.Version)
		}
		stops, err := shipmentStopsFromInput(input.Stops, authCtx)
		if err != nil {
			return nil, err
		}
		move.Stops = stops
		moves = append(moves, move)
	}
	return moves, nil
}

func shipmentStopsFromInput(
	inputs []*gqlmodel.ShipmentStopInput,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.Stop, error) {
	stops := make([]*shipmentdomain.Stop, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		id, err := optionalID(input.ID)
		if err != nil {
			return nil, err
		}
		shipmentMoveID, err := optionalID(input.ShipmentMoveID)
		if err != nil {
			return nil, err
		}
		locationID, err := pulid.MustParse(input.LocationID)
		if err != nil {
			return nil, err
		}
		status := shipmentdomain.StopStatusNew
		if input.Status != nil {
			status = shipmentdomain.StopStatus(*input.Status)
		}
		stopType := shipmentdomain.StopTypePickup
		if input.Type != nil {
			stopType = shipmentdomain.StopType(*input.Type)
		}
		scheduleType := shipmentdomain.StopScheduleTypeOpen
		if input.ScheduleType != nil {
			scheduleType = shipmentdomain.StopScheduleType(*input.ScheduleType)
		}
		stop := &shipmentdomain.Stop{
			ID:                     id,
			BusinessUnitID:         authCtx.BusinessUnitID,
			OrganizationID:         authCtx.OrganizationID,
			ShipmentMoveID:         shipmentMoveID,
			LocationID:             locationID,
			Status:                 status,
			Type:                   stopType,
			ScheduleType:           scheduleType,
			Sequence:               int64Value(input.Sequence),
			Pieces:                 int64Ptr(input.Pieces),
			Weight:                 int64Ptr(input.Weight),
			ScheduledWindowStart:   int64Value(input.ScheduledWindowStart),
			ScheduledWindowEnd:     int64Ptr(input.ScheduledWindowEnd),
			ActualArrival:          int64Ptr(input.ActualArrival),
			ActualDeparture:        int64Ptr(input.ActualDeparture),
			CountLateOverride:      input.CountLateOverride,
			CountDetentionOverride: input.CountDetentionOverride,
			AddressLine:            stringValue(input.AddressLine),
		}
		if input.Version != nil {
			stop.Version = int64(*input.Version)
		}
		stops = append(stops, stop)
	}
	return stops, nil
}

func shipmentAdditionalChargesFromInput(
	inputs []*gqlmodel.ShipmentAdditionalChargeInput,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.AdditionalCharge, error) {
	charges := make([]*shipmentdomain.AdditionalCharge, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		id, err := optionalID(input.ID)
		if err != nil {
			return nil, err
		}
		shipmentID, err := optionalID(input.ShipmentID)
		if err != nil {
			return nil, err
		}
		accessorialChargeID, err := pulid.MustParse(input.AccessorialChargeID)
		if err != nil {
			return nil, err
		}
		amount, err := decimalFromInput(input.Amount)
		if err != nil {
			return nil, err
		}
		unit := int16(1)
		if input.Unit != nil {
			parsedUnit, err := int16FromInput("unit", *input.Unit)
			if err != nil {
				return nil, err
			}
			unit = parsedUnit
		}
		isSystemGenerated := false
		if input.IsSystemGenerated != nil {
			isSystemGenerated = *input.IsSystemGenerated
		}
		method := accessorialcharge.MethodFlat
		if input.Method != nil {
			method = accessorialcharge.Method(*input.Method)
		}
		charge := &shipmentdomain.AdditionalCharge{
			ID:                  id,
			BusinessUnitID:      authCtx.BusinessUnitID,
			OrganizationID:      authCtx.OrganizationID,
			ShipmentID:          shipmentID,
			AccessorialChargeID: accessorialChargeID,
			IsSystemGenerated:   isSystemGenerated,
			Method:              method,
			Amount:              amount,
			Unit:                unit,
		}
		if input.Version != nil {
			charge.Version = int64(*input.Version)
		}
		charges = append(charges, charge)
	}
	return charges, nil
}

func shipmentCommoditiesFromInput(
	inputs []*gqlmodel.ShipmentCommodityInput,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.ShipmentCommodity, error) {
	commodities := make([]*shipmentdomain.ShipmentCommodity, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		id, err := optionalID(input.ID)
		if err != nil {
			return nil, err
		}
		shipmentID, err := optionalID(input.ShipmentID)
		if err != nil {
			return nil, err
		}
		commodityID, err := pulid.MustParse(input.CommodityID)
		if err != nil {
			return nil, err
		}
		commodity := &shipmentdomain.ShipmentCommodity{
			ID:             id,
			BusinessUnitID: authCtx.BusinessUnitID,
			OrganizationID: authCtx.OrganizationID,
			ShipmentID:     shipmentID,
			CommodityID:    commodityID,
			Pieces:         int64Value(input.Pieces),
			Weight:         int64Value(input.Weight),
		}
		if commodity.Pieces == 0 {
			commodity.Pieces = 1
		}
		if input.Version != nil {
			commodity.Version = int64(*input.Version)
		}
		commodities = append(commodities, commodity)
	}
	return commodities, nil
}

func shipmentToModel(entity *shipmentdomain.Shipment) (*gqlmodel.Shipment, error) {
	if entity == nil {
		return nil, nil
	}
	moves, err := shipmentMovesToModel(entity.Moves)
	if err != nil {
		return nil, err
	}
	additionalCharges, err := shipmentAdditionalChargesToModel(entity.AdditionalCharges)
	if err != nil {
		return nil, err
	}
	commodities, err := shipmentCommoditiesToModel(entity.Commodities)
	if err != nil {
		return nil, err
	}
	model := &gqlmodel.Shipment{
		ID:                     entity.ID.String(),
		BusinessUnitID:         entity.BusinessUnitID.String(),
		OrganizationID:         entity.OrganizationID.String(),
		SourceDocumentID:       stringPtrFromValue(entity.SourceDocumentID),
		ServiceTypeID:          entity.ServiceTypeID.String(),
		ShipmentTypeID:         entity.ShipmentTypeID.String(),
		CustomerID:             entity.CustomerID.String(),
		TractorTypeID:          idPtr(entity.TractorTypeID),
		TrailerTypeID:          idPtr(entity.TrailerTypeID),
		OwnerID:                idPtr(entity.OwnerID),
		EnteredByID:            idPtr(entity.EnteredByID),
		CanceledByID:           idPtr(entity.CanceledByID),
		FormulaTemplateID:      entity.FormulaTemplateID.String(),
		ConsolidationGroupID:   idPtr(entity.ConsolidationGroupID),
		OrderID:                idPtr(entity.OrderID),
		Status:                 gqlmodel.ShipmentStatus(entity.Status),
		TenderStatus:           tenderStatusToModel(entity.TenderStatus),
		EntryMethod:            entryMethodToModel(entity.EntryMethod),
		ProNumber:              entity.ProNumber,
		Bol:                    stringPtrFromValue(entity.BOL),
		CancelReason:           entity.CancelReason,
		OtherChargeAmount:      nullDecimalString(entity.OtherChargeAmount),
		FreightChargeAmount:    nullDecimalString(entity.FreightChargeAmount),
		BaseRate:               nullDecimalString(entity.BaseRate),
		TotalChargeAmount:      nullDecimalString(entity.TotalChargeAmount),
		Pieces:                 intPtr(entity.Pieces),
		Weight:                 intPtr(entity.Weight),
		TemperatureMin:         intPtrFromInt16(entity.TemperatureMin),
		TemperatureMax:         intPtrFromInt16(entity.TemperatureMax),
		ActualDeliveryDate:     intPtr(entity.ActualDeliveryDate),
		ActualShipDate:         intPtr(entity.ActualShipDate),
		CanceledAt:             intPtr(entity.CanceledAt),
		BillingTransferStatus:  stringPtrFromValue(string(entity.BillingTransferStatus)),
		TransferredToBillingAt: intPtr(entity.TransferredToBillingAt),
		MarkedReadyToBillAt:    intPtr(entity.MarkedReadyToBillAt),
		BilledAt:               intPtr(entity.BilledAt),
		RatingUnit:             int(entity.RatingUnit),
		RatingDetail:           ratingDetailToModel(entity.RatingDetail),
		Version:                int(entity.Version),
		CreatedAt:              int(entity.CreatedAt),
		UpdatedAt:              int(entity.UpdatedAt),
		Moves:                  moves,
		AdditionalCharges:      additionalCharges,
		Commodities:            commodities,
		Customer:               shipmentCustomerToModel(entity.Customer),
		Owner:                  entity.Owner,
		FormulaTemplate:        shipmentFormulaTemplateToModel(entity.FormulaTemplate),
	}
	return model, nil
}

func shipmentMovesToModel(
	entities []*shipmentdomain.ShipmentMove,
) ([]*gqlmodel.ShipmentMove, error) {
	moves := make([]*gqlmodel.ShipmentMove, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		stops, err := shipmentStopsToModel(entity.Stops)
		if err != nil {
			return nil, err
		}
		assignment, err := shipmentAssignmentToModel(entity.Assignment)
		if err != nil {
			return nil, err
		}
		moves = append(moves, &gqlmodel.ShipmentMove{
			ID:                     idPtr(entity.ID),
			BusinessUnitID:         entity.BusinessUnitID.String(),
			OrganizationID:         entity.OrganizationID.String(),
			ShipmentID:             idPtr(entity.ShipmentID),
			Status:                 gqlmodel.MoveStatus(entity.Status),
			Loaded:                 entity.Loaded,
			Sequence:               int(entity.Sequence),
			Distance:               entity.Distance,
			DistanceSource:         stringPtrFromValue(entity.DistanceSource),
			DistanceProvider:       stringPtrFromValue(entity.DistanceProvider),
			DistanceCalculatedAt:   intPtr(entity.DistanceCalculatedAt),
			DistanceRouteSignature: stringPtrFromValue(entity.DistanceRouteSignature),
			DistanceDataVersion:    stringPtrFromValue(entity.DistanceDataVersion),
			DistanceRoutingType:    stringPtrFromValue(entity.DistanceRoutingType),
			DistanceUnits:          stringPtrFromValue(entity.DistanceUnits),
			DistanceMetadata:       entity.DistanceMetadata,
			Version:                int(entity.Version),
			CreatedAt:              int(entity.CreatedAt),
			UpdatedAt:              int(entity.UpdatedAt),
			Stops:                  stops,
			Assignment:             assignment,
		})
	}
	return moves, nil
}

func shipmentStopsToModel(entities []*shipmentdomain.Stop) ([]*gqlmodel.ShipmentStop, error) {
	stops := make([]*gqlmodel.ShipmentStop, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		stops = append(stops, &gqlmodel.ShipmentStop{
			ID:                     idPtr(entity.ID),
			BusinessUnitID:         entity.BusinessUnitID.String(),
			OrganizationID:         entity.OrganizationID.String(),
			ShipmentMoveID:         idPtr(entity.ShipmentMoveID),
			LocationID:             entity.LocationID.String(),
			Status:                 gqlmodel.StopStatus(entity.Status),
			Type:                   gqlmodel.StopType(entity.Type),
			ScheduleType:           gqlmodel.StopScheduleType(entity.ScheduleType),
			Sequence:               int(entity.Sequence),
			Pieces:                 intPtr(entity.Pieces),
			Weight:                 intPtr(entity.Weight),
			ScheduledWindowStart:   int(entity.ScheduledWindowStart),
			ScheduledWindowEnd:     intPtr(entity.ScheduledWindowEnd),
			ActualArrival:          intPtr(entity.ActualArrival),
			ActualDeparture:        intPtr(entity.ActualDeparture),
			CountLateOverride:      entity.CountLateOverride,
			CountDetentionOverride: entity.CountDetentionOverride,
			AddressLine:            entity.AddressLine,
			Version:                int(entity.Version),
			CreatedAt:              int(entity.CreatedAt),
			UpdatedAt:              int(entity.UpdatedAt),
			Location:               entity.Location,
		})
	}
	return stops, nil
}

func shipmentAssignmentToModel(
	entity *shipmentdomain.Assignment,
) (*gqlmodel.ShipmentAssignment, error) {
	if entity == nil {
		return nil, nil
	}
	return &gqlmodel.ShipmentAssignment{
		ID:                idPtr(entity.ID),
		BusinessUnitID:    entity.BusinessUnitID.String(),
		OrganizationID:    entity.OrganizationID.String(),
		ShipmentMoveID:    idPtr(entity.ShipmentMoveID),
		PrimaryWorkerID:   idPtrFromPulidPtr(entity.PrimaryWorkerID),
		TractorID:         idPtrFromPulidPtr(entity.TractorID),
		TrailerID:         idPtrFromPulidPtr(entity.TrailerID),
		SecondaryWorkerID: idPtrFromPulidPtr(entity.SecondaryWorkerID),
		Status:            gqlmodel.AssignmentStatus(entity.Status),
		ArchivedAt:        intPtr(entity.ArchivedAt),
		Version:           int(entity.Version),
		CreatedAt:         int(entity.CreatedAt),
		UpdatedAt:         int(entity.UpdatedAt),
		Tractor:           entity.Tractor,
		Trailer:           entity.Trailer,
		PrimaryWorker:     entity.PrimaryWorker,
		SecondaryWorker:   entity.SecondaryWorker,
	}, nil
}

func shipmentAdditionalChargesToModel(
	entities []*shipmentdomain.AdditionalCharge,
) ([]*gqlmodel.ShipmentAdditionalCharge, error) {
	charges := make([]*gqlmodel.ShipmentAdditionalCharge, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		charges = append(charges, &gqlmodel.ShipmentAdditionalCharge{
			ID:                  idPtr(entity.ID),
			BusinessUnitID:      entity.BusinessUnitID.String(),
			OrganizationID:      entity.OrganizationID.String(),
			ShipmentID:          entity.ShipmentID.String(),
			AccessorialChargeID: entity.AccessorialChargeID.String(),
			IsSystemGenerated:   entity.IsSystemGenerated,
			Method:              string(entity.Method),
			Amount:              entity.Amount.String(),
			Unit:                int(entity.Unit),
			FuelSurchargeProgramID: idPtrFromPulidPtr(entity.FuelSurchargeProgramID),
			FuelSurchargeDetail:    fuelSurchargeDetailToMap(entity.FuelSurchargeDetail),
			Version:             int(entity.Version),
			CreatedAt:           int(entity.CreatedAt),
			UpdatedAt:           int(entity.UpdatedAt),
			AccessorialCharge:   shipmentAccessorialChargeToModel(entity.AccessorialCharge),
		})
	}
	return charges, nil
}

func shipmentCommoditiesToModel(
	entities []*shipmentdomain.ShipmentCommodity,
) ([]*gqlmodel.ShipmentCommodity, error) {
	commodities := make([]*gqlmodel.ShipmentCommodity, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		commodities = append(commodities, &gqlmodel.ShipmentCommodity{
			ID:             idPtr(entity.ID),
			BusinessUnitID: entity.BusinessUnitID.String(),
			OrganizationID: entity.OrganizationID.String(),
			ShipmentID:     entity.ShipmentID.String(),
			CommodityID:    entity.CommodityID.String(),
			Pieces:         int(entity.Pieces),
			Weight:         int(entity.Weight),
			Version:        int(entity.Version),
			CreatedAt:      int(entity.CreatedAt),
			UpdatedAt:      int(entity.UpdatedAt),
			Commodity:      shipmentCommodityDetailToModel(entity.Commodity),
		})
	}
	return commodities, nil
}

func shipmentCommentToModel(
	entity *shipmentdomain.ShipmentComment,
) (*gqlmodel.ShipmentComment, error) {
	if entity == nil {
		return nil, nil
	}
	mentions := make([]*gqlmodel.ShipmentCommentMention, 0, len(entity.MentionedUsers))
	for _, mention := range entity.MentionedUsers {
		if mention == nil {
			continue
		}
		mentions = append(mentions, &gqlmodel.ShipmentCommentMention{
			ID:              mention.ID.String(),
			CommentID:       mention.CommentID.String(),
			MentionedUserID: mention.MentionedUserID.String(),
			OrganizationID:  idPtr(mention.OrganizationID),
			BusinessUnitID:  idPtr(mention.BusinessUnitID),
			ShipmentID:      idPtr(mention.ShipmentID),
			CreatedAt:       int(mention.CreatedAt),
			MentionedUser:   mention.MentionedUser,
		})
	}
	mentionedUserIDs := make([]string, 0, len(entity.MentionedUserIDs))
	for _, id := range entity.MentionedUserIDs {
		mentionedUserIDs = append(mentionedUserIDs, id.String())
	}
	return &gqlmodel.ShipmentComment{
		ID:               entity.ID.String(),
		BusinessUnitID:   idPtr(entity.BusinessUnitID),
		OrganizationID:   idPtr(entity.OrganizationID),
		ShipmentID:       entity.ShipmentID.String(),
		UserID:           idPtr(entity.UserID),
		Comment:          entity.Comment,
		Type:             gqlmodel.ShipmentCommentType(entity.Type),
		Visibility:       gqlmodel.ShipmentCommentVisibility(entity.Visibility),
		Priority:         gqlmodel.ShipmentCommentPriority(entity.Priority),
		Source:           gqlmodel.ShipmentCommentSource(entity.Source),
		Metadata:         entity.Metadata,
		EditedAt:         intPtr(entity.EditedAt),
		Version:          int(entity.Version),
		CreatedAt:        int(entity.CreatedAt),
		UpdatedAt:        int(entity.UpdatedAt),
		MentionedUserIds: mentionedUserIDs,
		User:             entity.User,
		MentionedUsers:   mentions,
	}, nil
}

func shipmentDistanceToModel(
	response *services.DistanceCalculationResponse,
) *gqlmodel.ShipmentDistanceResponse {
	moves := make([]*gqlmodel.ShipmentDistanceMoveResult, 0, len(response.Moves))
	for _, move := range response.Moves {
		moves = append(moves, &gqlmodel.ShipmentDistanceMoveResult{
			MoveID:              idPtr(move.MoveID),
			MoveIndex:           move.MoveIndex,
			Distance:            move.Distance,
			Source:              move.Source,
			Provider:            stringPtrFromValue(move.Provider),
			RoutingType:         stringPtrFromValue(move.RoutingType),
			DataVersion:         stringPtrFromValue(move.DataVersion),
			DistanceUnits:       stringPtrFromValue(move.DistanceUnits),
			DistanceProfileID:   stringPtrFromValue(move.DistanceProfileID),
			DistanceProfileName: stringPtrFromValue(move.DistanceProfileName),
			Warnings:            move.Warnings,
			CalculatedAt:        int(move.CalculatedAt),
		})
	}
	return &gqlmodel.ShipmentDistanceResponse{
		ShipmentID:    idPtr(response.ShipmentID),
		TotalDistance: response.TotalDistance,
		Moves:         moves,
	}
}

func shipmentTotalsToModel(
	response *repositories.ShipmentTotalsResponse,
) *gqlmodel.ShipmentTotalsResponse {
	return &gqlmodel.ShipmentTotalsResponse{
		FreightChargeAmount: response.FreightChargeAmount.String(),
		OtherChargeAmount:   response.OtherChargeAmount.String(),
		TotalChargeAmount:   response.TotalChargeAmount.String(),
	}
}

func previousRatesToModel(
	result *pagination.ListResult[*repositories.PreviousRateSummary],
) *gqlmodel.ShipmentPreviousRatesResponse {
	items := make([]*gqlmodel.ShipmentPreviousRateSummary, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, &gqlmodel.ShipmentPreviousRateSummary{
			ShipmentID:          item.ShipmentID.String(),
			ProNumber:           item.ProNumber,
			CustomerID:          item.CustomerID.String(),
			ServiceTypeID:       item.ServiceTypeID.String(),
			ShipmentTypeID:      item.ShipmentTypeID.String(),
			FormulaTemplateID:   item.FormulaTemplateID.String(),
			FreightChargeAmount: item.FreightChargeAmount.String(),
			OtherChargeAmount:   item.OtherChargeAmount.String(),
			TotalChargeAmount:   item.TotalChargeAmount.String(),
			RatingUnit:          int(item.RatingUnit),
			Pieces:              intPtr(item.Pieces),
			Weight:              intPtr(item.Weight),
			CreatedAt:           int(item.CreatedAt),
		})
	}
	return &gqlmodel.ShipmentPreviousRatesResponse{
		Items: items,
		Total: result.Total,
	}
}

func loadingOptimizationRequestFromInput(
	input gqlmodel.ShipmentLoadingOptimizationInput,
	authCtx *authctx.AuthContext,
) (*repositories.LoadingOptimizationRequest, error) {
	commodities := make([]repositories.LoadingCommodityInput, 0, len(input.Commodities))
	for _, commodity := range input.Commodities {
		if commodity == nil {
			continue
		}
		commodityID, err := pulid.MustParse(commodity.CommodityID)
		if err != nil {
			return nil, err
		}
		commodities = append(commodities, repositories.LoadingCommodityInput{
			CommodityID: commodityID,
			Pieces:      int64(commodity.Pieces),
			Weight:      int64(commodity.Weight),
		})
	}
	var equipmentTypeID *pulid.ID
	if input.EquipmentTypeID != nil && *input.EquipmentTypeID != "" {
		parsed, err := pulid.MustParse(*input.EquipmentTypeID)
		if err != nil {
			return nil, err
		}
		equipmentTypeID = &parsed
	}
	stops := make([]repositories.StopInfo, 0, len(input.Stops))
	for _, stop := range input.Stops {
		if stop == nil {
			continue
		}
		stops = append(stops, repositories.StopInfo{
			Sequence:     stop.Sequence,
			LocationName: stop.LocationName,
			LocationCity: stop.LocationCity,
		})
	}
	return &repositories.LoadingOptimizationRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		Commodities:     commodities,
		EquipmentTypeID: equipmentTypeID,
		Stops:           stops,
	}, nil
}

func loadingOptimizationToModel(
	result *repositories.LoadingOptimizationResult,
) *gqlmodel.ShipmentLoadingOptimizationResponse {
	placements := make([]*gqlmodel.ShipmentLoadingCommodity, 0, len(result.Placements))
	for _, item := range result.Placements {
		placements = append(placements, &gqlmodel.ShipmentLoadingCommodity{
			CommodityID:         item.CommodityID.String(),
			CommodityName:       item.CommodityName,
			PositionFeet:        item.PositionFeet,
			LengthFeet:          item.LengthFeet,
			Weight:              int(item.Weight),
			Pieces:              int(item.Pieces),
			Stackable:           item.Stackable,
			Fragile:             item.Fragile,
			IsHazmat:            item.IsHazmat,
			HazmatClass:         stringPtrFromValue(item.HazmatClass),
			MinTemp:             item.MinTemp,
			MaxTemp:             item.MaxTemp,
			LoadingInstructions: stringPtrFromValue(item.LoadingInstructions),
			EstimatedLength:     item.EstimatedLength,
			StopNumber:          intPtrFromValue(item.StopNumber),
		})
	}
	hazmatZones := make([]*gqlmodel.ShipmentHazmatZone, 0, len(result.HazmatZones))
	for _, item := range result.HazmatZones {
		hazmatZones = append(hazmatZones, &gqlmodel.ShipmentHazmatZone{
			CommodityAId:         item.CommodityAID.String(),
			CommodityBId:         item.CommodityBID.String(),
			CommodityAName:       item.CommodityAName,
			CommodityBName:       item.CommodityBName,
			RuleName:             item.RuleName,
			SegregationType:      item.SegregationType,
			RequiredDistanceFeet: item.RequiredDistanceFeet,
			ActualDistanceFeet:   item.ActualDistanceFeet,
			Satisfied:            item.Satisfied,
		})
	}
	warnings := make([]*gqlmodel.ShipmentLoadingWarning, 0, len(result.Warnings))
	for _, item := range result.Warnings {
		warnings = append(warnings, &gqlmodel.ShipmentLoadingWarning{
			Type:         item.Type,
			Message:      item.Message,
			Severity:     item.Severity,
			CommodityIds: item.CommodityIDs,
		})
	}
	axleWeights := make([]*gqlmodel.ShipmentAxleWeight, 0, len(result.AxleWeights))
	for _, item := range result.AxleWeights {
		axleWeights = append(axleWeights, &gqlmodel.ShipmentAxleWeight{
			Axle:       item.Axle,
			Weight:     int(item.Weight),
			Limit:      int(item.Limit),
			Percentage: item.Percentage,
			Compliant:  item.Compliant,
		})
	}
	recommendations := make([]*gqlmodel.ShipmentLoadingRecommendation, 0, len(result.Recommendations))
	for _, item := range result.Recommendations {
		recommendations = append(recommendations, &gqlmodel.ShipmentLoadingRecommendation{
			Type:         item.Type,
			Priority:     item.Priority,
			Title:        item.Title,
			Description:  item.Description,
			Impact:       stringPtrFromValue(item.Impact),
			CommodityIds: item.CommodityIDs,
		})
	}
	stopDividers := make([]*gqlmodel.ShipmentStopDivider, 0, len(result.StopDividers))
	for _, item := range result.StopDividers {
		stopDividers = append(stopDividers, &gqlmodel.ShipmentStopDivider{
			PositionFeet: item.PositionFeet,
			StopNumber:   item.StopNumber,
			Label:        item.Label,
		})
	}
	return &gqlmodel.ShipmentLoadingOptimizationResponse{
		TrailerLengthFeet: result.TrailerLengthFeet,
		TotalLinearFeet:   result.TotalLinearFeet,
		TotalWeight:       int(result.TotalWeight),
		MaxWeight:         int(result.MaxWeight),
		LinearFeetUtil:    result.LinearFeetUtil,
		WeightUtil:        result.WeightUtil,
		UtilizationScore:  result.UtilizationScore,
		UtilizationGrade:  result.UtilizationGrade,
		Placements:        placements,
		HazmatZones:       hazmatZones,
		Warnings:          warnings,
		AxleWeights:       axleWeights,
		Recommendations:   recommendations,
		StopDividers:      stopDividers,
		AiAnalysis:        stringPtrFromValue(result.AIAnalysis),
	}
}

func nullDecimalFromInput(value *string) (decimal.NullDecimal, error) {
	parsed, err := decimalFromInput(value)
	if err != nil {
		return decimal.NullDecimal{}, err
	}
	return decimal.NewNullDecimal(parsed), nil
}

func decimalFromInput(value *string) (decimal.Decimal, error) {
	if value == nil || *value == "" {
		return decimal.Zero, nil
	}
	parsed, err := decimal.NewFromString(*value)
	if err != nil {
		return decimal.Decimal{}, errortypes.NewValidationError(
			"amount",
			errortypes.ErrInvalidFormat,
			"Amount must be a valid decimal",
		)
	}
	return parsed, nil
}

func nullDecimalString(value decimal.NullDecimal) string {
	if !value.Valid {
		return decimal.Zero.String()
	}
	return value.Decimal.String()
}

func int16PtrFromInput(field string, value *int) (*int16, error) {
	if value == nil {
		return nil, nil
	}
	converted, err := int16FromInput(field, *value)
	if err != nil {
		return nil, err
	}
	return &converted, nil
}

func int16FromInput(field string, value int) (int16, error) {
	if value < math.MinInt16 || value > math.MaxInt16 {
		return 0, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			fmt.Sprintf("%s is outside the allowed range", field),
		)
	}
	return int16(value), nil
}

func ratingDetailToModel(detail *shipmentdomain.RatingDetail) *gqlmodel.ShipmentRatingDetail {
	if detail == nil {
		return nil
	}
	return &gqlmodel.ShipmentRatingDetail{
		FormulaTemplateID:   detail.FormulaTemplateID,
		FormulaTemplateName: detail.FormulaTemplateName,
		Expression:          detail.Expression,
		ResolvedVariables:   detail.ResolvedVariables,
		Result:              detail.Result,
		RatedAt:             int(detail.RatedAt),
	}
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

func optionalJSON(value any) (map[string]any, error) {
	if value == nil {
		return nil, nil
	}
	return jsonutils.ToJSON(value)
}

func stringPtrFromValue(value string) *string {
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

func intPtrFromValue(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func idPtrFromPulidPtr(value *pulid.ID) *string {
	if value == nil || value.IsNil() {
		return nil
	}
	converted := value.String()
	return &converted
}

func pulidPtrFromOptionalString(value *string) (*pulid.ID, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	parsed, err := pulid.MustParse(*value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (r *shipmentResolver) loadShipmentOrder(
	ctx context.Context,
	obj *gqlmodel.Shipment,
) (*order.Order, error) {
	if obj == nil || obj.OrderID == nil || *obj.OrderID == "" {
		return nil, nil
	}

	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return nil, errortypes.NewDatabaseError("Order loader is not configured")
	}

	parent, err := loadersForRequest.OrderByID.Load(ctx, *obj.OrderID)()
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return parent, nil
}
