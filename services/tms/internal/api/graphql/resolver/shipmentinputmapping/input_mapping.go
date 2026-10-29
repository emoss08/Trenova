package shipmentinputmapping

import (
	"fmt"
	"math"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/tariffmapping"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func ShipmentFromInput(
	input gqlmodel.ShipmentInput,
	id pulid.ID,
	authCtx *authctx.AuthContext,
) (*shipmentdomain.Shipment, error) {
	serviceTypeID, err := gqlid.RequiredID("serviceTypeId", input.ServiceTypeID)
	if err != nil {
		return nil, err
	}
	shipmentTypeID, err := gqlid.RequiredID("shipmentTypeId", input.ShipmentTypeID)
	if err != nil {
		return nil, err
	}
	customerID, err := gqlid.RequiredID("customerId", input.CustomerID)
	if err != nil {
		return nil, err
	}
	formulaTemplateID, err := gqlid.OptionalScopedID("formulaTemplateId", input.FormulaTemplateID)
	if err != nil {
		return nil, err
	}
	tractorTypeID, err := gqlid.OptionalID(input.TractorTypeID)
	if err != nil {
		return nil, err
	}
	trailerTypeID, err := gqlid.OptionalID(input.TrailerTypeID)
	if err != nil {
		return nil, err
	}
	ownerID, err := gqlid.OptionalID(input.OwnerID)
	if err != nil {
		return nil, err
	}
	enteredByID, err := gqlid.OptionalID(input.EnteredByID)
	if err != nil {
		return nil, err
	}
	canceledByID, err := gqlid.OptionalID(input.CanceledByID)
	if err != nil {
		return nil, err
	}
	consolidationGroupID, err := gqlid.OptionalID(input.ConsolidationGroupID)
	if err != nil {
		return nil, err
	}
	orderID, err := gqlid.OptionalID(input.OrderID)
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
	tariffInputs, err := tariffmapping.TariffInputsFromModel("tariffInputs", input.TariffInputs)
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
		ID:                   id,
		BusinessUnitID:       authCtx.BusinessUnitID,
		OrganizationID:       authCtx.OrganizationID,
		ServiceTypeID:        serviceTypeID,
		ShipmentTypeID:       shipmentTypeID,
		CustomerID:           customerID,
		TractorTypeID:        tractorTypeID,
		TrailerTypeID:        trailerTypeID,
		OwnerID:              ownerID,
		EnteredByID:          enteredByID,
		CanceledByID:         canceledByID,
		FormulaTemplateID:    formulaTemplateID,
		ConsolidationGroupID: consolidationGroupID,
		OrderID:              orderID,
		Status:               status,
		EntryMethod:          entryMethod,
		ProNumber:            gqlscalar.StringValue(input.ProNumber),
		BOL:                  gqlscalar.StringValue(input.BOL),
		ExternalReference:    gqlscalar.StringValue(input.ExternalReference),
		CancelReason:         gqlscalar.StringValue(input.CancelReason),
		OtherChargeAmount:    otherChargeAmount,
		FreightChargeAmount:  freightChargeAmount,
		TariffInputs:         tariffInputs.OrEmpty(),
		TotalChargeAmount:    totalChargeAmount,
		Pieces:               gqlscalar.Int64Ptr(input.Pieces),
		Weight:               gqlscalar.Int64Ptr(input.Weight),
		TemperatureMin:       temperatureMin,
		TemperatureMax:       temperatureMax,
		ActualDeliveryDate:   gqlscalar.Int64Ptr(input.ActualDeliveryDate),
		ActualShipDate:       gqlscalar.Int64Ptr(input.ActualShipDate),
		CanceledAt:           gqlscalar.Int64Ptr(input.CanceledAt),
		BillingTransferStatus: shipmentdomain.BillingTransferStatus(
			gqlscalar.StringValue(input.BillingTransferStatus),
		),
		TransferredToBillingAt: gqlscalar.Int64Ptr(input.TransferredToBillingAt),
		MarkedReadyToBillAt:    gqlscalar.Int64Ptr(input.MarkedReadyToBillAt),
		BilledAt:               gqlscalar.Int64Ptr(input.BilledAt),
		FuelSurchargeLocked:    gqlscalar.BoolValue(input.FuelSurchargeLocked),
		RateOverrideReason:     gqlscalar.StringValue(input.RateOverrideReason),
		SourceDocumentID:       gqlscalar.StringValue(input.SourceDocumentID),
		FreightTerms:           shipmentdomain.FreightTermsPrepaid,
	}
	if input.FreightTerms != nil {
		entity.FreightTerms = *input.FreightTerms
	}
	billToCustomerID, err := gqlid.OptionalScopedID("billToCustomerId", input.BillToCustomerID)
	if err != nil {
		return nil, err
	}
	if billToCustomerID.IsNotNil() {
		entity.BillToCustomerID = &billToCustomerID
	}
	if input.TenderStatus != nil {
		tenderStatus := shipmentdomain.TenderStatus(*input.TenderStatus)
		entity.TenderStatus = &tenderStatus
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
	allocations, err := shipmentChargeAllocationsFromInput(&input, authCtx)
	if err != nil {
		return nil, err
	}
	entity.Moves = moves
	entity.AdditionalCharges = additionalCharges
	entity.Commodities = commodities
	entity.ChargeAllocations = allocations

	return entity, nil
}

func shipmentMovesFromInput(
	inputs []*gqlmodel.ShipmentMoveInput,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.ShipmentMove, error) {
	moves := make([]*shipmentdomain.ShipmentMove, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			continue
		}
		path := fmt.Sprintf("moves[%d]", idx)
		id, err := gqlid.OptionalScopedID(path+".id", input.ID)
		if err != nil {
			return nil, err
		}
		shipmentID, err := gqlid.OptionalScopedID(path+".shipmentId", input.ShipmentID)
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
			Sequence:               gqlscalar.Int64Value(input.Sequence),
			Distance:               input.Distance,
			DistanceSource:         gqlscalar.StringValue(input.DistanceSource),
			DistanceProvider:       gqlscalar.StringValue(input.DistanceProvider),
			DistanceCalculatedAt:   gqlscalar.Int64Ptr(input.DistanceCalculatedAt),
			DistanceRouteSignature: gqlscalar.StringValue(input.DistanceRouteSignature),
			DistanceDataVersion:    gqlscalar.StringValue(input.DistanceDataVersion),
			DistanceRoutingType:    gqlscalar.StringValue(input.DistanceRoutingType),
			DistanceUnits:          gqlscalar.StringValue(input.DistanceUnits),
			DistanceMetadata:       input.DistanceMetadata,
		}
		if input.Version != nil {
			move.Version = int64(*input.Version)
		}
		stops, err := shipmentStopsFromInput(input.Stops, path, authCtx)
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
	movePath string,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.Stop, error) {
	stops := make([]*shipmentdomain.Stop, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			continue
		}
		path := fmt.Sprintf("%s.stops[%d]", movePath, idx)
		id, err := gqlid.OptionalScopedID(path+".id", input.ID)
		if err != nil {
			return nil, err
		}
		shipmentMoveID, err := gqlid.OptionalScopedID(path+".shipmentMoveId", input.ShipmentMoveID)
		if err != nil {
			return nil, err
		}
		locationID, err := gqlid.RequiredID(path+".locationId", input.LocationID)
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
			Sequence:               gqlscalar.Int64Value(input.Sequence),
			Pieces:                 gqlscalar.Int64Ptr(input.Pieces),
			Weight:                 gqlscalar.Int64Ptr(input.Weight),
			ScheduledWindowStart:   gqlscalar.Int64Value(input.ScheduledWindowStart),
			ScheduledWindowEnd:     gqlscalar.Int64Ptr(input.ScheduledWindowEnd),
			ActualArrival:          gqlscalar.Int64Ptr(input.ActualArrival),
			ActualDeparture:        gqlscalar.Int64Ptr(input.ActualDeparture),
			CountLateOverride:      input.CountLateOverride,
			CountDetentionOverride: input.CountDetentionOverride,
			AddressLine:            gqlscalar.StringValue(input.AddressLine),
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
	for idx, input := range inputs {
		if input == nil {
			continue
		}
		path := fmt.Sprintf("additionalCharges[%d]", idx)
		id, err := gqlid.OptionalScopedID(path+".id", input.ID)
		if err != nil {
			return nil, err
		}
		shipmentID, err := gqlid.OptionalScopedID(path+".shipmentId", input.ShipmentID)
		if err != nil {
			return nil, err
		}
		accessorialChargeID, err := gqlid.RequiredID(
			path+".accessorialChargeId",
			input.AccessorialChargeID,
		)
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
		fuelSurchargeProgramID, err := gqlid.OptionalScopedID(
			path+".fuelSurchargeProgramId",
			input.FuelSurchargeProgramID,
		)
		if err != nil {
			return nil, err
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
		if !fuelSurchargeProgramID.IsNil() {
			charge.FuelSurchargeProgramID = &fuelSurchargeProgramID
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
	for idx, input := range inputs {
		if input == nil {
			continue
		}
		path := fmt.Sprintf("commodities[%d]", idx)
		id, err := gqlid.OptionalScopedID(path+".id", input.ID)
		if err != nil {
			return nil, err
		}
		shipmentID, err := gqlid.OptionalScopedID(path+".shipmentId", input.ShipmentID)
		if err != nil {
			return nil, err
		}
		commodityID, err := gqlid.RequiredID(path+".commodityId", input.CommodityID)
		if err != nil {
			return nil, err
		}
		commodity := &shipmentdomain.ShipmentCommodity{
			ID:             id,
			BusinessUnitID: authCtx.BusinessUnitID,
			OrganizationID: authCtx.OrganizationID,
			ShipmentID:     shipmentID,
			CommodityID:    commodityID,
			Pieces:         gqlscalar.Int64Value(input.Pieces),
			Weight:         gqlscalar.Int64Value(input.Weight),
			LengthFeet:     input.LengthFeet,
			WidthFeet:      input.WidthFeet,
			HeightFeet:     input.HeightFeet,
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

func CommentFiltersFromGraphQL(
	filter *gqlmodel.ShipmentCommentsFilterInput,
) (shipmentcommentports.ShipmentCommentListFilters, error) {
	filters := shipmentcommentports.ShipmentCommentListFilters{}
	if filter == nil {
		return filters, nil
	}

	if len(filter.Types) > 0 {
		filters.Types = make([]shipmentdomain.CommentType, 0, len(filter.Types))
		for _, commentType := range filter.Types {
			filters.Types = append(filters.Types, shipmentdomain.CommentType(commentType))
		}
	}
	if len(filter.Priorities) > 0 {
		filters.Priorities = make([]shipmentdomain.CommentPriority, 0, len(filter.Priorities))
		for _, priority := range filter.Priorities {
			filters.Priorities = append(
				filters.Priorities,
				shipmentdomain.CommentPriority(priority),
			)
		}
	}
	if len(filter.AuthorIds) > 0 {
		authorIDs, err := gqlid.ParseIDs(filter.AuthorIds)
		if err != nil {
			return filters, err
		}
		filters.AuthorIDs = authorIDs
	}

	mentionsUserID, err := gqlid.PulidPtrFromOptionalString(filter.MentionsUserID)
	if err != nil {
		return filters, err
	}
	filters.MentionsUserID = mentionsUserID
	filters.PinnedOnly = gqlscalar.BoolValue(filter.PinnedOnly)
	filters.UnresolvedOnly = gqlscalar.BoolValue(filter.UnresolvedOnly)
	filters.Search = strings.TrimSpace(gqlscalar.StringValue(filter.Search))

	return filters, nil
}

func LoadingOptimizationRequestFromInput(
	input gqlmodel.ShipmentLoadingOptimizationInput,
	authCtx *authctx.AuthContext,
) (*shipmentserviceports.LoadingOptimizationRequest, error) {
	commodities := make([]shipmentserviceports.LoadingCommodityInput, 0, len(input.Commodities))
	for _, commodity := range input.Commodities {
		if commodity == nil {
			continue
		}
		commodityID, err := pulid.MustParse(commodity.CommodityID)
		if err != nil {
			return nil, err
		}
		commodities = append(commodities, shipmentserviceports.LoadingCommodityInput{
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
	stops := make([]shipmentserviceports.StopInfo, 0, len(input.Stops))
	for _, stop := range input.Stops {
		if stop == nil {
			continue
		}
		stops = append(stops, shipmentserviceports.StopInfo{
			Sequence:     stop.Sequence,
			LocationName: stop.LocationName,
			LocationCity: stop.LocationCity,
		})
	}
	return &shipmentserviceports.LoadingOptimizationRequest{
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
			"{0} is outside the allowed range", field,
		)
	}
	return int16(value), nil
}
