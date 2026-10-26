package ladingdoc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

var (
	ErrNoKey   = errors.New("ladingdoc: the document has no key the partner knows the load by")
	ErrNoStops = errors.New(
		"ladingdoc: a load tender needs at least one stop with a scheduled time",
	)
	ErrNoEvents = errors.New("ladingdoc: a shipment status needs at least one event with a time")
	ErrNoNumber = errors.New("ladingdoc: a freight invoice needs a number and a date")
)

// Zones are the time zones of a document's places, by location; a time at a place without one
// is written in the organisation's zone.
type Zones struct {
	Default   string
	Locations map[pulid.ID]string
}

func (z Zones) of(locationID pulid.ID) string {
	if zone := z.Locations[locationID]; zone != "" {
		return zone
	}
	return z.Default
}

func at(unix int64, zone string) DateTime {
	instant := time.Unix(unix, 0).UTC()
	if location, err := time.LoadLocation(zone); zone != "" && err == nil {
		return DateTime{At: instant.In(location).Format(time.RFC3339), Zone: zone}
	}
	return DateTime{At: instant.Format(time.RFC3339)}
}

func day(unix int64, zone string) string {
	if unix == 0 {
		return ""
	}
	instant := time.Unix(unix, 0).UTC()
	if location, err := time.LoadLocation(zone); zone != "" && err == nil {
		instant = instant.In(location)
	}
	return instant.Format(time.DateOnly)
}

func pounds(weight *int64) *Measure {
	if weight == nil || *weight <= 0 {
		return nil
	}
	return &Measure{Value: strconv.FormatInt(*weight, 10), Unit: UnitPounds}
}

func fahrenheit(degrees *int16) *Measure {
	if degrees == nil {
		return nil
	}
	return &Measure{Value: strconv.Itoa(int(*degrees)), Unit: UnitFahrenheit}
}

func money(amount decimal.NullDecimal, currency string) *Money {
	if !amount.Valid {
		return nil
	}
	value := moneyOf(amount.Decimal, currency)
	return &value
}

func moneyOf(amount decimal.Decimal, currency string) Money {
	return Money{
		Amount:   amount.StringFixed(2),
		Currency: stringutils.FirstNonEmpty(currency, DefaultCurency),
	}
}

func ref(id pulid.ID, code, label string) *Ref {
	if id.IsNil() && code == "" && label == "" {
		return nil
	}
	return &Ref{ID: id.String(), Code: code, Label: label}
}

func refs(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		if strings.TrimSpace(value) != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func encode(document any) (jsonutils.RawJSON, error) {
	encoded, err := sonic.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("ladingdoc: encode the document: %w", err)
	}
	return encoded, nil
}

func purposeOf(code edi.LoadTenderPurposeCode) string {
	switch code {
	case edi.LoadTenderPurposeChange:
		return PurposeChange
	case edi.LoadTenderPurposeCancel:
		return PurposeCancel
	default:
		return PurposeOriginal
	}
}

func stopTypeOf(stopType string) string {
	switch strings.ToLower(stopType) {
	case "delivery", "splitdelivery":
		return StopDelivery
	default:
		return StopPickup
	}
}

// FromLoadTender is the loadTender document of a tender Trenova offers.
func FromLoadTender(
	payload *edi.LoadTenderPayload,
	key string,
	zones Zones,
) (jsonutils.RawJSON, error) {
	if strings.TrimSpace(key) == "" {
		return nil, ErrNoKey
	}
	document := LoadTender{
		Key:           key,
		Purpose:       purposeOf(payload.PurposeCode),
		BOL:           payload.BOL,
		Customer:      ref(payload.CustomerID, "", payload.CustomerLabel),
		ServiceType:   ref(payload.ServiceTypeID, "", payload.ServiceTypeLabel),
		ShipmentType:  ref(payload.ShipmentTypeID, "", payload.ShipmentTypeLabel),
		Rating:        ref(payload.FormulaTemplateID, "", payload.FormulaTemplateLabel),
		Weight:        pounds(payload.Weight),
		Pieces:        payload.Pieces,
		Total:         money(payload.TotalChargeAmount, ""),
		FreightCharge: money(payload.FreightChargeAmount, ""),
		OtherCharges:  money(payload.OtherChargeAmount, ""),
	}
	if payload.TemperatureMin != nil || payload.TemperatureMax != nil {
		document.Temperature = &TemperatureRange{
			Min: fahrenheit(payload.TemperatureMin),
			Max: fahrenheit(payload.TemperatureMax),
		}
	}
	sequence := int64(0)
	for moveIndex := range payload.Moves {
		for stopIndex := range payload.Moves[moveIndex].Stops {
			stop := &payload.Moves[moveIndex].Stops[stopIndex]
			if stop.ScheduledWindowStart == 0 {
				continue
			}
			sequence++
			zone := zones.of(stop.LocationID)
			written := Stop{
				Sequence: sequence,
				Type:     stopTypeOf(stop.Type),
				Location: ref(stop.LocationID, stop.LocationCode, stop.LocationLabel),
				Name:     stop.LocationName,
				Code:     stop.LocationCode,
				Window:   Window{Start: at(stop.ScheduledWindowStart, zone)},
				Weight:   pounds(stop.Weight),
				Pieces:   stop.Pieces,
			}
			if stop.ScheduledWindowEnd != nil && *stop.ScheduledWindowEnd > 0 {
				end := at(*stop.ScheduledWindowEnd, zone)
				written.Window.End = &end
			}
			if stop.LocationAddressLine1 != "" && stop.LocationCity != "" {
				written.Address = &Address{
					Line1:  stop.LocationAddressLine1,
					Line2:  stop.LocationAddressLine2,
					City:   stop.LocationCity,
					State:  stop.LocationStateCode,
					Postal: stop.LocationPostalCode,
				}
			}
			document.Stops = append(document.Stops, written)
		}
	}
	if len(document.Stops) == 0 {
		return nil, ErrNoStops
	}
	for index := range payload.Commodities {
		commodity := &payload.Commodities[index]
		line := CommodityLine{
			Commodity: ref(commodity.CommodityID, "", commodity.CommodityLabel),
			Description: stringutils.FirstNonEmpty(
				commodity.CommodityDescription,
				commodity.CommodityName,
				commodity.CommodityLabel,
				"Freight",
			),
		}
		if commodity.Weight > 0 {
			line.Weight = pounds(&commodity.Weight)
		}
		if commodity.Pieces > 0 {
			pieces := commodity.Pieces
			line.Pieces = &pieces
		}
		document.Commodities = append(document.Commodities, line)
	}
	for index := range payload.AdditionalCharges {
		charge := &payload.AdditionalCharges[index]
		document.Charges = append(document.Charges, Charge{
			Accessorial: ref(
				charge.AccessorialChargeID,
				charge.AccessorialCode,
				charge.AccessorialLabel,
			),
			Code:        charge.AccessorialCode,
			Description: charge.AccessorialDescription,
			Amount:      moneyOf(charge.Amount, ""),
		})
	}
	return encode(document)
}

// FromTenderResponse is the tenderResponse document of Trenova's answer to a tender.
func FromTenderResponse(
	payload *edi.TenderResponsePayload,
	key, scac string,
	respondedAt int64,
	zone string,
) (jsonutils.RawJSON, error) {
	if strings.TrimSpace(key) == "" {
		return nil, ErrNoKey
	}
	decision := DecisionDecline
	if strings.EqualFold(payload.ResponseCode, "A") {
		decision = DecisionAccept
	}
	return encode(TenderResponse{
		Key:         key,
		Decision:    decision,
		Reason:      stringutils.FirstNonEmpty(payload.RejectionReason, payload.ReasonCode),
		SCAC:        scac,
		RespondedAt: at(respondedAt, zone),
	})
}

// FromShipmentStatus is the shipmentStatus document of what happened to a load.
func FromShipmentStatus(
	payload *edi.ShipmentStatusPayload,
	key string,
	zones Zones,
) (jsonutils.RawJSON, error) {
	if strings.TrimSpace(key) == "" {
		return nil, ErrNoKey
	}
	document := ShipmentStatus{
		Key:              key,
		BOL:              payload.BOL,
		ProNumber:        payload.ProNumber,
		ShipperReference: payload.ShipperReference,
		Weight:           pounds(payload.Weight),
		Pieces:           payload.Pieces,
		Equipment:        payload.EquipmentNumber,
		Refs:             refs(payload.References),
	}
	zone := zones.of(payload.LocationID)
	if len(payload.Events) > 0 {
		for index := range payload.Events {
			event := &payload.Events[index]
			if event.EventAt == 0 {
				continue
			}
			written := StatusEvent{
				Kind: EventKindOf(
					stringutils.FirstNonEmpty(event.StatusCode, event.AppointmentCode),
				),
				Reason:    stringutils.FirstNonEmpty(event.ReasonCode, event.AppointmentReasonCode),
				At:        at(event.EventAt, zone),
				Stop:      event.StopSequence,
				City:      event.City,
				State:     event.StateCode,
				Country:   event.CountryCode,
				Equipment: event.EquipmentNumber,
			}
			if event.Latitude != nil && event.Longitude != nil {
				written.Latitude = strconv.FormatFloat(*event.Latitude, 'f', -1, 64)
				written.Longitude = strconv.FormatFloat(*event.Longitude, 'f', -1, 64)
			}
			document.Events = append(document.Events, written)
		}
	} else if occurred := firstNonZero(payload.EventTime, payload.EventDate); occurred > 0 {
		event := StatusEvent{
			Kind:    EventKindOf(payload.StatusCode),
			Reason:  stringutils.FirstNonEmpty(payload.StatusReasonCode, payload.ReasonCode),
			At:      at(occurred, zone),
			City:    payload.City,
			State:   payload.StateCode,
			Country: payload.CountryCode,
		}
		if payload.StopSequence > 0 {
			sequence := payload.StopSequence
			event.Stop = &sequence
		}
		document.Events = append(document.Events, event)
	}
	if len(document.Events) == 0 {
		return nil, ErrNoEvents
	}
	return encode(document)
}

// FromFreightInvoice is the freightInvoice document of an invoice Trenova bills.
func FromFreightInvoice(
	payload *edi.FreightInvoicePayload,
	key, zone string,
) (jsonutils.RawJSON, error) {
	if payload.InvoiceNumber == "" || payload.InvoiceDate == 0 || !payload.TotalAmount.Valid {
		return nil, ErrNoNumber
	}
	currency := payload.CurrencyCode
	document := FreightInvoice{
		Number:       payload.InvoiceNumber,
		Date:         day(payload.InvoiceDate, zone),
		DeliveryDate: day(payload.DeliveryDate, zone),
		Key:          key,
		BOL:          payload.BOL,
		ProNumber:    payload.ProNumber,
		Weight:       pounds(payload.Weight),
		Total:        moneyOf(payload.TotalAmount.Decimal, currency),
		Refs:         refs(payload.ReferenceNumbers),
	}
	if payload.BillToName != "" {
		document.BillTo = &Party{Name: payload.BillToName}
		if payload.BillToAddressLine1 != "" && payload.BillToCity != "" {
			document.BillTo.Address = &Address{
				Line1:   payload.BillToAddressLine1,
				Line2:   payload.BillToAddressLine2,
				City:    payload.BillToCity,
				State:   payload.BillToStateCode,
				Postal:  payload.BillToPostalCode,
				Country: payload.BillToCountry,
			}
		}
	}
	for index := range payload.LineCharges {
		line := &payload.LineCharges[index]
		written := InvoiceLine{
			Sequence:    line.Sequence,
			Code:        line.Code,
			Description: line.Description,
			Amount:      moneyOf(line.Amount, currency),
			Weight:      pounds(line.Weight),
		}
		if line.Rate.Valid {
			written.Rate = line.Rate.Decimal.String()
		}
		document.Lines = append(document.Lines, written)
	}
	for index := range payload.Shipments {
		shipment := &payload.Shipments[index]
		document.Shipments = append(document.Shipments, InvoicedShipment{
			ProNumber:    shipment.ProNumber,
			BOL:          shipment.BOL,
			PONumber:     shipment.PONumber,
			DeliveryDate: day(shipment.DeliveryDate, zone),
			Amount:       money(shipment.Amount, currency),
		})
	}
	return encode(document)
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
