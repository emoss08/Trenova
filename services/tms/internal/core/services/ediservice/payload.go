package ediservice

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

//nolint:funlen // Tender payload construction intentionally mirrors the outbound document shape.
func buildTenderPayload(source *shipment.Shipment) edi.LoadTenderPayload {
	payload := edi.LoadTenderPayload{
		ShipmentID:               source.ID,
		BusinessUnitID:           source.BusinessUnitID,
		OrganizationID:           source.OrganizationID,
		ServiceTypeID:            source.ServiceTypeID,
		ServiceTypeLabel:         serviceTypeLabel(source),
		ShipmentTypeID:           source.ShipmentTypeID,
		ShipmentTypeLabel:        shipmentTypeLabel(source),
		CustomerID:               source.CustomerID,
		CustomerLabel:            customerLabel(source),
		FormulaTemplateID:        source.FormulaTemplateID,
		FormulaTemplateLabel:     formulaTemplateLabel(source),
		BOL:                      source.BOL,
		Pieces:                   source.Pieces,
		Weight:                   source.Weight,
		TemperatureMin:           source.TemperatureMin,
		TemperatureMax:           source.TemperatureMax,
		FreightChargeAmount:      source.FreightChargeAmount,
		OtherChargeAmount:        source.OtherChargeAmount,
		BaseRate:                 source.BaseRate,
		TotalChargeAmount:        source.TotalChargeAmount,
		RatingUnit:               source.RatingUnit,
		Moves:                    make([]edi.LoadTenderMove, 0, len(source.Moves)),
		Commodities:              make([]edi.LoadTenderCommodity, 0, len(source.Commodities)),
		AdditionalCharges:        make([]edi.LoadTenderCharge, 0, len(source.AdditionalCharges)),
		RequiredMappingEntityIDs: map[edi.MappingEntityType][]pulid.ID{},
	}
	if source.RatingDetail != nil {
		payload.RatingDetail = jsonutils.MustToJSON(source.RatingDetail)
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
		tenderMove := edi.LoadTenderMove{
			Loaded:   move.Loaded,
			Sequence: move.Sequence,
			Distance: move.Distance,
			Stops:    make([]edi.LoadTenderStop, 0, len(move.Stops)),
		}
		for _, stop := range move.Stops {
			locLabel, locAddr := stopLocationDetails(stop)
			tenderMove.Stops = append(tenderMove.Stops, edi.LoadTenderStop{
				LocationID:           stop.LocationID,
				LocationLabel:        locLabel,
				Type:                 string(stop.Type),
				ScheduleType:         string(stop.ScheduleType),
				Sequence:             stop.Sequence,
				Pieces:               stop.Pieces,
				Weight:               stop.Weight,
				ScheduledWindowStart: stop.ScheduledWindowStart,
				ScheduledWindowEnd:   stop.ScheduledWindowEnd,
				AddressLine:          stringutils.FirstNonEmpty(stop.AddressLine, locAddr),
			})
			if stop.Location != nil {
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationName = stop.Location.Name
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationCode = stop.Location.Code
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationAddressLine1 = stop.Location.AddressLine1
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationAddressLine2 = stop.Location.AddressLine2
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationCity = stop.Location.City
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationPostalCode = stop.Location.PostalCode
				if stop.Location.State != nil {
					tenderMove.Stops[len(tenderMove.Stops)-1].LocationStateCode = stop.Location.State.Abbreviation
				}
			}
			addRequiredID(
				payload.RequiredMappingEntityIDs,
				edi.MappingEntityTypeLocation,
				stop.LocationID,
			)
		}
		payload.Moves = append(payload.Moves, tenderMove)
	}

	for _, commodity := range source.Commodities {
		payload.Commodities = append(payload.Commodities, edi.LoadTenderCommodity{
			CommodityID:          commodity.CommodityID,
			CommodityLabel:       commodityLabel(commodity),
			CommodityName:        commodityName(commodity),
			CommodityDescription: commodityDescription(commodity),
			Weight:               commodity.Weight,
			Pieces:               commodity.Pieces,
		})
		addRequiredID(
			payload.RequiredMappingEntityIDs,
			edi.MappingEntityTypeCommodity,
			commodity.CommodityID,
		)
	}

	for _, charge := range source.AdditionalCharges {
		payload.AdditionalCharges = append(payload.AdditionalCharges, edi.LoadTenderCharge{
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

func buildFreightInvoicePayload(source *invoice.Invoice) edi.DocumentPayload {
	payload := edi.FreightInvoicePayload{
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
		BillToName:         source.BillToName,
		BillToAddressLine1: source.BillToAddressLine1,
		BillToAddressLine2: source.BillToAddressLine2,
		BillToCity:         source.BillToCity,
		BillToStateCode:    source.BillToState,
		BillToPostalCode:   source.BillToPostalCode,
		BillToCountry:      source.BillToCountry,
		CurrencyCode:       source.CurrencyCode,
		TotalAmount: decimal.NullDecimal{
			Decimal: source.TotalAmount,
			Valid:   true,
		},
		LineCharges: make([]edi.FreightInvoiceCharge, 0, len(source.Lines)),
		ReferenceNumbers: map[string]string{
			"qualifier": "BM",
			"bol": stringutils.FirstNonEmpty(
				source.ShipmentBOL,
				shipmentBOL(source.Shipment),
			),
			"pro": stringutils.FirstNonEmpty(
				source.ShipmentProNumber,
				shipmentProNumber(source.Shipment),
			),
			"shipmentId": source.ShipmentID.String(),
		},
	}

	lines := make([]*invoice.InoviceLine, 0, len(source.Lines))
	for _, line := range source.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, line)
	}
	slices.SortFunc(lines, func(a, b *invoice.InoviceLine) int {
		return cmp.Compare(a.LineNumber, b.LineNumber)
	})

	for _, line := range lines {
		payload.LineCharges = append(payload.LineCharges, edi.FreightInvoiceCharge{
			Sequence:    int64(line.LineNumber),
			Code:        string(line.Type),
			Description: line.Description,
			Amount:      line.Amount,
		})
	}

	return edi.DocumentPayload{
		TransactionSet: edi.TransactionSet210,
		FreightInvoice: &payload,
	}
}

func buildShipmentStatusPayload(source *shipment.Shipment) edi.DocumentPayload {
	if source == nil {
		return edi.DocumentPayload{
			TransactionSet: edi.TransactionSet214,
			ShipmentStatus: &edi.ShipmentStatusPayload{References: map[string]string{}},
		}
	}
	payload := edi.ShipmentStatusPayload{
		ShipmentID: source.ID,
		BOL:        source.BOL,
		ProNumber:  source.ProNumber,
		StatusCode: "X3",
		References: map[string]string{
			"shipmentId": source.ID.String(),
			"bol":        source.BOL,
			"pro":        source.ProNumber,
		},
	}
	if len(source.Moves) > 0 &&
		source.Moves[0] != nil &&
		len(source.Moves[0].Stops) > 0 &&
		source.Moves[0].Stops[0] != nil {
		stop := source.Moves[0].Stops[0]
		payload.EventDate = stop.ScheduledWindowStart
		payload.EventTime = stop.ScheduledWindowStart
		applyShipmentStatusStop(&payload, stop)
	}
	return edi.DocumentPayload{
		TransactionSet: edi.TransactionSet214,
		ShipmentStatus: &payload,
	}
}

func buildShipmentEventStatusPayload(
	event *shipmentevent.Event,
	source *shipment.Shipment,
) edi.DocumentPayload {
	payload := edi.ShipmentStatusPayload{
		ShipmentID: event.ShipmentID,
		BOL:        shipmentBOL(source),
		ProNumber:  shipmentProNumber(source),
		StatusCode: shipmentEventStatusCode(event),
		EventDate:  event.OccurredAt,
		EventTime:  event.OccurredAt,
		References: map[string]string{
			"shipmentId": event.ShipmentID.String(),
			"eventId":    event.ID.String(),
			"eventType":  string(event.Type),
			"bol":        shipmentBOL(source),
			"pro":        shipmentProNumber(source),
		},
	}
	if event.Summary != "" {
		payload.References["summary"] = event.Summary
	}
	if reason := stringutils.FirstNonEmpty(
		metadataString(event.Metadata, "statusReasonCode"),
		metadataString(event.Metadata, "reasonCode"),
		metadataString(event.Metadata, "reason"),
	); reason != "" {
		payload.StatusReasonCode = reason
		payload.ReasonCode = reason
	}
	if reasonDescription := metadataString(event.Metadata, "reasonDescription"); reasonDescription != "" {
		payload.ReasonDescription = reasonDescription
	}
	if exceptionCode := metadataString(event.Metadata, "exceptionCode"); exceptionCode != "" {
		payload.ExceptionCode = exceptionCode
	}
	if lateMinutes, ok := metadataInt64(event.Metadata, "lateMinutes"); ok {
		payload.LateMinutes = &lateMinutes
	}
	stop := shipmentEventStop(event, source)
	applyShipmentStatusStop(&payload, stop)
	return edi.DocumentPayload{
		TransactionSet: edi.TransactionSet214,
		ShipmentStatus: &payload,
	}
}

func (s *Service) BuildShipmentStatusPayloadForServiceFailure(
	ctx context.Context,
	req *services.BuildServiceFailureEDIPayloadRequest,
) (*services.ServiceFailureEDIPayloadResult, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	failure, err := s.serviceFailureRepo.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
		ID:         req.ServiceFailureID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	source, err := s.shipmentSvc.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:         failure.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	payload := buildServiceFailureShipmentStatusPayload(failure, source)
	diagnostics := serviceFailurePayloadDiagnostics(payload.ShipmentStatus)
	return &services.ServiceFailureEDIPayloadResult{
		Payload:     payload,
		Diagnostics: diagnostics,
	}, nil
}

func buildServiceFailureShipmentStatusPayload(
	failure *servicefailure.ServiceFailure,
	source *shipment.Shipment,
) edi.DocumentPayload {
	eventTimestamp := serviceFailureEventTimestamp(failure)
	status := edi.ShipmentStatusPayload{
		ShipmentID:           failure.ShipmentID,
		BOL:                  shipmentBOL(source),
		ProNumber:            shipmentProNumber(source),
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
		References: map[string]string{
			"shipmentId":       failure.ShipmentID.String(),
			"serviceFailureId": failure.ID.String(),
			"serviceFailure":   failure.Number,
			"bol":              shipmentBOL(source),
			"pro":              shipmentProNumber(source),
			"type":             string(failure.Type),
			"status":           string(failure.Status),
		},
	}
	if failure.ReasonCodeID != nil {
		status.ServiceFailureReasonCodeID = failure.ReasonCodeID
	}
	if failure.ReasonCode != nil {
		status.ServiceFailureReasonCode = failure.ReasonCode.Code
		if status.ReasonDescription == "" {
			status.ReasonDescription = stringutils.FirstNonEmpty(failure.ReasonCode.Label, failure.ReasonCode.Description)
		}
	}
	if strings.TrimSpace(failure.Notes) != "" {
		status.References["notes"] = strings.TrimSpace(failure.Notes)
	}

	move := serviceFailureMove(source, failure.ShipmentMoveID)
	stop := serviceFailureStop(source, failure.StopID)
	if stop == nil {
		stop = failure.Stop
	}
	applyShipmentStatusStop(&status, stop)
	applyServiceFailureEquipment(&status, source, move)

	return edi.DocumentPayload{
		TransactionSet: edi.TransactionSet214,
		ShipmentStatus: &status,
	}
}

func serviceFailureStatusCode(failure *servicefailure.ServiceFailure) string {
	if failure == nil {
		return "SD"
	}
	if value := strings.TrimSpace(failure.X12StatusCodeOverride); value != "" {
		return strings.ToUpper(value)
	}
	if failure.ReasonCode != nil && strings.TrimSpace(failure.ReasonCode.DefaultStatusCode) != "" {
		return strings.ToUpper(strings.TrimSpace(failure.ReasonCode.DefaultStatusCode))
	}
	return "SD"
}

func serviceFailureReasonCode(failure *servicefailure.ServiceFailure) string {
	if failure == nil {
		return ""
	}
	if value := strings.TrimSpace(failure.X12ReasonCodeOverride); value != "" {
		return strings.ToUpper(value)
	}
	if failure.ReasonCode != nil {
		return strings.ToUpper(strings.TrimSpace(failure.ReasonCode.DefaultReasonCode))
	}
	return ""
}

func serviceFailureExceptionCode(failure *servicefailure.ServiceFailure) string {
	if failure == nil {
		return ""
	}
	if value := strings.TrimSpace(failure.X12ExceptionCode); value != "" {
		return strings.ToUpper(value)
	}
	if failure.ReasonCode != nil {
		return strings.ToUpper(strings.TrimSpace(failure.ReasonCode.DefaultExceptionCode))
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
	payload *edi.ShipmentStatusPayload,
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

func serviceFailurePayloadDiagnostics(payload *edi.ShipmentStatusPayload) []edix12.Diagnostic {
	if payload == nil ||
		strings.ToUpper(strings.TrimSpace(payload.StatusCode)) != "SD" ||
		strings.TrimSpace(payload.StatusReasonCode) != "" {
		return nil
	}
	return []edix12.Diagnostic{
		{
			Severity:        edi.ValidationSeverityError,
			Code:            "required",
			SegmentID:       "AT7",
			ElementPosition: 2,
			Path:            "shipmentStatus.statusReasonCode",
			Message:         "X12 214 service failure status code SD requires a status reason code",
			SuggestedFix:    "Set an override reason code or configure a default reason code on the service failure reason code.",
		},
	}
}

func buildTenderResponsePayload(transfer *edi.EDITransfer) edi.DocumentPayload {
	responseCode := "A"
	reasonCode := ""
	if transfer.Status == edi.TransferStatusRejected {
		responseCode = "D"
		reasonCode = "ZZ"
	}
	payload := edi.TenderResponsePayload{
		TransferID:      transfer.ID,
		ShipmentID:      transfer.SourceShipmentID,
		BOL:             transfer.TenderPayload.BOL,
		ResponseCode:    responseCode,
		ReasonCode:      reasonCode,
		RejectionReason: transfer.RejectionReason,
		Status:          transfer.Status,
	}
	return edi.DocumentPayload{
		TransactionSet: edi.TransactionSet990,
		TenderResponse: &payload,
	}
}

func shipmentEventStatusCode(event *shipmentevent.Event) string {
	//nolint:exhaustive // Unmapped shipment events intentionally fall back to the generic status code.
	switch event.Type {
	case shipmentevent.TypeMoveDeparted:
		return "AF"
	case shipmentevent.TypeMoveArrived:
		return "X1"
	case shipmentevent.TypeStopCompleted:
		return "D1"
	case shipmentevent.TypeShipmentCanceled:
		return "A7"
	case shipmentevent.TypeStatusChanged:
		//nolint:exhaustive // Only statuses with EDI 214-specific codes need overrides.
		switch shipment.Status(metadataString(event.Metadata, "newStatus")) {
		case shipment.StatusInTransit:
			return "AF"
		case shipment.StatusCompleted, shipment.StatusInvoiced, shipment.StatusReadyToInvoice:
			return "D1"
		case shipment.StatusDelayed:
			return "A3"
		case shipment.StatusCanceled:
			return "A7"
		}
	}
	return "X3"
}

func applyShipmentStatusStop(payload *edi.ShipmentStatusPayload, stop *shipment.Stop) {
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
	payload.AddressLine = stringutils.FirstNonEmpty(payload.AddressLine, locationAddress(stop.Location))
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
	var moveFallback *shipment.Stop
	for _, move := range source.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			if event.StopID.IsNotNil() && stop.ID == event.StopID {
				return stop
			}
			if event.MoveID.IsNotNil() && move.ID == event.MoveID && moveFallback == nil {
				moveFallback = stop
			}
		}
	}
	if moveFallback != nil {
		return moveFallback
	}
	if len(source.Moves) > 0 && source.Moves[0] != nil && len(source.Moves[0].Stops) > 0 {
		return source.Moves[0].Stops[0]
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

func shipmentBOL(source *shipment.Shipment) string {
	if source == nil {
		return ""
	}
	return source.BOL
}

func shipmentProNumber(source *shipment.Shipment) string {
	if source == nil {
		return ""
	}
	return source.ProNumber
}

func (s *Service) buildAcknowledgmentPayload(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
	transactionSet edi.TransactionSet,
) (edi.DocumentPayload, error) {
	message, err := s.messageRepo.GetMessageByID(ctx, repositories.GetEDIMessageByIDRequest{
		ID:         req.SourceMessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return edi.DocumentPayload{}, err
	}
	ack := edi.FunctionalAcknowledgmentPayload{
		SourceMessageID:                  message.ID,
		OriginalFunctionalGroupID:        edi.FunctionalGroupDefault(message.TransactionSet),
		OriginalGroupControlNumber:       message.GroupControlNumber,
		OriginalTransactionSet:           message.TransactionSet,
		OriginalTransactionControlNumber: message.TransactionControlNumber,
		GroupAcknowledgmentCode:          "A",
		TransactionAcknowledgmentCode:    "A",
		AcceptedTransactionSetCount:      1,
		ReceivedTransactionSetCount:      1,
		IncludedTransactionSetCount:      1,
	}
	if message.PartnerDocumentProfile != nil {
		ack.OriginalFunctionalGroupID = message.PartnerDocumentProfile.FunctionalGroupID
	}
	if transactionSet == edi.TransactionSet999 {
		implementation := edi.ImplementationAckPayload(ack)
		return edi.DocumentPayload{
			TransactionSet:               edi.TransactionSet999,
			ImplementationAcknowledgment: &implementation,
		}, nil
	}
	return edi.DocumentPayload{
		TransactionSet:           edi.TransactionSet997,
		FunctionalAcknowledgment: &ack,
	}, nil
}

func sourceLabelIndex(
	payload *edi.LoadTenderPayload,
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

func stopLabel(stop *edi.LoadTenderStop) string {
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
