package ediinboundservice

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/edix12inspect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

func parseInterchange(rawX12 string) (*parsedInterchange, error) {
	if strings.TrimSpace(rawX12) == "" {
		return nil, errors.New("inbound file is empty")
	}
	inspection := edix12inspect.InspectX12(&edix12inspect.InspectX12Request{RawX12: rawX12})
	if len(inspection.Segments) == 0 {
		return nil, errors.New("inbound file does not contain any X12 segments")
	}
	isa := findSegment(inspection.Segments, "ISA")
	if isa == nil {
		return nil, errors.New("inbound file does not start with an ISA interchange header")
	}

	interchange := &parsedInterchange{
		inspection:        inspection,
		controlNumber:     strings.TrimSpace(elementValue(isa, 13)),
		senderQualifier:   strings.TrimSpace(elementValue(isa, 5)),
		senderID:          strings.TrimSpace(elementValue(isa, 6)),
		receiverQualifier: strings.TrimSpace(elementValue(isa, 7)),
		receiverID:        strings.TrimSpace(elementValue(isa, 8)),
	}
	if gs := findSegment(inspection.Segments, "GS"); gs != nil {
		interchange.functionalGroupID = strings.TrimSpace(elementValue(gs, 1))
		interchange.applicationSender = strings.TrimSpace(elementValue(gs, 2))
		interchange.applicationReceiver = strings.TrimSpace(elementValue(gs, 3))
		interchange.groupControlNumber = strings.TrimSpace(elementValue(gs, 6))
	}
	if len(inspection.Transactions) == 0 {
		return nil, errors.New("inbound file does not contain any X12 transactions")
	}

	interchange.transactions = make([]parsedTransaction, 0, len(inspection.Transactions))
	for index := range inspection.Transactions {
		transaction := &inspection.Transactions[index]
		segments := transactionSegments(inspection.Segments, transaction)
		if len(segments) == 0 {
			return nil, fmt.Errorf(
				"transaction %d does not contain any segments",
				transaction.Index,
			)
		}
		raw := strings.Builder{}
		for segmentIndex := range segments {
			raw.WriteString(segments[segmentIndex].RawWithTerminator)
		}
		interchange.transactions = append(interchange.transactions, parsedTransaction{
			set:                edi.TransactionSet(transaction.TransactionSet),
			controlNumber:      strings.TrimSpace(transaction.STControlNumber),
			groupControlNumber: interchange.groupControlNumber,
			functionalGroupID:  interchange.functionalGroupID,
			segments:           segments,
			raw:                raw.String(),
		})
	}
	return interchange, nil
}

func transactionSegments(
	segments []edix12inspect.X12Segment,
	transaction *edix12inspect.X12Transaction,
) []edix12inspect.X12Segment {
	result := make([]edix12inspect.X12Segment, 0, transaction.ActualSegments+2)
	end := transaction.EndSegmentIndex
	if end == 0 {
		end = len(segments) - 1
	}
	for index := range segments {
		if segments[index].Index >= transaction.StartSegmentIndex &&
			segments[index].Index <= end {
			result = append(result, segments[index])
		}
	}
	return result
}

func findSegment(segments []edix12inspect.X12Segment, segmentID string) *edix12inspect.X12Segment {
	for index := range segments {
		if segments[index].SegmentID == segmentID {
			return &segments[index]
		}
	}
	return nil
}

func findSegmentsByID(
	segments []edix12inspect.X12Segment,
	segmentID string,
) []edix12inspect.X12Segment {
	result := make([]edix12inspect.X12Segment, 0, 4)
	for index := range segments {
		if segments[index].SegmentID == segmentID {
			result = append(result, segments[index])
		}
	}
	return result
}

func elementValue(segment *edix12inspect.X12Segment, position int) string {
	if segment == nil {
		return ""
	}
	for index := range segment.Elements {
		if segment.Elements[index].Position == position {
			return segment.Elements[index].Value
		}
	}
	return ""
}

func (t *parsedTransaction) documentPayload() edi.DocumentPayload {
	switch t.set {
	case edi.TransactionSet204:
		payload := parseLoadTender(t)
		return edi.DocumentPayload{
			TransactionSet: edi.TransactionSet204,
			PurposeCode:    payload.PurposeCode,
			LoadTender:     &payload,
		}
	case edi.TransactionSet210:
		payload := parseFreightInvoice(t)
		return edi.DocumentPayload{
			TransactionSet: edi.TransactionSet210,
			FreightInvoice: &payload,
		}
	case edi.TransactionSet990:
		details := parseTenderResponse(t)
		return edi.DocumentPayload{
			TransactionSet: edi.TransactionSet990,
			TenderResponse: &edi.TenderResponsePayload{
				BOL:             details.shipmentRef,
				ResponseCode:    details.reservationCode,
				RejectionReason: details.remarks,
			},
		}
	case edi.TransactionSet214:
		details := parseShipmentStatus(t)
		return edi.DocumentPayload{
			TransactionSet: edi.TransactionSet214,
			ShipmentStatus: &edi.ShipmentStatusPayload{
				BOL:              details.shipmentRef,
				ProNumber:        details.referenceID,
				StatusCode:       details.statusCode,
				StatusReasonCode: details.reasonCode,
				EventDate:        details.eventAt,
				EventTime:        details.eventAt,
				References: map[string]string{
					"referenceId": details.referenceID,
					"shipmentId":  details.shipmentRef,
				},
			},
		}
	case edi.TransactionSet997, edi.TransactionSet999:
		entries := parseAcknowledgments(t)
		ack := edi.FunctionalAcknowledgmentPayload{}
		if len(entries) > 0 {
			first := entries[0]
			ack.OriginalFunctionalGroupID = first.originalFunctionalGroupID
			ack.OriginalGroupControlNumber = first.originalGroupControl
			ack.OriginalTransactionSet = transactionSetFromCode(first.originalTransactionSet)
			ack.OriginalTransactionControlNumber = first.originalControlNumber
			ack.GroupAcknowledgmentCode = first.groupAcknowledgmentCode
			ack.TransactionAcknowledgmentCode = first.acknowledgmentCode
			ack.AcceptedTransactionSetCount = first.acceptedCount
			ack.ReceivedTransactionSetCount = first.receivedCount
			ack.IncludedTransactionSetCount = first.includedCount
			ack.Diagnostics = first.diagnostics
		}
		if t.set == edi.TransactionSet999 {
			implementation := edi.ImplementationAckPayload(ack)
			return edi.DocumentPayload{
				TransactionSet:               edi.TransactionSet999,
				ImplementationAcknowledgment: &implementation,
			}
		}
		return edi.DocumentPayload{
			TransactionSet:           edi.TransactionSet997,
			FunctionalAcknowledgment: &ack,
		}
	}
	return edi.DocumentPayload{TransactionSet: t.set}
}

func parseAcknowledgments(t *parsedTransaction) []acknowledgmentEntry {
	ak1 := findSegment(t.segments, "AK1")
	ak9 := findSegment(t.segments, "AK9")
	base := acknowledgmentEntry{}
	if ak1 != nil {
		base.originalFunctionalGroupID = strings.TrimSpace(elementValue(ak1, 1))
		base.originalGroupControl = strings.TrimSpace(elementValue(ak1, 2))
	}
	if ak9 != nil {
		base.groupAcknowledgmentCode = strings.TrimSpace(elementValue(ak9, 1))
		base.includedCount = parseInt64(elementValue(ak9, 2))
		base.receivedCount = parseInt64(elementValue(ak9, 3))
		base.acceptedCount = parseInt64(elementValue(ak9, 4))
	}

	entries := make([]acknowledgmentEntry, 0, 2)
	var current *acknowledgmentEntry
	for index := range t.segments {
		segment := &t.segments[index]
		switch segment.SegmentID {
		case "AK2", "IK2":
			if current != nil {
				entries = append(entries, *current)
			}
			entry := base
			entry.originalTransactionSet = strings.TrimSpace(elementValue(segment, 1))
			entry.originalControlNumber = strings.TrimSpace(elementValue(segment, 2))
			current = &entry
		case "AK5", "IK5":
			if current != nil {
				current.acknowledgmentCode = strings.TrimSpace(elementValue(segment, 1))
			}
		case "AK3", "IK3":
			if current != nil {
				current.diagnostics = append(current.diagnostics, edi.AcknowledgmentDiagnostic{
					SegmentID:       strings.TrimSpace(elementValue(segment, 1)),
					SegmentPosition: parseInt64(elementValue(segment, 2)),
					ErrorCode:       strings.TrimSpace(elementValue(segment, 4)),
					Message:         "segment level acknowledgment error",
				})
			}
		case "AK4", "IK4":
			if current != nil {
				current.diagnostics = append(current.diagnostics, edi.AcknowledgmentDiagnostic{
					ElementPosition: parseInt64(elementValue(segment, 1)),
					ErrorCode:       strings.TrimSpace(elementValue(segment, 3)),
					Message:         "element level acknowledgment error",
				})
			}
		}
	}
	if current != nil {
		entries = append(entries, *current)
	}
	if len(entries) == 0 && (ak1 != nil || ak9 != nil) {
		entries = append(entries, base)
	}
	return entries
}

func parseTenderResponse(t *parsedTransaction) tenderResponseDetails {
	details := tenderResponseDetails{}
	if b1 := findSegment(t.segments, "B1"); b1 != nil {
		details.scac = strings.TrimSpace(elementValue(b1, 1))
		details.shipmentRef = strings.TrimSpace(elementValue(b1, 2))
		details.reservationCode = strings.ToUpper(strings.TrimSpace(elementValue(b1, 4)))
	}
	if details.shipmentRef == "" {
		details.shipmentRef = referenceFromL11(t.segments)
	}
	if k1 := findSegment(t.segments, "K1"); k1 != nil {
		details.remarks = strings.TrimSpace(strings.Join(stringutils.NonEmptyStrings(
			elementValue(k1, 1),
			elementValue(k1, 2),
		), " "))
	}
	return details
}

func parseShipmentStatus(t *parsedTransaction) shipmentStatusDetails {
	details := shipmentStatusDetails{}
	if b10 := findSegment(t.segments, "B10"); b10 != nil {
		details.referenceID = strings.TrimSpace(elementValue(b10, 1))
		details.shipmentRef = strings.TrimSpace(elementValue(b10, 2))
	}
	if details.shipmentRef == "" {
		details.shipmentRef = referenceFromL11(t.segments)
	}
	if at7 := findSegment(t.segments, "AT7"); at7 != nil {
		details.statusCode = strings.ToUpper(strings.TrimSpace(elementValue(at7, 1)))
		details.reasonCode = strings.ToUpper(strings.TrimSpace(elementValue(at7, 2)))
		details.eventAt = parseX12Timestamp(elementValue(at7, 5), elementValue(at7, 6))
	}
	return details
}

func parseFreightInvoice(t *parsedTransaction) edi.FreightInvoicePayload {
	payload := edi.FreightInvoicePayload{ReferenceNumbers: map[string]string{}}
	applyFreightInvoiceHeader(&payload, t.segments)
	applyFreightInvoiceDates(&payload, t.segments)
	applyFreightInvoiceReferences(&payload, t.segments)
	applyFreightInvoiceBillTo(&payload, t.segments)
	payload.LineCharges = parseFreightInvoiceCharges(t.segments)
	applyFreightInvoiceTotals(&payload, findSegment(t.segments, "L3"))
	return payload
}

func applyFreightInvoiceHeader(
	payload *edi.FreightInvoicePayload,
	segments []edix12inspect.X12Segment,
) {
	b3 := findSegment(segments, "B3")
	applyFreightInvoiceB3(payload, b3)
	if c3 := findSegment(segments, "C3"); c3 != nil {
		payload.CurrencyCode = strings.ToUpper(strings.TrimSpace(elementValue(c3, 1)))
	}
	if payload.CurrencyCode == "" && b3 != nil {
		payload.CurrencyCode = currencyCodeFromX12(elementValue(b3, 11))
	}
}

func applyFreightInvoiceB3(
	payload *edi.FreightInvoicePayload,
	b3 *edix12inspect.X12Segment,
) {
	if b3 == nil {
		return
	}
	payload.InvoiceNumber = strings.TrimSpace(elementValue(b3, 2))
	if shipmentRef := strings.TrimSpace(elementValue(b3, 3)); shipmentRef != "" {
		payload.ReferenceNumbers["shipmentId"] = shipmentRef
	}
	if paymentMethod := strings.TrimSpace(elementValue(b3, 4)); paymentMethod != "" {
		payload.ReferenceNumbers["paymentMethod"] = paymentMethod
	}
	if amount, err := decimalFromX12(elementValue(b3, 6)); err == nil {
		payload.TotalAmount = amount
	}
	if correction := strings.TrimSpace(elementValue(b3, 7)); correction != "" {
		payload.ReferenceNumbers["correctionIndicator"] = correction
	}
	payload.DeliveryDate = parseX12Timestamp(elementValue(b3, 8), "")
	if scac := strings.TrimSpace(elementValue(b3, 10)); scac != "" {
		payload.ReferenceNumbers["scac"] = scac
	}
}

func currencyCodeFromX12(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return ""
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return value
}

func applyFreightInvoiceDates(
	payload *edi.FreightInvoicePayload,
	segments []edix12inspect.X12Segment,
) {
	g62Segments := findSegmentsByID(segments, "G62")
	for index := range g62Segments {
		segment := &g62Segments[index]
		timestamp := parseX12Timestamp(elementValue(segment, 2), elementValue(segment, 4))
		if timestamp == 0 {
			continue
		}
		switch strings.TrimSpace(elementValue(segment, 1)) {
		case "17", "35":
			if payload.DeliveryDate == 0 {
				payload.DeliveryDate = timestamp
			}
		default:
			if payload.InvoiceDate == 0 {
				payload.InvoiceDate = timestamp
			}
		}
	}
}

func applyFreightInvoiceReferences(
	payload *edi.FreightInvoicePayload,
	segments []edix12inspect.X12Segment,
) {
	n9Segments := findSegmentsByID(segments, "N9")
	for index := range n9Segments {
		assignFreightInvoiceReference(
			payload,
			elementValue(&n9Segments[index], 1),
			elementValue(&n9Segments[index], 2),
		)
	}
	l11Segments := findSegmentsByID(segments, "L11")
	for index := range l11Segments {
		assignFreightInvoiceReference(
			payload,
			elementValue(&l11Segments[index], 2),
			elementValue(&l11Segments[index], 1),
		)
	}
}

func assignFreightInvoiceReference(
	payload *edi.FreightInvoicePayload,
	qualifier, value string,
) {
	qualifier = strings.ToUpper(strings.TrimSpace(qualifier))
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	switch qualifier {
	case "", "BM":
		if payload.BOL == "" {
			payload.BOL = value
		}
	case "CN", "P8", "PO#":
		if payload.ProNumber == "" {
			payload.ProNumber = value
		}
	}
	if qualifier != "" {
		if _, exists := payload.ReferenceNumbers[qualifier]; !exists {
			payload.ReferenceNumbers[qualifier] = value
		}
	}
}

func applyFreightInvoiceBillTo(
	payload *edi.FreightInvoicePayload,
	segments []edix12inspect.X12Segment,
) {
	inBillTo := false
	for index := range segments {
		segment := &segments[index]
		switch segment.SegmentID {
		case "N1":
			code := strings.ToUpper(strings.TrimSpace(elementValue(segment, 1)))
			inBillTo = code == "BT" || code == "BI"
			if inBillTo {
				payload.BillToName = strings.TrimSpace(elementValue(segment, 2))
			}
		case "N3":
			if inBillTo {
				payload.BillToAddressLine1 = strings.TrimSpace(elementValue(segment, 1))
				payload.BillToAddressLine2 = strings.TrimSpace(elementValue(segment, 2))
			}
		case "N4":
			if inBillTo {
				payload.BillToCity = strings.TrimSpace(elementValue(segment, 1))
				payload.BillToStateCode = strings.TrimSpace(elementValue(segment, 2))
				payload.BillToPostalCode = strings.TrimSpace(elementValue(segment, 3))
				payload.BillToCountry = strings.TrimSpace(elementValue(segment, 4))
			}
		}
	}
}

func parseFreightInvoiceCharges(
	segments []edix12inspect.X12Segment,
) []edi.FreightInvoiceCharge {
	ordered := make([]int64, 0, 4)
	bySequence := make(map[int64]*edi.FreightInvoiceCharge, 4)
	currentSequence := int64(0)
	resolve := func(segment *edix12inspect.X12Segment) *edi.FreightInvoiceCharge {
		if sequence := parseInt64(elementValue(segment, 1)); sequence > 0 {
			currentSequence = sequence
		} else if currentSequence == 0 {
			currentSequence = 1
		}
		charge, ok := bySequence[currentSequence]
		if !ok {
			charge = &edi.FreightInvoiceCharge{Sequence: currentSequence}
			bySequence[currentSequence] = charge
			ordered = append(ordered, currentSequence)
		}
		return charge
	}
	for index := range segments {
		segment := &segments[index]
		switch segment.SegmentID {
		case "LX":
			resolve(segment)
		case "L5":
			charge := resolve(segment)
			if description := strings.TrimSpace(elementValue(segment, 2)); description != "" {
				charge.Description = description
			}
		case "L0":
			charge := resolve(segment)
			if weight := parseInt64(elementValue(segment, 4)); weight > 0 {
				charge.Weight = &weight
			}
		case "L1":
			charge := resolve(segment)
			if rate, err := decimalFromX12(elementValue(segment, 2)); err == nil {
				charge.Rate = rate
			}
			if amount, err := decimalFromX12(elementValue(segment, 4)); err == nil {
				charge.Amount = amount.Decimal
			}
			if code := strings.ToUpper(strings.TrimSpace(elementValue(segment, 8))); code != "" {
				charge.Code = code
			}
		}
	}
	charges := make([]edi.FreightInvoiceCharge, 0, len(ordered))
	for _, sequence := range ordered {
		charges = append(charges, *bySequence[sequence])
	}
	return charges
}

func applyFreightInvoiceTotals(
	payload *edi.FreightInvoicePayload,
	l3 *edix12inspect.X12Segment,
) {
	if l3 == nil {
		return
	}
	if weight := parseInt64(elementValue(l3, 1)); weight > 0 {
		payload.Weight = &weight
	}
	if !payload.TotalAmount.Valid {
		if amount, err := decimalFromX12(elementValue(l3, 5)); err == nil {
			payload.TotalAmount = amount
		}
	}
}

func parseLoadTender(t *parsedTransaction) edi.LoadTenderPayload {
	payload := edi.LoadTenderPayload{
		PurposeCode:              edi.LoadTenderPurposeOriginal,
		CustomerID:               pulid.ID(inboundDefaultMappingKey),
		CustomerLabel:            "Default customer for inbound tenders",
		ServiceTypeID:            pulid.ID(inboundDefaultMappingKey),
		ServiceTypeLabel:         "Default service type for inbound tenders",
		FormulaTemplateID:        pulid.ID(inboundDefaultMappingKey),
		FormulaTemplateLabel:     "Default rating formula for inbound tenders",
		RatingDetail:             map[string]any{},
		RequiredMappingEntityIDs: map[edi.MappingEntityType][]pulid.ID{},
	}
	if b2 := findSegment(t.segments, "B2"); b2 != nil {
		if ref := strings.TrimSpace(elementValue(b2, 2)); ref != "" {
			payload.RatingDetail["externalShipmentId"] = ref
		}
		if paymentMethod := strings.TrimSpace(elementValue(b2, 4)); paymentMethod != "" {
			payload.RatingDetail["paymentMethod"] = paymentMethod
		}
	}
	if b2a := findSegment(t.segments, "B2A"); b2a != nil {
		if purpose := strings.TrimSpace(elementValue(b2a, 1)); purpose != "" {
			payload.PurposeCode = edi.LoadTenderPurposeCode(purpose)
		}
	}
	payload.BOL = referenceFromL11(t.segments)

	if at8 := findSegment(t.segments, "AT8"); at8 != nil {
		if weight := parseInt64(elementValue(at8, 3)); weight > 0 {
			payload.Weight = &weight
		}
		if pieces := parseInt64(elementValue(at8, 4)); pieces > 0 {
			payload.Pieces = &pieces
		}
	}
	applyTotalsFromL3(&payload, findSegment(t.segments, "L3"))

	stops := parseTenderStops(t.segments)
	if len(stops) > 0 {
		payload.Moves = []edi.LoadTenderMove{{Loaded: true, Sequence: 0, Stops: stops}}
	}
	for index := range stops {
		addRequiredMappingID(&payload, edi.MappingEntityTypeLocation, stops[index].LocationID)
	}

	l5Segments := findSegmentsByID(t.segments, "L5")
	for index := range l5Segments {
		description := strings.TrimSpace(elementValue(&l5Segments[index], 2))
		if description == "" {
			continue
		}
		commodityID := edi.MappingSourceID(description)
		payload.Commodities = append(payload.Commodities, edi.LoadTenderCommodity{
			CommodityID:          commodityID,
			CommodityLabel:       description,
			CommodityDescription: description,
		})
		addRequiredMappingID(&payload, edi.MappingEntityTypeCommodity, commodityID)
	}

	addRequiredMappingID(&payload, edi.MappingEntityTypeCustomer, payload.CustomerID)
	addRequiredMappingID(&payload, edi.MappingEntityTypeServiceType, payload.ServiceTypeID)
	addRequiredMappingID(
		&payload,
		edi.MappingEntityTypeFormulaTemplate,
		payload.FormulaTemplateID,
	)
	return payload
}

func applyTotalsFromL3(payload *edi.LoadTenderPayload, l3 *edix12inspect.X12Segment) {
	if l3 == nil {
		return
	}
	if payload.Weight == nil {
		if weight := parseInt64(elementValue(l3, 1)); weight > 0 {
			payload.Weight = &weight
		}
	}
	charge := strings.TrimSpace(elementValue(l3, 5))
	if charge == "" {
		return
	}
	if amount, err := decimalFromX12(charge); err == nil {
		payload.TotalChargeAmount = amount
	}
}

type stopSegmentGroup struct {
	s5   *edix12inspect.X12Segment
	n1   *edix12inspect.X12Segment
	n3   *edix12inspect.X12Segment
	n4   *edix12inspect.X12Segment
	g62s []*edix12inspect.X12Segment
}

func parseTenderStops(segments []edix12inspect.X12Segment) []edi.LoadTenderStop {
	groups := make([]*stopSegmentGroup, 0, 4)
	var current *stopSegmentGroup
	preN1 := make([]*edix12inspect.X12Segment, 0, 4)
	preN3 := make([]*edix12inspect.X12Segment, 0, 4)
	preN4 := make([]*edix12inspect.X12Segment, 0, 4)
	preG62 := make([]*edix12inspect.X12Segment, 0, 4)

	for index := range segments {
		segment := &segments[index]
		switch segment.SegmentID {
		case "S5":
			current = &stopSegmentGroup{s5: segment}
			groups = append(groups, current)
		case "N1":
			if current != nil {
				current.n1 = segment
			} else {
				preN1 = append(preN1, segment)
			}
		case "N3":
			if current != nil {
				current.n3 = segment
			} else {
				preN3 = append(preN3, segment)
			}
		case "N4":
			if current != nil {
				current.n4 = segment
			} else {
				preN4 = append(preN4, segment)
			}
		case "G62":
			if current != nil {
				current.g62s = append(current.g62s, segment)
			} else {
				preG62 = append(preG62, segment)
			}
		}
	}

	stops := make([]edi.LoadTenderStop, 0, len(groups))
	for index, group := range groups {
		backfillStopGroup(group, index, preN1, preN3, preN4, preG62)
		stops = append(stops, buildTenderStop(group, int64(index+1)))
	}
	return stops
}

func backfillStopGroup(
	group *stopSegmentGroup,
	index int,
	preN1, preN3, preN4, preG62 []*edix12inspect.X12Segment,
) {
	if group.n1 == nil && index < len(preN1) {
		group.n1 = preN1[index]
	}
	if group.n3 == nil && index < len(preN3) {
		group.n3 = preN3[index]
	}
	if group.n4 == nil && index < len(preN4) {
		group.n4 = preN4[index]
	}
	if len(group.g62s) == 0 && index < len(preG62) {
		group.g62s = append(group.g62s, preG62[index])
	}
}

func buildTenderStop(group *stopSegmentGroup, fallbackSequence int64) edi.LoadTenderStop {
	stop := edi.LoadTenderStop{
		ScheduleType: inboundStopScheduleType(),
		Sequence:     fallbackSequence,
	}
	if sequence := parseInt64(elementValue(group.s5, 1)); sequence > 0 {
		stop.Sequence = sequence
	}
	stop.Type = stopTypeFromCode(elementValue(group.s5, 2))
	if weight := parseInt64(elementValue(group.s5, 3)); weight > 0 {
		stop.Weight = &weight
	}
	if pieces := parseInt64(elementValue(group.s5, 5)); pieces > 0 {
		stop.Pieces = &pieces
	}
	locationCode := ""
	if group.n1 != nil {
		stop.LocationName = strings.TrimSpace(elementValue(group.n1, 2))
		locationCode = strings.TrimSpace(elementValue(group.n1, 4))
	}
	if group.n3 != nil {
		stop.LocationAddressLine1 = strings.TrimSpace(elementValue(group.n3, 1))
		stop.LocationAddressLine2 = strings.TrimSpace(elementValue(group.n3, 2))
	}
	if group.n4 != nil {
		stop.LocationCity = strings.TrimSpace(elementValue(group.n4, 1))
		stop.LocationStateCode = strings.TrimSpace(elementValue(group.n4, 2))
		stop.LocationPostalCode = strings.TrimSpace(elementValue(group.n4, 3))
	}
	stop.AddressLine = strings.Join(stringutils.NonEmptyStrings(
		stop.LocationAddressLine1,
		stop.LocationAddressLine2,
		strings.Join(stringutils.NonEmptyStrings(
			stop.LocationCity,
			stop.LocationStateCode,
			stop.LocationPostalCode,
		), ", "),
	), ", ")
	stop.LocationCode = locationCode
	stop.LocationID = edi.MappingSourceID(stringutils.FirstNonEmpty(
		locationCode,
		stop.LocationName,
		fmt.Sprintf("STOP-%d", stop.Sequence),
	))
	stop.LocationLabel = stringutils.FirstNonEmpty(
		strings.TrimSpace(strings.Join(
			stringutils.NonEmptyStrings(stop.LocationName, stop.AddressLine),
			" - ",
		)),
		string(stop.LocationID),
	)

	for _, g62 := range group.g62s {
		qualifier := strings.TrimSpace(elementValue(g62, 1))
		timestamp := parseX12Timestamp(elementValue(g62, 2), elementValue(g62, 4))
		if timestamp == 0 {
			continue
		}
		switch qualifier {
		case "38", "54":
			stop.ScheduledWindowEnd = &timestamp
		default:
			if stop.ScheduledWindowStart == 0 {
				stop.ScheduledWindowStart = timestamp
			}
		}
	}
	return stop
}

func inboundStopScheduleType() string {
	return "Open"
}

func stopTypeFromCode(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "LD", "PU", "CL", "PL", "PICKUP":
		return "Pickup"
	case "UL", "DO", "CU", "DELIVERY":
		return "Delivery"
	case "SPLITPICKUP":
		return "SplitPickup"
	case "SPLITDELIVERY":
		return "SplitDelivery"
	default:
		return "Pickup"
	}
}

func referenceFromL11(segments []edix12inspect.X12Segment) string {
	l11Segments := findSegmentsByID(segments, "L11")
	for index := range l11Segments {
		qualifier := strings.ToUpper(strings.TrimSpace(elementValue(&l11Segments[index], 2)))
		if qualifier != "" && qualifier != "BM" {
			continue
		}
		if value := strings.TrimSpace(elementValue(&l11Segments[index], 1)); value != "" {
			return value
		}
	}
	return ""
}

func acknowledgmentPayloadForTransaction(
	transaction *parsedTransaction,
	ackSet edi.TransactionSet,
) edi.DocumentPayload {
	ack := edi.FunctionalAcknowledgmentPayload{
		OriginalFunctionalGroupID: stringutils.FirstNonEmpty(
			transaction.functionalGroupID,
			edi.FunctionalGroupDefault(transaction.set),
		),
		OriginalGroupControlNumber:       transaction.groupControlNumber,
		OriginalTransactionSet:           transaction.set,
		OriginalTransactionControlNumber: transaction.controlNumber,
		GroupAcknowledgmentCode:          "A",
		TransactionAcknowledgmentCode:    "A",
		AcceptedTransactionSetCount:      1,
		ReceivedTransactionSetCount:      1,
		IncludedTransactionSetCount:      1,
	}
	if ackSet == edi.TransactionSet999 {
		implementation := edi.ImplementationAckPayload(ack)
		return edi.DocumentPayload{
			TransactionSet:               edi.TransactionSet999,
			ImplementationAcknowledgment: &implementation,
		}
	}
	return edi.DocumentPayload{
		TransactionSet:           edi.TransactionSet997,
		FunctionalAcknowledgment: &ack,
	}
}

func transactionSetFromCode(code string) edi.TransactionSet {
	code = strings.TrimSpace(code)
	switch code {
	case "SM":
		return edi.TransactionSet204
	case "IM":
		return edi.TransactionSet210
	case "QM":
		return edi.TransactionSet214
	case "GF":
		return edi.TransactionSet990
	default:
		return edi.TransactionSet(code)
	}
}

func addRequiredMappingID(
	payload *edi.LoadTenderPayload,
	entityType edi.MappingEntityType,
	id pulid.ID,
) {
	if id == "" {
		return
	}
	if slices.Contains(payload.RequiredMappingEntityIDs[entityType], id) {
		return
	}
	payload.RequiredMappingEntityIDs[entityType] = append(
		payload.RequiredMappingEntityIDs[entityType],
		id,
	)
}

func parseInt64(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func decimalFromX12(value string) (decimal.NullDecimal, error) {
	parsed, err := decimal.NewFromString(strings.TrimSpace(value))
	if err != nil {
		return decimal.NullDecimal{}, err
	}
	return decimal.NullDecimal{Decimal: parsed, Valid: true}, nil
}

func parseX12Timestamp(dateValue, timeValue string) int64 {
	dateValue = strings.TrimSpace(dateValue)
	timeValue = strings.TrimSpace(timeValue)
	if dateValue == "" {
		return 0
	}
	layout := "20060102"
	if len(dateValue) == 6 {
		layout = "060102"
	}
	if timeValue != "" {
		switch len(timeValue) {
		case 4:
			parsed, err := time.ParseInLocation(layout+"1504", dateValue+timeValue, time.UTC)
			if err == nil {
				return parsed.Unix()
			}
		case 6:
			parsed, err := time.ParseInLocation(layout+"150405", dateValue+timeValue, time.UTC)
			if err == nil {
				return parsed.Unix()
			}
		}
	}
	parsed, err := time.ParseInLocation(layout, dateValue, time.UTC)
	if err != nil {
		return 0
	}
	return parsed.Unix()
}
