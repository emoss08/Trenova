package ediinboundservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

var (
	errEmptyFile     = errors.New("inbound file is empty")
	errNoSegments    = errors.New("inbound file does not contain any X12 segments")
	errNoInterchange = errors.New(
		"inbound file does not start with an ISA interchange header",
	)
	errNoTransactions             = errors.New("inbound file does not contain any X12 transactions")
	errTransactionWithoutSegments = errors.New("transaction does not contain any segments")
	errRejectedTransaction        = errors.New("rejected, so it was not processed")
)

// parseInterchange reads a file through the EDI module against the partner's guides.
// A returned problem is the file's fault and quarantines it; an error is the
// module's and is retried.
func parseInterchange(
	ctx context.Context,
	rawX12 string,
) (interchange *parsedInterchange, problem, err error) {
	if strings.TrimSpace(rawX12) == "" {
		return nil, errEmptyFile, nil
	}
	inspection, err := edix12inspect.InspectX12(
		ctx,
		&edix12inspect.InspectX12Request{RawX12: rawX12},
	)
	if err != nil {
		return nil, nil, err
	}
	reading := inspection.Reading
	if len(inspection.Segments) == 0 {
		return nil, errNoSegments, nil
	}
	if len(reading.Interchanges) == 0 || reading.Interchanges[0].Header == nil {
		return nil, errNoInterchange, nil
	}
	if reading.Summary.Transactions == 0 {
		return nil, errNoTransactions, nil
	}

	first := &reading.Interchanges[0]
	interchange = &parsedInterchange{
		inspection:        inspection,
		reading:           reading,
		controlNumber:     first.Header.ControlNumber,
		senderQualifier:   first.Header.SenderQualifier,
		senderID:          first.Header.SenderID,
		receiverQualifier: first.Header.ReceiverQualifier,
		receiverID:        first.Header.ReceiverID,
		version:           first.Header.Version,
		usage:             first.Header.Usage,
		transactions:      make([]parsedTransaction, 0, reading.Summary.Transactions),
	}
	for interchangeIndex := range reading.Interchanges {
		for groupIndex := range reading.Interchanges[interchangeIndex].Groups {
			group := &reading.Interchanges[interchangeIndex].Groups[groupIndex]
			parsed := parsedGroup{group: group}
			if group.Header != nil {
				parsed.functionalGroupID = group.Header.FunctionalID
				parsed.controlNumber = group.Header.ControlNumber
				parsed.applicationSender = group.Header.Sender
				parsed.applicationReceiver = group.Header.Receiver
				parsed.version = group.Header.Version
			}
			interchange.groups = append(interchange.groups, parsed)
			for transactionIndex := range group.Transactions {
				transaction, buildErr := buildTransaction(
					inspection,
					&group.Transactions[transactionIndex],
					len(interchange.groups)-1,
					&parsed,
				)
				if buildErr != nil {
					return nil, buildErr, nil
				}
				interchange.transactions = append(interchange.transactions, *transaction)
			}
		}
	}
	if len(interchange.groups) > 0 {
		lead := &interchange.groups[0]
		interchange.functionalGroupID = lead.functionalGroupID
		interchange.groupControlNumber = lead.controlNumber
		interchange.applicationSender = lead.applicationSender
		interchange.applicationReceiver = lead.applicationReceiver
	}
	return interchange, nil, nil
}

func buildTransaction(
	inspection *edix12inspect.InspectX12Result,
	transaction *x12.Transaction,
	group int,
	parsed *parsedGroup,
) (*parsedTransaction, error) {
	if transaction.FirstSegment > transaction.LastSegment ||
		transaction.LastSegment >= len(inspection.Segments) {
		return nil, fmt.Errorf(
			"%w: transaction %d",
			errTransactionWithoutSegments,
			transaction.Index+1,
		)
	}
	segments := inspection.Segments[transaction.FirstSegment : transaction.LastSegment+1]
	raw := strings.Builder{}
	for index := range segments {
		raw.WriteString(segments[index].RawWithTerminator)
	}
	return &parsedTransaction{
		set:                edi.TransactionSet(transaction.TransactionSet),
		controlNumber:      transaction.ControlNumber,
		groupControlNumber: parsed.controlNumber,
		functionalGroupID:  parsed.functionalGroupID,
		segments:           segments,
		raw:                raw.String(),
		version:            stringutils.FirstNonEmpty(transaction.Convention, parsed.version),
		group:              group,
		status:             transaction.Status,
		findings:           transaction.Findings,
	}, nil
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
				EventTimeCode:    firstEventTimeCode(details.events),
				Events:           details.events,
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
	details.references = referenceValues(t.segments)
	if k1 := findSegment(t.segments, "K1"); k1 != nil {
		details.remarks = strings.TrimSpace(strings.Join(stringutils.NonEmptyStrings(
			elementValue(k1, 1),
			elementValue(k1, 2),
		), " "))
	}
	return details
}

func parseShipmentStatus(t *parsedTransaction) shipmentStatusDetails {
	details := shipmentStatusDetails{controlNumber: t.controlNumber}
	if b10 := findSegment(t.segments, "B10"); b10 != nil {
		details.referenceID = strings.TrimSpace(elementValue(b10, 1))
		details.shipmentRef = strings.TrimSpace(elementValue(b10, 2))
	}
	if details.shipmentRef == "" {
		details.shipmentRef = referenceFromL11(t.segments)
	}
	details.references = referenceValues(t.segments)
	details.events = parseShipmentStatusEvents(t.segments)
	if len(details.events) > 0 {
		first := details.events[0]
		details.statusCode = first.StatusCode
		details.reasonCode = first.ReasonCode
		details.eventAt = first.EventAt
	}
	return details
}

type statusLoop struct {
	stopSequence *int64
	city         string
	stateCode    string
	countryCode  string
	events       []int
}

func parseShipmentStatusEvents(segments []edix12inspect.X12Segment) []edi.ShipmentStatusEvent {
	events := make([]edi.ShipmentStatusEvent, 0, 1)
	loops := make([]*statusLoop, 0, 1)
	var loop *statusLoop
	var current *edi.ShipmentStatusEvent
	openLoop := func() {
		loop = &statusLoop{}
		loops = append(loops, loop)
		current = nil
	}

	for index := range segments {
		segment := &segments[index]
		switch segment.SegmentID {
		case "LX":
			openLoop()
		case "AT7":
			if loop == nil {
				openLoop()
			}
			event, ok := shipmentStatusEventFromAT7(segment, len(events))
			if !ok {
				current = nil
				continue
			}
			events = append(events, event)
			loop.events = append(loop.events, len(events)-1)
			current = &events[len(events)-1]
		case "MS1":
			applyStatusPlace(segment, current, loop)
		case "MS2":
			applyStatusEquipment(segment, current)
		case "L11":
			applyStatusStopReference(segment, loop)
		case "N4":
			applyStatusPartyPlace(segment, loop)
		}
	}

	for _, statusLoop := range loops {
		for _, eventIndex := range statusLoop.events {
			inheritLoopDetails(&events[eventIndex], statusLoop)
		}
	}
	return events
}

func shipmentStatusEventFromAT7(
	segment *edix12inspect.X12Segment,
	index int,
) (edi.ShipmentStatusEvent, bool) {
	event := edi.ShipmentStatusEvent{
		Index:                 index,
		StatusCode:            upperElement(segment, 1),
		ReasonCode:            upperElement(segment, 2),
		AppointmentCode:       upperElement(segment, 3),
		AppointmentReasonCode: upperElement(segment, 4),
		DateTime: edi.X12DateTime{
			Date:     strings.TrimSpace(elementValue(segment, 5)),
			Time:     strings.TrimSpace(elementValue(segment, 6)),
			TimeCode: edi.NormalizeX12TimeCode(elementValue(segment, 7)),
		},
	}
	if event.StatusCode == "" && event.AppointmentCode == "" {
		return edi.ShipmentStatusEvent{}, false
	}
	event.EventAt, _ = event.DateTime.Unix(time.UTC)
	return event, true
}

func applyStatusPlace(
	segment *edix12inspect.X12Segment,
	current *edi.ShipmentStatusEvent,
	loop *statusLoop,
) {
	city := strings.TrimSpace(elementValue(segment, 1))
	state := upperElement(segment, 2)
	country := upperElement(segment, 3)
	if current == nil {
		if loop != nil && city != "" && state != "" {
			loop.city, loop.stateCode, loop.countryCode = city, state, country
		}
		return
	}
	if city != "" && state != "" {
		current.City, current.StateCode, current.CountryCode = city, state, country
	}
	latitude := edi.ParseX12Latitude(elementValue(segment, 5), elementValue(segment, 7))
	longitude := edi.ParseX12Longitude(elementValue(segment, 4), elementValue(segment, 6))
	if latitude != nil && longitude != nil {
		current.Latitude, current.Longitude = latitude, longitude
	}
}

func applyStatusEquipment(segment *edix12inspect.X12Segment, current *edi.ShipmentStatusEvent) {
	if current == nil {
		return
	}
	current.EquipmentOwner = upperElement(segment, 1)
	current.EquipmentNumber = strings.TrimSpace(elementValue(segment, 2))
	current.EquipmentType = upperElement(segment, 3)
}

func applyStatusStopReference(segment *edix12inspect.X12Segment, loop *statusLoop) {
	if loop == nil || upperElement(segment, 2) != stopSequenceQualifier {
		return
	}
	sequence, err := strconv.ParseInt(strings.TrimSpace(elementValue(segment, 1)), 10, 64)
	if err != nil || sequence < 0 {
		return
	}
	loop.stopSequence = &sequence
}

func applyStatusPartyPlace(segment *edix12inspect.X12Segment, loop *statusLoop) {
	if loop == nil || loop.city != "" {
		return
	}
	city := strings.TrimSpace(elementValue(segment, 1))
	state := upperElement(segment, 2)
	if city == "" || state == "" {
		return
	}
	loop.city, loop.stateCode, loop.countryCode = city, state, upperElement(segment, 4)
}

func inheritLoopDetails(event *edi.ShipmentStatusEvent, loop *statusLoop) {
	if event.StopSequence == nil && loop.stopSequence != nil {
		sequence := *loop.stopSequence
		event.StopSequence = &sequence
	}
	if !event.HasPlace() && loop.city != "" {
		event.City, event.StateCode, event.CountryCode = loop.city, loop.stateCode, loop.countryCode
	}
}

func upperElement(segment *edix12inspect.X12Segment, position int) string {
	return strings.ToUpper(strings.TrimSpace(elementValue(segment, position)))
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
	applyFreightInvoiceB3(payload, findSegment(segments, "B3"))
	if c3 := findSegment(segments, "C3"); c3 != nil {
		payload.CurrencyCode = currencyCodeFromX12(elementValue(c3, 1))
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
	payload.InvoiceDate = parseX12Timestamp(elementValue(b3, 6), "")
	if amount, ok := numberElement(b3, 7); ok {
		payload.TotalAmount = amount
	}
	if correction := strings.TrimSpace(elementValue(b3, 8)); correction != "" {
		payload.ReferenceNumbers["correctionIndicator"] = correction
	}
	payload.DeliveryDate = parseX12Timestamp(elementValue(b3, 9), "")
	if scac := strings.TrimSpace(elementValue(b3, 11)); scac != "" {
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
			if rate, ok := numberElement(segment, 2); ok {
				charge.Rate = rate
			}
			if amount, ok := numberElement(segment, 4); ok {
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
		if amount, ok := numberElement(l3, 5); ok {
			payload.TotalAmount = amount
		}
	}
}

var shipmentPaymentMethods = map[string]struct{}{
	"BP": {}, "CA": {}, "CC": {}, "CD": {}, "CF": {}, "DE": {}, "DF": {}, "FO": {},
	"HP": {}, "MX": {}, "NC": {}, "NR": {}, "PA": {}, "PB": {}, "PC": {}, "PD": {},
	"PE": {}, "PL": {}, "PO": {}, "PP": {}, "PS": {}, "PU": {}, "RC": {}, "RF": {},
	"RS": {}, "SC": {}, "TP": {}, "WC": {},
}

func isShipmentPaymentMethod(value string) bool {
	_, ok := shipmentPaymentMethods[value]
	return ok
}

type TenderReference struct {
	ShipmentID    string
	PaymentMethod string
}

func TenderReferences(
	ctx context.Context,
	rawTransaction string,
	envelope *edi.X12EnvelopeSettings,
) (TenderReference, bool, error) {
	if strings.TrimSpace(rawTransaction) == "" {
		return TenderReference{}, false, nil
	}
	inspection, err := edix12inspect.InspectX12(ctx, &edix12inspect.InspectX12Request{
		RawX12:   rawTransaction,
		Envelope: envelope,
	})
	if err != nil {
		return TenderReference{}, false, err
	}
	b2 := findSegment(inspection.Segments, "B2")
	if b2 == nil {
		return TenderReference{}, false, nil
	}
	shipmentID, paymentMethod := readB2(b2)
	return TenderReference{ShipmentID: shipmentID, PaymentMethod: paymentMethod}, true, nil
}

const trenovaShipmentIDPrefix = "shp_"

func readB2(b2 *edix12inspect.X12Segment) (shipmentID, paymentMethod string) {
	element := func(position int) string {
		return strings.TrimSpace(elementValue(b2, position))
	}
	b202, b204, b206 := element(2), element(4), element(6)
	legacy := b206 == "" &&
		(isShipmentPaymentMethod(b204) ||
			(b204 == "" && strings.HasPrefix(b202, trenovaShipmentIDPrefix)))
	if legacy {
		return b202, b204
	}
	return b204, b206
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
		shipmentID, paymentMethod := readB2(b2)
		if shipmentID != "" {
			payload.RatingDetail["externalShipmentId"] = shipmentID
		}
		if paymentMethod != "" {
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
	if amount, ok := numberElement(l3, 5); ok {
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

// referenceValues collects every L11 (value, qualifier) and N9 (qualifier,
// value) reference on the transaction, regardless of qualifier.
func referenceValues(segments []edix12inspect.X12Segment) []string {
	references := make([]string, 0, 4)
	l11Segments := findSegmentsByID(segments, "L11")
	for index := range l11Segments {
		if upperElement(&l11Segments[index], 2) == stopSequenceQualifier {
			continue
		}
		if value := strings.TrimSpace(elementValue(&l11Segments[index], 1)); value != "" {
			references = append(references, value)
		}
	}
	n9Segments := findSegmentsByID(segments, "N9")
	for index := range n9Segments {
		if value := strings.TrimSpace(elementValue(&n9Segments[index], 2)); value != "" {
			references = append(references, value)
		}
	}
	return references
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

func numberElement(segment *edix12inspect.X12Segment, position int) (decimal.NullDecimal, bool) {
	if segment == nil {
		return decimal.NullDecimal{}, false
	}
	for index := range segment.Elements {
		element := &segment.Elements[index]
		if element.Position != position {
			continue
		}
		parsed, err := x12.ParseNumber(element.Value, element.DataType)
		if err != nil {
			return decimal.NullDecimal{}, false
		}
		return decimal.NullDecimal{Decimal: parsed, Valid: true}, true
	}
	return decimal.NullDecimal{}, false
}

func parseX12Timestamp(dateValue, timeValue string) int64 {
	at, _ := edi.X12DateTime{Date: dateValue, Time: timeValue}.Unix(time.UTC)
	return at
}

func firstEventTimeCode(events []edi.ShipmentStatusEvent) string {
	if len(events) == 0 {
		return ""
	}
	return events[0].DateTime.TimeCode
}
