package edipayloadbuild

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

func BuildTenderPayload(source *shipment.Shipment) editransfer.LoadTenderPayload {
	payload := editransfer.LoadTenderPayload{
		PurposeCode:          editransfer.LoadTenderPurposeOriginal,
		ShipmentID:           source.ID,
		BusinessUnitID:       source.BusinessUnitID,
		OrganizationID:       source.OrganizationID,
		ServiceTypeID:        source.ServiceTypeID,
		ServiceTypeLabel:     serviceTypeLabel(source),
		ShipmentTypeID:       source.ShipmentTypeID,
		ShipmentTypeLabel:    shipmentTypeLabel(source),
		CustomerID:           source.CustomerID,
		CustomerLabel:        customerLabel(source),
		FormulaTemplateID:    source.FormulaTemplateID,
		FormulaTemplateLabel: formulaTemplateLabel(source),
		ProNumber:            source.ProNumber,
		BOL:                  source.BOL,
		Payment:              string(source.FreightTerms),
		Equipment:            EquipmentCode(source),
		Pieces:               source.Pieces,
		Weight:               source.Weight,
		TemperatureMin:       source.TemperatureMin,
		TemperatureMax:       source.TemperatureMax,
		FreightChargeAmount:  source.FreightChargeAmount,
		OtherChargeAmount:    source.OtherChargeAmount,
		TotalChargeAmount:    source.TotalChargeAmount,
		Moves:                make([]editransfer.LoadTenderMove, 0, len(source.Moves)),
		Commodities: make(
			[]editransfer.LoadTenderCommodity,
			0,
			len(source.Commodities),
		),
		AdditionalCharges: make(
			[]editransfer.LoadTenderCharge,
			0,
			len(source.AdditionalCharges),
		),
		RequiredMappingEntityIDs: map[edi.MappingEntityType][]pulid.ID{},
	}
	addRequiredID(
		payload.RequiredMappingEntityIDs,
		edi.MappingEntityTypeCustomer,
		source.CustomerID,
	)
	addRequiredID(
		payload.RequiredMappingEntityIDs,
		edi.MappingEntityTypeServiceType,
		source.ServiceTypeID,
	)
	addRequiredID(
		payload.RequiredMappingEntityIDs,
		edi.MappingEntityTypeFormulaTemplate,
		source.FormulaTemplateID,
	)
	addRequiredID(
		payload.RequiredMappingEntityIDs,
		edi.MappingEntityTypeShipmentType,
		source.ShipmentTypeID,
	)

	for _, move := range source.Moves {
		tenderMove := LoadTenderMoveFromShipmentMove(move)
		for i := range tenderMove.Stops {
			addRequiredID(
				payload.RequiredMappingEntityIDs,
				edi.MappingEntityTypeLocation,
				tenderMove.Stops[i].LocationID,
			)
		}
		payload.Moves = append(payload.Moves, tenderMove)
	}

	for _, commodity := range source.Commodities {
		payload.Commodities = append(
			payload.Commodities,
			LoadTenderCommodityFromShipmentCommodity(commodity),
		)
		addRequiredID(
			payload.RequiredMappingEntityIDs,
			edi.MappingEntityTypeCommodity,
			commodity.CommodityID,
		)
	}

	for _, charge := range source.AdditionalCharges {
		payload.AdditionalCharges = append(payload.AdditionalCharges, editransfer.LoadTenderCharge{
			AccessorialChargeID:    charge.AccessorialChargeID,
			AccessorialLabel:       accessorialLabel(charge),
			AccessorialCode:        accessorialCode(charge),
			AccessorialDescription: accessorialDescription(charge),
			Method:                 string(charge.Method),
			Amount:                 charge.Amount,
			Unit:                   charge.Unit,
		})
		addRequiredID(
			payload.RequiredMappingEntityIDs,
			edi.MappingEntityTypeAccessorialCharge,
			charge.AccessorialChargeID,
		)
	}

	return payload
}

func LoadTenderMoveFromShipmentMove(move *shipment.ShipmentMove) editransfer.LoadTenderMove {
	tenderMove := editransfer.LoadTenderMove{
		Loaded:   move.Loaded,
		Sequence: move.Sequence,
		Distance: move.Distance,
		Stops:    make([]editransfer.LoadTenderStop, 0, len(move.Stops)),
	}
	for _, stop := range move.Stops {
		locLabel, locAddr := stopLocationDetails(stop)
		tenderStop := editransfer.LoadTenderStop{
			LocationID:           stop.LocationID,
			LocationLabel:        locLabel,
			Type:                 string(stop.Type),
			ScheduleType:         string(stop.ScheduleType),
			Sequence:             stop.Sequence,
			Pieces:               stop.Pieces,
			Weight:               stop.Weight,
			TemperatureMin:       stop.TemperatureMin,
			TemperatureMax:       stop.TemperatureMax,
			ScheduledWindowStart: stop.ScheduledWindowStart,
			ScheduledWindowEnd:   stop.ScheduledWindowEnd,
			AddressLine:          stringutils.FirstNonEmpty(stop.AddressLine, locAddr),
		}
		if stop.Location != nil {
			tenderStop.LocationName = stop.Location.Name
			tenderStop.LocationCode = stop.Location.Code
			tenderStop.LocationAddressLine1 = stop.Location.AddressLine1
			tenderStop.LocationAddressLine2 = stop.Location.AddressLine2
			tenderStop.LocationCity = stop.Location.City
			tenderStop.LocationPostalCode = stop.Location.PostalCode
			if stop.Location.State != nil {
				tenderStop.LocationStateCode = stop.Location.State.Abbreviation
			}
		}
		tenderMove.Stops = append(tenderMove.Stops, tenderStop)
	}
	return tenderMove
}

func LoadTenderCommodityFromShipmentCommodity(
	commodity *shipment.ShipmentCommodity,
) editransfer.LoadTenderCommodity {
	return editransfer.LoadTenderCommodity{
		CommodityID:          commodity.CommodityID,
		CommodityLabel:       commodityLabel(commodity),
		CommodityName:        commodityName(commodity),
		CommodityDescription: commodityDescription(commodity),
		Weight:               commodity.Weight,
		Pieces:               commodity.Pieces,
		TemperatureMin:       commodity.TemperatureMin,
		TemperatureMax:       commodity.TemperatureMax,
		Hazmat:               commodityHazmat(commodity),
	}
}

func EquipmentCode(source *shipment.Shipment) string {
	if source == nil {
		return ""
	}
	if source.TrailerType != nil && source.TrailerType.Code != "" {
		return source.TrailerType.Code
	}
	if source.TractorType != nil {
		return source.TractorType.Code
	}
	return ""
}

func BuildFreightInvoicePayload(source *invoice.Invoice) edipayload.DocumentPayload {
	payload := edipayload.FreightInvoicePayload{
		InvoiceID:     source.ID,
		InvoiceNumber: source.Number,
		InvoiceDate:   source.InvoiceDate,
		ShipmentID:    source.ShipmentID,
		BOL: stringutils.FirstNonEmpty(
			source.ShipmentBOL,
			shipmentBOL(source.Shipment),
		),
		ProNumber: stringutils.FirstNonEmpty(
			source.ShipmentProNumber,
			shipmentProNumber(source.Shipment),
		),
		PONumber:           strings.TrimSpace(source.PONumber),
		ShipperReference:   partnerReference(source.Shipment),
		BillToName:         source.BillToName,
		BillToAddressLine1: source.BillToAddressLine1,
		BillToAddressLine2: source.BillToAddressLine2,
		BillToCity:         source.BillToCity,
		BillToStateCode:    source.BillToState,
		BillToPostalCode:   source.BillToPostalCode,
		BillToCountry:      source.BillToCountry,
		CurrencyCode:       source.CurrencyCode,
		Payment:            shipmentFreightTerms(source.Shipment),
		TotalAmount: decimal.NullDecimal{
			Decimal: source.TotalAmount,
			Valid:   true,
		},
		LineCharges: make([]edipayload.FreightInvoiceCharge, 0, len(source.Lines)),
	}
	payload.ReferenceNumbers = partnerReferences(payload.BOL, payload.ProNumber)
	if payload.PONumber != "" {
		payload.ReferenceNumbers[edipayload.RefPurchaseOrder] = payload.PONumber
	}

	lines := make([]*invoice.InvoiceLine, 0, len(source.Lines))
	for _, line := range source.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, line)
	}
	slices.SortFunc(lines, func(a, b *invoice.InvoiceLine) int {
		return cmp.Compare(a.LineNumber, b.LineNumber)
	})

	// Attribution is only meaningful when the invoice bills more than one
	// shipment. On a single-shipment invoice the header already says which one,
	// and repeating it on every charge changes the wire format of every 210 we
	// have ever sent for no gain.
	consolidated := source.ShipmentCount > 1

	for _, line := range lines {
		charge := edipayload.FreightInvoiceCharge{
			Sequence:    int64(line.LineNumber),
			Code:        specialChargeCode(line),
			Description: line.Description,
			Amount:      line.Amount,
		}
		if consolidated {
			charge.ShipmentID = line.ShipmentID
			charge.ProNumber = line.ShipmentProNumber
			charge.BOL = line.ShipmentBOL
		}
		payload.LineCharges = append(payload.LineCharges, charge)
	}

	if consolidated {
		payload.Shipments = freightInvoiceShipments(lines)
	}

	return edipayload.DocumentPayload{
		TransactionSet: edidocument.TransactionSet210,
		FreightInvoice: &payload,
	}
}

// freightInvoiceShipments rolls the lines up into one reference per shipment.
//
// The lines are the only record of which shipments an invoice covers, so this
// reads them rather than taking a list that could disagree. Order-level charges
// carry no shipment and contribute no reference, which is correct: they belong
// to the invoice, not to any one load.
func freightInvoiceShipments(lines []*invoice.InvoiceLine) []edipayload.FreightInvoiceShipment {
	order := make([]pulid.ID, 0, len(lines))
	byID := make(map[pulid.ID]*edipayload.FreightInvoiceShipment, len(lines))

	for _, line := range lines {
		if line.ShipmentID.IsNil() {
			continue
		}
		existing, seen := byID[line.ShipmentID]
		if !seen {
			order = append(order, line.ShipmentID)
			byID[line.ShipmentID] = &edipayload.FreightInvoiceShipment{
				ShipmentID: line.ShipmentID,
				ProNumber:  line.ShipmentProNumber,
				BOL:        line.ShipmentBOL,
				Amount:     decimal.NewNullDecimal(line.Amount),
			}
			continue
		}
		existing.Amount = decimal.NewNullDecimal(existing.Amount.Decimal.Add(line.Amount))
	}

	out := make([]edipayload.FreightInvoiceShipment, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}

	return out
}

func BuildShipmentStatusPayload(source *shipment.Shipment) edipayload.DocumentPayload {
	if source == nil {
		return edipayload.DocumentPayload{
			TransactionSet: edidocument.TransactionSet214,
			ShipmentStatus: &edipayload.ShipmentStatusPayload{References: map[string]string{}},
		}
	}
	payload := edipayload.ShipmentStatusPayload{
		ShipmentID: source.ID,
		StatusCode: "X3",
		References: partnerReferences(source.BOL, source.ProNumber),
	}
	applyShipmentStatusShipment(&payload, source)
	if len(source.Moves) > 0 &&
		source.Moves[0] != nil &&
		len(source.Moves[0].Stops) > 0 &&
		source.Moves[0].Stops[0] != nil {
		stop := source.Moves[0].Stops[0]
		payload.EventDate = stop.ScheduledWindowStart
		payload.EventTime = stop.ScheduledWindowStart
		applyShipmentStatusStop(&payload, stop)
	}
	return edipayload.DocumentPayload{
		TransactionSet: edidocument.TransactionSet214,
		ShipmentStatus: &payload,
	}
}

func BuildShipmentEventStatusPayload(
	event *shipmentevent.Event,
	source *shipment.Shipment,
) edipayload.DocumentPayload {
	stop := shipmentEventStop(event, source)
	payload := edipayload.ShipmentStatusPayload{
		ShipmentID: event.ShipmentID,
		EventID:    event.ID,
		StatusCode: shipmentEventStatusCode(event, stop, completesShipment(source, stop)),
		EventDate:  event.OccurredAt,
		EventTime:  event.OccurredAt,
		References: partnerReferences(shipmentBOL(source), shipmentProNumber(source)),
	}
	applyShipmentStatusShipment(&payload, source)
	if reason := stringutils.FirstNonEmpty(
		metadataString(event.Metadata, "statusReasonCode"),
		metadataString(event.Metadata, "reasonCode"),
		metadataString(event.Metadata, "reason"),
	); reason != "" {
		payload.StatusReasonCode = reason
		payload.ReasonCode = reason
	}
	if reasonDescription := metadataString(
		event.Metadata,
		"reasonDescription",
	); reasonDescription != "" {
		payload.ReasonDescription = reasonDescription
	}
	if exceptionCode := metadataString(event.Metadata, "exceptionCode"); exceptionCode != "" {
		payload.ExceptionCode = exceptionCode
	}
	if lateMinutes, ok := metadataInt64(event.Metadata, "lateMinutes"); ok {
		payload.LateMinutes = &lateMinutes
	}
	applyShipmentStatusStop(&payload, stop)
	return edipayload.DocumentPayload{
		TransactionSet: edidocument.TransactionSet214,
		ShipmentStatus: &payload,
	}
}

func (s *Service) BuildShipmentStatusPayloadForServiceFailure(
	ctx context.Context,
	req *servicefailureports.BuildServiceFailureEDIPayloadRequest,
) (*servicefailureports.ServiceFailureEDIPayloadResult, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	failure, err := s.serviceFailureRepo.GetByID(
		ctx,
		&servicefailureports.GetServiceFailureByIDRequest{
			ID:         req.ServiceFailureID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	source, err := s.shipmentSvc.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         failure.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	payload := BuildServiceFailureShipmentStatusPayload(failure, source)
	diagnostics := ServiceFailurePayloadDiagnostics(payload.ShipmentStatus)
	return &servicefailureports.ServiceFailureEDIPayloadResult{
		Payload:     payload,
		Diagnostics: diagnostics,
	}, nil
}

func BuildServiceFailureShipmentStatusPayload(
	failure *servicefailure.ServiceFailure,
	source *shipment.Shipment,
) edipayload.DocumentPayload {
	eventTimestamp := serviceFailureEventTimestamp(failure)
	status := edipayload.ShipmentStatusPayload{
		ShipmentID:           failure.ShipmentID,
		StatusCode:           serviceFailureStatusCode(failure),
		StatusReasonCode:     serviceFailureReasonCode(failure),
		EventDate:            eventTimestamp,
		EventTime:            eventTimestamp,
		ExceptionCode:        serviceFailureExceptionCode(failure),
		ReasonCode:           serviceFailureReasonCode(failure),
		ReasonDescription:    serviceFailureReasonDescription(failure),
		LateMinutes:          &failure.LateMinutes,
		ServiceFailureID:     &failure.ID,
		ServiceFailureNumber: failure.Number,
		ServiceFailureStatus: string(failure.Status),
		References:           partnerReferences(shipmentBOL(source), shipmentProNumber(source)),
	}
	applyShipmentStatusShipment(&status, source)
	if failure.ReasonCodeID != nil {
		status.ServiceFailureReasonCodeID = failure.ReasonCodeID
	}
	if failure.ReasonCode != nil {
		status.ServiceFailureReasonCode = failure.ReasonCode.Code
		if status.ReasonDescription == "" {
			status.ReasonDescription = stringutils.FirstNonEmpty(
				failure.ReasonCode.Label,
				failure.ReasonCode.Description,
			)
		}
	}
	move := serviceFailureMove(source, failure.ShipmentMoveID)
	stop := serviceFailureStop(source, failure.StopID)
	if stop == nil {
		stop = failure.Stop
	}
	applyShipmentStatusStop(&status, stop)
	applyServiceFailureEquipment(&status, source, move)

	return edipayload.DocumentPayload{
		TransactionSet: edidocument.TransactionSet214,
		ShipmentStatus: &status,
	}
}

func serviceFailureStatusCode(failure *servicefailure.ServiceFailure) string {
	if failure == nil {
		return "SD"
	}
	if value := NormalizedX12Code(failure.X12StatusCodeOverride); value != "" {
		return value
	}
	if failure.ReasonCode != nil {
		if value := NormalizedX12Code(failure.ReasonCode.DefaultStatusCode); value != "" {
			return value
		}
	}
	return "SD"
}

func serviceFailureReasonCode(failure *servicefailure.ServiceFailure) string {
	if failure == nil {
		return ""
	}
	if value := NormalizedX12Code(failure.X12ReasonCodeOverride); value != "" {
		return value
	}
	if failure.ReasonCode != nil {
		return NormalizedX12Code(failure.ReasonCode.DefaultReasonCode)
	}
	return ""
}

func serviceFailureExceptionCode(failure *servicefailure.ServiceFailure) string {
	if failure == nil {
		return ""
	}
	if value := NormalizedX12Code(failure.X12ExceptionCode); value != "" {
		return value
	}
	if failure.ReasonCode != nil {
		return NormalizedX12Code(failure.ReasonCode.DefaultExceptionCode)
	}
	return ""
}

func serviceFailureEventTimestamp(failure *servicefailure.ServiceFailure) int64 {
	if failure == nil {
		return 0
	}
	if failure.DetectedAt > 0 {
		return failure.DetectedAt
	}
	if failure.CreatedAt > 0 {
		return failure.CreatedAt
	}
	return failure.ActualArrival
}

func serviceFailureReasonDescription(failure *servicefailure.ServiceFailure) string {
	if failure == nil || failure.ReasonCode == nil {
		return ""
	}
	return stringutils.FirstNonEmpty(failure.ReasonCode.Label, failure.ReasonCode.Description)
}

func serviceFailureMove(source *shipment.Shipment, moveID pulid.ID) *shipment.ShipmentMove {
	if source == nil {
		return nil
	}
	for _, move := range source.Moves {
		if move != nil && move.ID == moveID {
			return move
		}
	}
	return nil
}

func serviceFailureStop(source *shipment.Shipment, stopID pulid.ID) *shipment.Stop {
	if source == nil {
		return nil
	}
	for _, move := range source.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop != nil && stop.ID == stopID {
				return stop
			}
		}
	}
	return nil
}

func applyServiceFailureEquipment(
	payload *edipayload.ShipmentStatusPayload,
	source *shipment.Shipment,
	move *shipment.ShipmentMove,
) {
	if payload == nil {
		return
	}
	if move != nil && move.Assignment != nil && move.Assignment.Trailer != nil {
		payload.EquipmentNumber = move.Assignment.Trailer.Code
	}
	if source != nil && source.TrailerType != nil {
		payload.EquipmentType = source.TrailerType.Code
	}
}

func ServiceFailurePayloadDiagnostics(
	payload *edipayload.ShipmentStatusPayload,
) []edidocument.Diagnostic {
	return serviceFailure214Diagnostics(payload)
}

func BuildTenderResponsePayload(transfer *editransfer.EDITransfer) edipayload.DocumentPayload {
	responseCode := "A"
	reasonCode := ""
	if transfer.Status == editransfer.TransferStatusRejected {
		responseCode = "D"
		reasonCode = "ZZ"
	}
	payload := edipayload.TenderResponsePayload{
		TransferID:      transfer.ID,
		ShipmentID:      transfer.SourceShipmentID,
		BOL:             transfer.TenderPayload.BOL,
		ResponseCode:    responseCode,
		ReasonCode:      reasonCode,
		RejectionReason: transfer.RejectionReason,
		Status:          transfer.Status,
	}
	return edipayload.DocumentPayload{
		TransactionSet: edidocument.TransactionSet990,
		TenderResponse: &payload,
	}
}

const dispatchedStatusCode = "P1"

func shipmentEventStatusCode(
	event *shipmentevent.Event,
	stop *shipment.Stop,
	final bool,
) string {
	//nolint:exhaustive // Unmapped shipment events intentionally fall back to the generic status code.
	switch event.Type {
	case shipmentevent.TypeStopArrived:
		return stopStatusCode(stop, false, false)
	case shipmentevent.TypeStopDeparted:
		return stopStatusCode(stop, true, final)
	case shipmentevent.TypeMoveDeparted:
		return stopStatusCode(stop, stopDeparted(stop, event.OccurredAt), false)
	case shipmentevent.TypeMoveArrived:
		return stopStatusCode(stop, true, true)
	case shipmentevent.TypeShipmentCanceled:
		return "A7"
	case shipmentevent.TypeDriverAssigned, shipmentevent.TypeCarrierAssigned:
		return dispatchedStatusCode
	case shipmentevent.TypeStatusChanged:
		//nolint:exhaustive // Only statuses with EDI 214-specific codes need overrides.
		switch shipmentvocab.Status(metadataString(event.Metadata, "newStatus")) {
		case shipmentvocab.StatusAssigned, shipmentvocab.StatusPartiallyAssigned:
			return dispatchedStatusCode
		case shipmentvocab.StatusInTransit:
			return "AF"
		case shipmentvocab.StatusCompleted,
			shipmentvocab.StatusInvoiced,
			shipmentvocab.StatusReadyToInvoice:
			return "D1"
		case shipmentvocab.StatusDelayed:
			return "SD"
		case shipmentvocab.StatusCanceled:
			return "A7"
		}
	}
	return "X3"
}

func completesShipment(source *shipment.Shipment, stop *shipment.Stop) bool {
	if source == nil || stop == nil {
		return false
	}
	var last *shipment.ShipmentMove
	for _, move := range source.Moves {
		if move == nil || move.Status == shipmentvocab.MoveStatusCanceled ||
			len(move.ActiveStopsBySequence()) == 0 {
			continue
		}
		if last == nil || move.Sequence >= last.Sequence {
			last = move
		}
	}
	if last == nil {
		return false
	}
	stops := last.ActiveStopsBySequence()
	return stops[len(stops)-1].ID == stop.ID
}

func stopDeparted(stop *shipment.Stop, occurredAt int64) bool {
	return stop != nil && stop.ActualDeparture != nil && *stop.ActualDeparture <= occurredAt
}

func stopStatusCode(stop *shipment.Stop, departed, completesMove bool) string {
	pickup := stop != nil && stop.IsOriginStop()
	if stop == nil {
		pickup = !departed
	}
	switch {
	case pickup && departed:
		return "AF"
	case pickup:
		return "X3"
	case !departed:
		return "X1"
	case completesMove:
		return "D1"
	default:
		return "CD"
	}
}

func applyShipmentStatusShipment(
	payload *edipayload.ShipmentStatusPayload,
	source *shipment.Shipment,
) {
	if source == nil {
		return
	}
	payload.BOL = source.BOL
	payload.ProNumber = source.ProNumber
	payload.ShipperReference = partnerReference(source)
	payload.Weight = source.Weight
	payload.Pieces = source.Pieces
}

func applyShipmentStatusStop(payload *edipayload.ShipmentStatusPayload, stop *shipment.Stop) {
	if payload == nil || stop == nil {
		return
	}

	payload.StopID = stop.ID
	payload.StopType = string(stop.Type)
	payload.StopSequence = stop.Sequence
	payload.LocationID = stop.LocationID
	payload.AddressLine = stop.AddressLine
	payload.ScheduledWindowStart = stop.ScheduledWindowStart
	payload.ScheduledWindowEnd = stop.ScheduledWindowEnd
	payload.ActualArrival = stop.ActualArrival
	payload.ActualDeparture = stop.ActualDeparture

	if stop.Location == nil {
		return
	}

	payload.LocationName = stop.Location.Name
	payload.LocationCode = stop.Location.Code
	payload.AddressLine = stringutils.FirstNonEmpty(
		payload.AddressLine,
		locationAddress(stop.Location),
	)
	payload.City = stop.Location.City
	payload.PostalCode = stop.Location.PostalCode
	if stop.Location.State != nil {
		payload.StateCode = stop.Location.State.Abbreviation
	}
}

func shipmentEventStop(event *shipmentevent.Event, source *shipment.Shipment) *shipment.Stop {
	if source == nil {
		return nil
	}
	var fallback *shipment.Stop
	for _, move := range source.Moves {
		if move == nil {
			continue
		}
		stops := move.ActiveStopsBySequence()
		for _, stop := range stops {
			if event.StopID.IsNotNil() && stop.ID == event.StopID {
				return stop
			}
		}
		if event.MoveID.IsNil() || move.ID != event.MoveID || len(stops) == 0 {
			continue
		}
		fallback = stops[0]
		if event.Type == shipmentevent.TypeMoveArrived {
			fallback = stops[len(stops)-1]
		}
	}
	if fallback != nil {
		return fallback
	}
	if len(source.Moves) > 0 && source.Moves[0] != nil {
		if stops := source.Moves[0].ActiveStopsBySequence(); len(stops) > 0 {
			return stops[0]
		}
	}
	return nil
}

func metadataInt64(metadata map[string]any, key string) (int64, bool) {
	value, ok := metadata[key]
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case float32:
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func metadataString(metadata map[string]any, key string) string {
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func partnerReferences(bol, proNumber string) map[string]string {
	references := make(map[string]string, 3)
	if bol = strings.TrimSpace(bol); bol != "" {
		references[edipayload.RefBillOfLading] = bol
	}
	if proNumber = strings.TrimSpace(proNumber); proNumber != "" {
		references[edipayload.RefCarrierPRO] = proNumber
	}
	return references
}

func partnerReference(source *shipment.Shipment) string {
	if source == nil {
		return ""
	}
	return stringutils.FirstNonEmpty(strings.TrimSpace(source.ExternalReference), source.BOL)
}

func shipmentBOL(source *shipment.Shipment) string {
	if source == nil {
		return ""
	}
	return source.BOL
}

func shipmentFreightTerms(source *shipment.Shipment) string {
	if source == nil {
		return ""
	}
	return string(source.FreightTerms)
}

func shipmentProNumber(source *shipment.Shipment) string {
	if source == nil {
		return ""
	}
	return source.ProNumber
}

func SourceLabelIndex(
	payload *editransfer.LoadTenderPayload,
) map[edi.MappingEntityType]map[pulid.ID]string {
	labels := map[edi.MappingEntityType]map[pulid.ID]string{}
	setLabel(labels, edi.MappingEntityTypeCustomer, payload.CustomerID, payload.CustomerLabel)
	setLabel(
		labels,
		edi.MappingEntityTypeServiceType,
		payload.ServiceTypeID,
		payload.ServiceTypeLabel,
	)
	setLabel(
		labels,
		edi.MappingEntityTypeShipmentType,
		payload.ShipmentTypeID,
		payload.ShipmentTypeLabel,
	)
	setLabel(
		labels,
		edi.MappingEntityTypeFormulaTemplate,
		payload.FormulaTemplateID,
		payload.FormulaTemplateLabel,
	)

	for _, move := range payload.Moves {
		for i := range move.Stops {
			stop := &move.Stops[i]
			setLabel(labels, edi.MappingEntityTypeLocation, stop.LocationID, stopLabel(stop))
		}
	}
	for _, commodity := range payload.Commodities {
		setLabel(
			labels,
			edi.MappingEntityTypeCommodity,
			commodity.CommodityID,
			commodity.CommodityLabel,
		)
	}
	for _, charge := range payload.AdditionalCharges {
		setLabel(
			labels,
			edi.MappingEntityTypeAccessorialCharge,
			charge.AccessorialChargeID,
			charge.AccessorialLabel,
		)
	}

	return labels
}

func addRequiredID(
	required map[edi.MappingEntityType][]pulid.ID,
	entityType edi.MappingEntityType,
	id pulid.ID,
) {
	if id.IsNil() {
		return
	}
	if slices.Contains(required[entityType], id) {
		return
	}
	required[entityType] = append(required[entityType], id)
}

func setLabel(
	labels map[edi.MappingEntityType]map[pulid.ID]string,
	entityType edi.MappingEntityType,
	id pulid.ID,
	label string,
) {
	if id.IsNil() || strings.TrimSpace(label) == "" {
		return
	}
	if _, ok := labels[entityType]; !ok {
		labels[entityType] = map[pulid.ID]string{}
	}
	labels[entityType][id] = label
}

func customerLabel(source *shipment.Shipment) string {
	if source.Customer == nil {
		return ""
	}

	return joinCodeName(source.Customer.Code, source.Customer.Name)
}

func serviceTypeLabel(source *shipment.Shipment) string {
	if source.ServiceType == nil {
		return ""
	}

	return stringutils.FirstNonEmpty(source.ServiceType.Code, source.ServiceType.Description)
}

func shipmentTypeLabel(source *shipment.Shipment) string {
	if source.ShipmentType == nil {
		return ""
	}

	return stringutils.FirstNonEmpty(source.ShipmentType.Code, source.ShipmentType.Description)
}

func formulaTemplateLabel(source *shipment.Shipment) string {
	if source.FormulaTemplate == nil {
		return ""
	}

	return source.FormulaTemplate.Name
}

func stopLocationDetails(stop *shipment.Stop) (label, address string) {
	if stop.Location == nil {
		return "", ""
	}

	address = locationAddress(stop.Location)
	label = joinCodeName(stop.Location.Code, stop.Location.Name)
	return stringutils.FirstNonEmpty(label, address), address
}

func stopLabel(stop *editransfer.LoadTenderStop) string {
	return stringutils.FirstNonEmpty(
		stop.LocationLabel,
		joinCodeName(stop.LocationCode, stop.LocationName),
		stop.AddressLine,
		stop.LocationID.String(),
	)
}

func commodityLabel(commodity *shipment.ShipmentCommodity) string {
	if commodity.Commodity == nil {
		return ""
	}

	return stringutils.FirstNonEmpty(commodity.Commodity.Name, commodity.Commodity.Description)
}

func commodityName(commodity *shipment.ShipmentCommodity) string {
	if commodity.Commodity == nil {
		return ""
	}

	return commodity.Commodity.Name
}

func commodityHazmat(commodity *shipment.ShipmentCommodity) bool {
	return commodity.Commodity != nil && commodity.Commodity.HazardousMaterialID.IsNotNil()
}

func commodityDescription(commodity *shipment.ShipmentCommodity) string {
	if commodity.Commodity == nil {
		return ""
	}
	return commodity.Commodity.Description
}

func accessorialLabel(charge *shipment.AdditionalCharge) string {
	if charge.AccessorialCharge == nil {
		return ""
	}
	return stringutils.FirstNonEmpty(
		joinCodeName(charge.AccessorialCharge.Code, charge.AccessorialCharge.Description),
		charge.AccessorialCharge.Code,
		charge.AccessorialCharge.Description,
	)
}

func accessorialCode(charge *shipment.AdditionalCharge) string {
	if charge.AccessorialCharge == nil {
		return ""
	}

	return charge.AccessorialCharge.Code
}

func accessorialDescription(charge *shipment.AdditionalCharge) string {
	if charge.AccessorialCharge == nil {
		return ""
	}

	return charge.AccessorialCharge.Description
}

func locationAddress(loc *location.Location) string {
	if loc == nil {
		return ""
	}

	state := ""
	if loc.State != nil {
		state = loc.State.Abbreviation
	}

	return strings.Join(stringutils.NonEmptyStrings(
		loc.AddressLine1,
		loc.AddressLine2,
		strings.Join(stringutils.NonEmptyStrings(loc.City, state, loc.PostalCode), ", "),
	), ", ")
}

func joinCodeName(code, name string) string {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if code != "" && name != "" {
		return code + " - " + name
	}
	return stringutils.FirstNonEmpty(code, name)
}

const x12FreightChargeCode = "400"

func specialChargeCode(line *invoice.InvoiceLine) string {
	if line.Type == invoice.InvoiceLineTypeFreight {
		return x12FreightChargeCode
	}
	return strings.TrimSpace(line.ChargeCode)
}
