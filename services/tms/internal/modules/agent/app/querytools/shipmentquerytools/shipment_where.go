package shipmentquerytools

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	extraDriver        = "driverId"
	extraTractor       = "tractorId"
	extraTrailer       = "trailerId"
	extraCarrier       = "carrierId"
	extraCommodity     = "commodityId"
	extraStopLocation  = "stops.locationId"
	extraStopCity      = "stops.city"
	extraStopState     = "stops.state"
	extraStopType      = "stops.type"
	extraUnassigned    = "unassigned"
	extraLate          = "late"
	extraOnHold        = "onHold"
	extraHoldReason    = "hold.reasonCode"
	extraHoldAgeDays   = "hold.ageDays"
	extraBillingReady  = "billingReady"
	extraStopFrom      = "stops.scheduledFrom"
	extraStopTo        = "stops.scheduledTo"
	extraEquipment     = "equipment"
	extraCommodityName = "commodity"
	extraStatusAgeDays = "status.ageDays"
	secondsPerDayExtra = int64(86400)
	equipmentIDPrefix  = "et_"
)

type shipmentExtras struct {
	workers, tractors, trailers, carriers, commodities, locations []pulid.ID
	equipmentIDs                                                  []pulid.ID
	equipmentText, commodityText                                  string
	place                                                         shipmentports.ShipmentStopPlace
	unassigned, billingReady                                      *bool
	late                                                          bool
	inStatusSince                                                 int64
	held                                                          shipmentHoldFilter
}

func (e *shipmentExtras) apply(options *shipmentports.ShipmentOptions) {
	options.WorkerIDs = append(options.WorkerIDs, e.workers...)
	options.TractorIDs = append(options.TractorIDs, e.tractors...)
	options.TrailerIDs = append(options.TrailerIDs, e.trailers...)
	options.CarrierIDs = append(options.CarrierIDs, e.carriers...)
	options.CommodityIDs = append(options.CommodityIDs, e.commodities...)
	options.StopLocationIDs = append(options.StopLocationIDs, e.locations...)
	if e.place.Set() {
		options.StopPlaces = append(options.StopPlaces, e.place)
	}
	if e.unassigned != nil {
		options.Unassigned = e.unassigned
	}
	if e.billingReady != nil {
		options.BillingReady = e.billingReady
	}
	options.EquipmentTypeIDs = append(options.EquipmentTypeIDs, e.equipmentIDs...)
	if e.equipmentText != "" {
		options.EquipmentTypeText = e.equipmentText
	}
	if e.commodityText != "" {
		options.CommodityText = e.commodityText
	}
	if e.inStatusSince > 0 {
		options.InStatusSince = e.inStatusSince
	}
}

func (t *shipmentListTool) WhereFields() *domaintypes.FieldConfiguration { return t.list.Spec.Config }

const (
	kindID      = "id"
	kindText    = "text"
	kindBoolean = "boolean"
	kindNumber  = "number"
	kindDate    = "date"
	kindEnum    = "enum"
	opEq        = "eq"
	opIn        = "in"
	opGte       = "gte"
	opLte       = "lte"
	opContains  = "contains"
)

var (
	errHoldAgeDays   = errors.New("hold.ageDays takes a number of days")
	errStatusAgeDays = errors.New("status.ageDays takes a number of days")
	errEquipment     = errors.New("equipment takes one description or equipment type ids")
)

func (t *shipmentListTool) WhereExtras() []querytool.Extra {
	return append(shipmentAssetExtras(), shipmentStateExtras()...)
}

func shipmentAssetExtras() []querytool.Extra {
	ids := []string{opEq, opIn}
	texts := []string{opEq, opIn}
	stopTypes := make([]string, 0, 4)
	for _, stopType := range shipmentvocab.StopTypeValues() {
		stopTypes = append(stopTypes, string(stopType))
	}

	return []querytool.Extra{
		{Name: extraDriver, Kind: kindID, Operators: ids,
			Description: "Assigned driver, primary or secondary (worker ids)."},
		{Name: extraTractor, Kind: kindID, Operators: ids, Description: "Assigned tractor ids."},
		{Name: extraTrailer, Kind: kindID, Operators: ids, Description: "Assigned trailer ids."},
		{Name: extraCarrier, Kind: kindID, Operators: ids,
			Description: "Carrier covering a move (carrier ids)."},
		{Name: extraCommodity, Kind: kindID, Operators: ids,
			Description: "Commodity carried (commodity ids)."},
		{Name: extraCommodityName, Kind: kindText, Operators: []string{opEq, opContains},
			Description: "A commodity carried whose name contains these words."},
		{Name: extraEquipment, Kind: kindText, Operators: []string{opEq, opContains, opIn},
			Description: "Trailer or tractor equipment type: words in its code or description " +
				"(reefer), or equipment type ids."},
		{Name: extraStopLocation, Kind: kindID, Operators: ids,
			Description: "A live stop at these locations."},
		{Name: extraStopCity, Kind: kindText, Operators: texts,
			Description: "A live stop in these cities; combine with stops.type."},
		{Name: extraStopState, Kind: kindText, Operators: texts,
			Description: "A live stop in these states (two-letter codes)."},
		{Name: extraStopType, Kind: kindEnum, Operators: texts, Values: stopTypes,
			Description: "Which stops stops.city, stops.state and the scheduled window read; " +
				"without it, any pickup or delivery."},
		{Name: extraStopFrom, Kind: kindDate, Operators: []string{opGte},
			Description: "A stop whose scheduled window is open on or after this day " +
				"(YYYY-MM-DD or today); the same stop as stops.city and stops.type. Not the " +
				"actual dates."},
		{Name: extraStopTo, Kind: kindDate, Operators: []string{opLte},
			Description: "A stop whose scheduled window opens on or before this day " +
				"(YYYY-MM-DD or today)."},
	}
}

func shipmentStateExtras() []querytool.Extra {
	flag := []string{opEq}
	texts := []string{opEq, opIn}

	return []querytool.Extra{
		{Name: extraUnassigned, Kind: kindBoolean, Operators: flag,
			Description: "No live driver or carrier assignment; false for loads one covers."},
		{Name: extraLate, Kind: kindBoolean, Operators: flag,
			Description: "A stop past its window with no arrival."},
		{Name: extraOnHold, Kind: kindBoolean, Operators: flag,
			Description: "An active hold, or none when false."},
		{Name: extraHoldReason, Kind: kindText, Operators: texts,
			Description: "Active hold with these reason codes."},
		{Name: extraHoldAgeDays, Kind: kindNumber, Operators: []string{opGte},
			Description: "Active hold placed at least this many days ago."},
		{Name: extraStatusAgeDays, Kind: kindNumber, Operators: []string{opGte},
			Description: "In its current status for at least this many days."},
		{Name: extraBillingReady, Kind: kindBoolean, Operators: flag,
			Description: "Delivered and ready to transfer to billing; false for the rest."},
	}
}

func (e *shipmentExtras) idsFor(name string) *[]pulid.ID {
	switch name {
	case extraDriver:
		return &e.workers
	case extraTractor:
		return &e.tractors
	case extraTrailer:
		return &e.trailers
	case extraCarrier:
		return &e.carriers
	case extraCommodity:
		return &e.commodities
	case extraStopLocation:
		return &e.locations
	default:
		return nil
	}
}

func applyExtras(extras map[string]any, clock filtercatalog.Clock) (shipmentExtras, error) {
	var out shipmentExtras
	for name, raw := range extras {
		if into := out.idsFor(name); into != nil {
			parsed, err := ids(name, raw)
			if err != nil {
				return out, err
			}
			*into = append(*into, parsed...)

			continue
		}
		applied, err := out.applyStop(name, raw, clock)
		if err != nil || applied {
			if err != nil {
				return out, err
			}

			continue
		}
		if err = out.applyState(name, raw, clock.Instant()); err != nil {
			return out, err
		}
	}

	return out, nil
}

func (e *shipmentExtras) applyStop(name string, raw any, clock filtercatalog.Clock) (bool, error) {
	switch name {
	case extraStopCity:
		e.place.Cities = append(e.place.Cities, texts(raw)...)
	case extraStopState:
		e.place.States = append(e.place.States, texts(raw)...)
	case extraStopType:
		for _, value := range texts(raw) {
			stopType := shipmentvocab.StopType(value)
			if !stopType.IsValid() {
				return true, fmt.Errorf("stops.type %q is not a stop type; use one of %s",
					value, strings.Join([]string{"Pickup", "Delivery", "SplitPickup",
						"SplitDelivery"}, ", "))
			}
			e.place.Types = append(e.place.Types, stopType)
		}
	case extraStopFrom:
		day, err := dayOf(clock, name, raw)
		if err != nil {
			return true, err
		}
		e.place.WindowFrom = day
	case extraStopTo:
		day, err := dayOf(clock, name, raw)
		if err != nil {
			return true, err
		}
		e.place.WindowTo = clock.DayEnd(day)
	default:
		return false, nil
	}

	return true, nil
}

func (e *shipmentExtras) applyState(name string, raw any, now int64) error {
	switch name {
	case extraUnassigned:
		e.unassigned = flagOf(raw)
	case extraLate:
		e.late = truthy(raw)
	case extraOnHold:
		e.held.active = flagOf(raw)
	case extraHoldReason:
		e.held.codes = append(e.held.codes, texts(raw)...)
	case extraHoldAgeDays:
		since, err := daysAgo(raw, now, errHoldAgeDays)
		if err != nil {
			return err
		}
		e.held.before = since
	case extraStatusAgeDays:
		since, err := daysAgo(raw, now, errStatusAgeDays)
		if err != nil {
			return err
		}
		e.inStatusSince = since
	case extraBillingReady:
		e.billingReady = flagOf(raw)
	case extraEquipment:
		return e.applyEquipment(raw)
	case extraCommodityName:
		e.commodityText = strings.TrimSpace(strings.Join(texts(raw), " "))
	default:
		return fmt.Errorf("%q is not a shipment filter", name)
	}

	return nil
}

func (e *shipmentExtras) applyEquipment(raw any) error {
	for _, value := range texts(raw) {
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, equipmentIDPrefix) {
			if id, err := pulid.Parse(value); err == nil {
				e.equipmentIDs = append(e.equipmentIDs, id)
				continue
			}
		}
		if e.equipmentText != "" {
			return errEquipment
		}
		e.equipmentText = value
	}

	return nil
}

func daysAgo(raw any, now int64, refusal error) (int64, error) {
	days, ok := raw.(float64)
	if !ok || days < 0 {
		return 0, refusal
	}

	return now - int64(days)*secondsPerDayExtra, nil
}

func texts(raw any) []string {
	switch value := raw.(type) {
	case string:
		return []string{value}
	case []string:
		return value
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}

		return out
	}

	return nil
}

func ids(name string, raw any) ([]pulid.ID, error) {
	values := texts(raw)
	if len(values) == 0 {
		return nil, fmt.Errorf("%s takes record ids", name)
	}
	out := make([]pulid.ID, 0, len(values))
	for _, value := range values {
		id, err := pulid.Parse(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf(
				"%s takes record ids, and %q is not one; look the record up first",
				name,
				value,
			)
		}
		out = append(out, id)
	}

	return out, nil
}

func dayOf(clock filtercatalog.Clock, name string, raw any) (int64, error) {
	if value, ok := raw.(float64); ok && value > 0 {
		return clock.DayStart(int64(value)), nil
	}
	text, _ := raw.(string)
	day, ok := clock.Day(text)
	if !ok {
		return 0, fmt.Errorf("%s takes a day as YYYY-MM-DD or today, not %v", name, raw)
	}

	return day, nil
}

func flagOf(raw any) *bool {
	on := truthy(raw)

	return &on
}

func truthy(raw any) bool {
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		return value == "true"
	}

	return false
}
