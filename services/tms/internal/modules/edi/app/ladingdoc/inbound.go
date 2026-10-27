package ladingdoc

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	defaultMappingKey  = "DEFAULT"
	openScheduleType   = "Open"
	flatChargeMethod   = "Flat"
	externalShipmentID = "externalShipmentId"
)

// Unmapped indexes the codes a run found no mapping for by the document path that asked, so a
// reference the run left empty still carries the partner's code.
type Unmapped map[string]string

func UnmappedOf(items []lading.Unmapped) Unmapped {
	out := make(Unmapped, len(items))
	for _, item := range items {
		out[item.Path] = item.Code
	}
	return out
}

func decode(raw jsonutils.RawJSON, target any) error {
	if len(raw) == 0 {
		return ErrNoDocument
	}
	if err := sonic.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("ladingdoc: decode the document: %w", err)
	}
	return nil
}

func unix(value DateTime) int64 {
	instant, err := time.Parse(time.RFC3339, value.At)
	if err != nil {
		return 0
	}
	return instant.Unix()
}

func localDateTime(event *StatusEvent) edi.X12DateTime {
	if !event.Local {
		return edi.X12DateTime{}
	}
	instant, err := time.Parse(time.RFC3339, event.At.At)
	if err != nil {
		return edi.X12DateTime{}
	}
	if zone, zoneErr := time.LoadLocation(event.At.Zone); event.At.Zone != "" && zoneErr == nil {
		instant = instant.In(zone)
	}
	return edi.X12DateTime{
		Date:     instant.Format("20060102"),
		Time:     instant.Format("150405"),
		TimeCode: edi.X12TimeCodeLocal,
	}
}

func unixDay(value string) int64 {
	instant, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return 0
	}
	return instant.Unix()
}

func wholePounds(measure *Measure) *int64 {
	if measure == nil {
		return nil
	}
	amount, err := decimal.NewFromString(measure.Value)
	if err != nil || !amount.IsPositive() {
		return nil
	}
	value := amount.Round(0).IntPart()
	return &value
}

func wholeDegrees(measure *Measure) *int16 {
	if measure == nil {
		return nil
	}
	amount, err := decimal.NewFromString(measure.Value)
	if err != nil {
		return nil
	}
	value := amount.Round(0).IntPart()
	if value > math.MaxInt16 || value < math.MinInt16 {
		return nil
	}
	degrees := int16(value)
	return &degrees
}

func amountOf(value *Money) decimal.NullDecimal {
	if value == nil {
		return decimal.NullDecimal{}
	}
	amount, err := decimal.NewFromString(value.Amount)
	if err != nil {
		return decimal.NullDecimal{}
	}
	return decimal.NewNullDecimal(amount)
}

type mapped struct {
	payload *edi.LoadTenderPayload
}

func (m mapped) require(entity edi.MappingEntityType, id pulid.ID) {
	if id.IsNil() || slices.Contains(m.payload.RequiredMappingEntityIDs[entity], id) {
		return
	}
	m.payload.RequiredMappingEntityIDs[entity] = append(
		m.payload.RequiredMappingEntityIDs[entity],
		id,
	)
}

// source is the mapping source of a reference: the partner's code, from the reference or from
// the unmapped code its path asked for.
func source(
	reference *Ref,
	unmapped Unmapped,
	path string,
	fallbacks ...string,
) (id pulid.ID, display string) {
	code := ""
	label := ""
	if reference != nil {
		code = reference.Code
		label = reference.Label
	}
	if code == "" {
		code = unmapped[path]
	}
	if code == "" {
		code = stringutils.FirstNonEmpty(fallbacks...)
	}
	if code == "" {
		return pulid.Nil, label
	}
	return edi.MappingSourceID(code), stringutils.FirstNonEmpty(label, code)
}

func purposeCode(purpose string) edi.LoadTenderPurposeCode {
	switch purpose {
	case PurposeChange:
		return edi.LoadTenderPurposeChange
	case PurposeCancel:
		return edi.LoadTenderPurposeCancel
	default:
		return edi.LoadTenderPurposeOriginal
	}
}

var paymentCodes = map[string]string{
	"Prepaid":    "PP",
	"Collect":    "CC",
	"ThirdParty": "TP",
}

// ToLoadTender is the tender payload of a loadTender a map read. Every reference carries the
// partner's code as its mapping source; the transfer resolves them through the partner's
// mappings when it is approved.
//
//nolint:funlen // One pass over the document keeps each field's mapping beside its source.
func ToLoadTender(
	raw jsonutils.RawJSON,
	unmapped Unmapped,
) (*edi.LoadTenderPayload, string, error) {
	var document LoadTender
	if err := decode(raw, &document); err != nil {
		return nil, "", err
	}
	payload := &edi.LoadTenderPayload{
		PurposeCode:              purposeCode(document.Purpose),
		BOL:                      document.BOL,
		Weight:                   wholePounds(document.Weight),
		Pieces:                   document.Pieces,
		TotalChargeAmount:        amountOf(document.Total),
		FreightChargeAmount:      amountOf(document.FreightCharge),
		OtherChargeAmount:        amountOf(document.OtherCharges),
		RatingDetail:             map[string]any{externalShipmentID: document.Key},
		RequiredMappingEntityIDs: map[edi.MappingEntityType][]pulid.ID{},
		CustomFields:             document.Custom,
	}
	if payment := paymentCodes[document.Payment]; payment != "" {
		payload.RatingDetail["paymentMethod"] = payment
	}
	if document.PONumber != "" {
		payload.RatingDetail["poNumber"] = document.PONumber
	}
	if document.ProNumber != "" {
		payload.RatingDetail["proNumber"] = document.ProNumber
	}
	if document.Temperature != nil {
		payload.TemperatureMin = wholeDegrees(document.Temperature.Min)
		payload.TemperatureMax = wholeDegrees(document.Temperature.Max)
	}
	need := mapped{payload: payload}
	payload.CustomerID, payload.CustomerLabel = source(
		document.Customer,
		unmapped,
		"customer",
		defaultMappingKey,
	)
	payload.ServiceTypeID, payload.ServiceTypeLabel = source(
		document.ServiceType,
		unmapped,
		"serviceType",
		defaultMappingKey,
	)
	payload.FormulaTemplateID, payload.FormulaTemplateLabel = source(
		document.Rating,
		unmapped,
		"rating",
		defaultMappingKey,
	)
	payload.ShipmentTypeID, payload.ShipmentTypeLabel = source(
		document.ShipmentType,
		unmapped,
		"shipmentType",
	)
	need.require(edi.MappingEntityTypeCustomer, payload.CustomerID)
	need.require(edi.MappingEntityTypeServiceType, payload.ServiceTypeID)
	need.require(edi.MappingEntityTypeFormulaTemplate, payload.FormulaTemplateID)
	need.require(edi.MappingEntityTypeShipmentType, payload.ShipmentTypeID)

	stops := make([]edi.LoadTenderStop, 0, len(document.Stops))
	for index := range document.Stops {
		stop := &document.Stops[index]
		path := fmt.Sprintf("stops[%d].location", index)
		written := edi.LoadTenderStop{
			Type:                 stop.Type,
			ScheduleType:         openScheduleType,
			Sequence:             stop.Sequence,
			LocationName:         stop.Name,
			LocationCode:         stop.Code,
			Pieces:               stop.Pieces,
			Weight:               wholePounds(stop.Weight),
			ScheduledWindowStart: unix(stop.Window.Start),
		}
		if written.Sequence == 0 {
			written.Sequence = int64(index + 1)
		}
		if stop.Window.End != nil {
			end := unix(*stop.Window.End)
			if end > 0 {
				written.ScheduledWindowEnd = &end
			}
		}
		if stop.Address != nil {
			written.LocationAddressLine1 = stop.Address.Line1
			written.LocationAddressLine2 = stop.Address.Line2
			written.LocationCity = stop.Address.City
			written.LocationStateCode = stop.Address.State
			written.LocationPostalCode = stop.Address.Postal
		}
		written.AddressLine = strings.Join(stringutils.FilterEmpty([]string{
			written.LocationAddressLine1,
			written.LocationAddressLine2,
			strings.Join(stringutils.FilterEmpty([]string{
				written.LocationCity,
				written.LocationStateCode,
				written.LocationPostalCode,
			}), ", "),
		}), ", ")
		var label string
		written.LocationID, label = source(
			stop.Location,
			unmapped,
			path,
			stop.Code,
			stop.Name,
			fmt.Sprintf("STOP-%d", written.Sequence),
		)
		written.LocationCode = stringutils.FirstNonEmpty(written.LocationCode, unmapped[path])
		written.LocationLabel = stringutils.FirstNonEmpty(
			strings.TrimSpace(strings.Join(stringutils.FilterEmpty([]string{
				written.LocationName,
				written.AddressLine,
			}), " - ")),
			label,
		)
		need.require(edi.MappingEntityTypeLocation, written.LocationID)
		stops = append(stops, written)
	}
	if len(stops) > 0 {
		payload.Moves = []edi.LoadTenderMove{{Loaded: true, Stops: stops}}
	}
	for index := range document.Commodities {
		line := &document.Commodities[index]
		id, label := source(
			line.Commodity,
			unmapped,
			fmt.Sprintf("commodities[%d].commodity", index),
			line.Description,
		)
		commodity := edi.LoadTenderCommodity{
			CommodityID:          id,
			CommodityLabel:       stringutils.FirstNonEmpty(label, line.Description),
			CommodityDescription: line.Description,
		}
		if weight := wholePounds(line.Weight); weight != nil {
			commodity.Weight = *weight
		}
		if line.Pieces != nil {
			commodity.Pieces = *line.Pieces
		}
		need.require(edi.MappingEntityTypeCommodity, id)
		payload.Commodities = append(payload.Commodities, commodity)
	}
	for index := range document.Charges {
		charge := &document.Charges[index]
		id, label := source(
			charge.Accessorial,
			unmapped,
			fmt.Sprintf("charges[%d].accessorial", index),
			charge.Code,
		)
		amount := amountOf(&charge.Amount)
		payload.AdditionalCharges = append(payload.AdditionalCharges, edi.LoadTenderCharge{
			AccessorialChargeID:    id,
			AccessorialLabel:       label,
			AccessorialCode:        charge.Code,
			AccessorialDescription: charge.Description,
			Method:                 flatChargeMethod,
			Amount:                 amount.Decimal,
			Unit:                   1,
		})
		need.require(edi.MappingEntityTypeAccessorialCharge, id)
	}
	return payload, document.Key, nil
}

// TenderAnswer is what a partner's tender response says.
type TenderAnswer struct {
	Key             string
	Accepted        bool
	ReservationCode string
	Reason          string
	SCAC            string
	References      []string
}

func ToTenderAnswer(raw jsonutils.RawJSON) (*TenderAnswer, error) {
	var document TenderResponse
	if err := decode(raw, &document); err != nil {
		return nil, err
	}
	answer := &TenderAnswer{
		Key:      document.Key,
		Accepted: document.Decision == DecisionAccept,
		Reason:   document.Reason,
		SCAC:     document.SCAC,
	}
	answer.ReservationCode = "D"
	if answer.Accepted {
		answer.ReservationCode = "A"
	}
	answer.References = append(answer.References, sortedValues(document.Refs)...)
	return answer, nil
}

func sortedValues(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if values[key] != "" {
			out = append(out, values[key])
		}
	}
	return out
}

// ToShipmentStatus is the status payload of a shipmentStatus a map read.
func ToShipmentStatus(raw jsonutils.RawJSON) (*edi.ShipmentStatusPayload, error) {
	var document ShipmentStatus
	if err := decode(raw, &document); err != nil {
		return nil, err
	}
	payload := &edi.ShipmentStatusPayload{
		BOL:              document.BOL,
		ProNumber:        document.ProNumber,
		ShipperReference: stringutils.FirstNonEmpty(document.ShipperReference, document.Key),
		Weight:           wholePounds(document.Weight),
		Pieces:           document.Pieces,
		EquipmentNumber:  document.Equipment,
		References:       map[string]string{"shipmentId": document.Key},
	}
	for key, value := range document.Refs {
		payload.References[key] = value
	}
	for index := range document.Events {
		event := &document.Events[index]
		written := edi.ShipmentStatusEvent{
			Index:           index,
			StatusCode:      StatusCodeOf(event.Kind),
			ReasonCode:      event.Reason,
			EventAt:         unix(event.At),
			DateTime:        localDateTime(event),
			StopSequence:    event.Stop,
			City:            event.City,
			StateCode:       event.State,
			CountryCode:     event.Country,
			EquipmentNumber: event.Equipment,
		}
		if latitude, err := strconv.ParseFloat(event.Latitude, 64); err == nil {
			written.Latitude = &latitude
		}
		if longitude, err := strconv.ParseFloat(event.Longitude, 64); err == nil {
			written.Longitude = &longitude
		}
		payload.Events = append(payload.Events, written)
	}
	if len(payload.Events) > 0 {
		first := payload.Events[0]
		payload.StatusCode = first.StatusCode
		payload.StatusReasonCode = first.ReasonCode
		payload.EventDate = first.EventAt
		payload.EventTime = first.EventAt
		payload.City = first.City
		payload.StateCode = first.StateCode
		payload.CountryCode = first.CountryCode
	}
	return payload, nil
}

// ToFreightInvoice is the invoice payload of a freightInvoice a map read.
func ToFreightInvoice(raw jsonutils.RawJSON) (*edi.FreightInvoicePayload, error) {
	var document FreightInvoice
	if err := decode(raw, &document); err != nil {
		return nil, err
	}
	payload := &edi.FreightInvoicePayload{
		InvoiceNumber:    document.Number,
		InvoiceDate:      unixDay(document.Date),
		DeliveryDate:     unixDay(document.DeliveryDate),
		BOL:              document.BOL,
		ProNumber:        document.ProNumber,
		Weight:           wholePounds(document.Weight),
		CurrencyCode:     stringutils.FirstNonEmpty(document.Total.Currency, DefaultCurency),
		TotalAmount:      amountOf(&document.Total),
		ReferenceNumbers: map[string]string{},
	}
	for key, value := range document.Refs {
		payload.ReferenceNumbers[key] = value
	}
	if document.Key != "" {
		payload.ReferenceNumbers["shipmentId"] = document.Key
	}
	if document.BillTo != nil {
		payload.BillToName = document.BillTo.Name
		if address := document.BillTo.Address; address != nil {
			payload.BillToAddressLine1 = address.Line1
			payload.BillToAddressLine2 = address.Line2
			payload.BillToCity = address.City
			payload.BillToStateCode = address.State
			payload.BillToPostalCode = address.Postal
			payload.BillToCountry = address.Country
		}
	}
	for index := range document.Lines {
		line := &document.Lines[index]
		amount := amountOf(&line.Amount)
		charge := edi.FreightInvoiceCharge{
			Sequence:    line.Sequence,
			Code:        stringutils.FirstNonEmpty(line.Code, refCode(line.Accessorial)),
			Description: line.Description,
			Amount:      amount.Decimal,
			Weight:      wholePounds(line.Weight),
		}
		if rate, err := decimal.NewFromString(line.Rate); err == nil {
			charge.Rate = decimal.NewNullDecimal(rate)
		}
		payload.LineCharges = append(payload.LineCharges, charge)
	}
	for index := range document.Shipments {
		shipment := &document.Shipments[index]
		payload.Shipments = append(payload.Shipments, edi.FreightInvoiceShipment{
			ProNumber:    shipment.ProNumber,
			BOL:          shipment.BOL,
			PONumber:     shipment.PONumber,
			DeliveryDate: unixDay(shipment.DeliveryDate),
			Amount:       amountOf(shipment.Amount),
		})
	}
	return payload, nil
}

func refCode(reference *Ref) string {
	if reference == nil {
		return ""
	}
	return reference.Code
}
