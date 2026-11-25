package shipmentquerytools

import (
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

func (t *shipmentListTool) WhereExtras() []querytool.Extra {
	ids := []string{"eq", "in"}
	texts := []string{"eq", "in"}
	flag := []string{"eq"}
	stopTypes := make([]string, 0, 4)
	for _, stopType := range shipmentvocab.StopTypeValues() {
		stopTypes = append(stopTypes, string(stopType))
	}

	return []querytool.Extra{
		{
			Name:        extraDriver,
			Kind:        "id",
			Operators:   ids,
			Description: "Assigned driver, primary or secondary (worker ids).",
		},
		{Name: extraTractor, Kind: "id", Operators: ids, Description: "Assigned tractor ids."},
		{Name: extraTrailer, Kind: "id", Operators: ids, Description: "Assigned trailer ids."},
		{
			Name:        extraCarrier,
			Kind:        "id",
			Operators:   ids,
			Description: "Carrier covering a move (carrier ids).",
		},
		{
			Name:        extraCommodity,
			Kind:        "id",
			Operators:   ids,
			Description: "Commodity carried (commodity ids).",
		},
		{
			Name:        extraStopLocation,
			Kind:        "id",
			Operators:   ids,
			Description: "A live stop at these locations.",
		},
		{
			Name:        extraStopCity,
			Kind:        "text",
			Operators:   texts,
			Description: "A live stop in these cities; combine with stops.type.",
		},
		{
			Name:        extraStopState,
			Kind:        "text",
			Operators:   texts,
			Description: "A live stop in these states (two-letter codes).",
		},
		{
			Name:        extraStopType,
			Kind:        "enum",
			Operators:   texts,
			Values:      stopTypes,
			Description: "Which stops stops.city, stops.state and the scheduled window read; without it, any pickup or delivery.",
		},
		{
			Name:        extraStopFrom,
			Kind:        "date",
			Operators:   []string{"gte"},
			Description: "A stop whose scheduled window is open on or after this day (YYYY-MM-DD or today); the same stop as stops.city and stops.type. Not the actual dates.",
		},
		{
			Name:        extraStopTo,
			Kind:        "date",
			Operators:   []string{"lte"},
			Description: "A stop whose scheduled window opens on or before this day (YYYY-MM-DD or today).",
		},
		{
			Name:        extraEquipment,
			Kind:        "text",
			Operators:   []string{"eq", "contains", "in"},
			Description: "Trailer or tractor equipment type: words in its code or description (reefer), or equipment type ids.",
		},
		{
			Name:        extraCommodityName,
			Kind:        "text",
			Operators:   []string{"eq", "contains"},
			Description: "A commodity carried whose name contains these words.",
		},
		{
			Name:        extraStatusAgeDays,
			Kind:        "number",
			Operators:   []string{"gte"},
			Description: "In its current status for at least this many days.",
		},
		{
			Name:        extraUnassigned,
			Kind:        "boolean",
			Operators:   flag,
			Description: "No live driver or carrier assignment; false for loads one covers.",
		},
		{
			Name:        extraLate,
			Kind:        "boolean",
			Operators:   flag,
			Description: "A stop past its window with no arrival.",
		},
		{
			Name:        extraOnHold,
			Kind:        "boolean",
			Operators:   flag,
			Description: "An active hold, or none when false.",
		},
		{
			Name:        extraHoldReason,
			Kind:        "text",
			Operators:   texts,
			Description: "Active hold with these reason codes.",
		},
		{
			Name:        extraHoldAgeDays,
			Kind:        "number",
			Operators:   []string{"gte"},
			Description: "Active hold placed at least this many days ago.",
		},
		{
			Name:        extraBillingReady,
			Kind:        "boolean",
			Operators:   flag,
			Description: "Delivered and ready to transfer to billing; false for the rest.",
		},
	}
}

func applyExtras(extras map[string]any, clock filtercatalog.Clock) (shipmentExtras, error) {
	var out shipmentExtras
	now := clock.Instant()
	for name, raw := range extras {
		switch name {
		case extraDriver:
			parsed, err := ids(name, raw)
			if err != nil {
				return out, err
			}
			out.workers = append(out.workers, parsed...)
		case extraTractor:
			parsed, err := ids(name, raw)
			if err != nil {
				return out, err
			}
			out.tractors = append(out.tractors, parsed...)
		case extraTrailer:
			parsed, err := ids(name, raw)
			if err != nil {
				return out, err
			}
			out.trailers = append(out.trailers, parsed...)
		case extraCarrier:
			parsed, err := ids(name, raw)
			if err != nil {
				return out, err
			}
			out.carriers = append(out.carriers, parsed...)
		case extraCommodity:
			parsed, err := ids(name, raw)
			if err != nil {
				return out, err
			}
			out.commodities = append(out.commodities, parsed...)
		case extraStopLocation:
			parsed, err := ids(name, raw)
			if err != nil {
				return out, err
			}
			out.locations = append(out.locations, parsed...)
		case extraStopCity:
			out.place.Cities = append(out.place.Cities, texts(raw)...)
		case extraStopState:
			out.place.States = append(out.place.States, texts(raw)...)
		case extraStopType:
			for _, value := range texts(raw) {
				stopType := shipmentvocab.StopType(value)
				if !stopType.IsValid() {
					return out, fmt.Errorf(
						"stops.type %q is not a stop type; use one of %s",
						value,
						strings.Join(
							[]string{"Pickup", "Delivery", "SplitPickup", "SplitDelivery"},
							", ",
						),
					)
				}
				out.place.Types = append(out.place.Types, stopType)
			}
		case extraUnassigned:
			out.unassigned = flagOf(raw)
		case extraLate:
			out.late = truthy(raw)
		case extraOnHold:
			on := truthy(raw)
			out.held.active = &on
		case extraHoldReason:
			out.held.codes = append(out.held.codes, texts(raw)...)
		case extraHoldAgeDays:
			days, ok := raw.(float64)
			if !ok || days < 0 {
				return out, fmt.Errorf("hold.ageDays takes a number of days")
			}
			out.held.before = now - int64(days)*secondsPerDayExtra
		case extraBillingReady:
			out.billingReady = flagOf(raw)
		case extraStopFrom:
			day, err := dayOf(clock, name, raw)
			if err != nil {
				return out, err
			}
			out.place.WindowFrom = day
		case extraStopTo:
			day, err := dayOf(clock, name, raw)
			if err != nil {
				return out, err
			}
			out.place.WindowTo = clock.DayEnd(day)
		case extraEquipment:
			for _, value := range texts(raw) {
				value = strings.TrimSpace(value)
				if id, err := pulid.Parse(
					value,
				); err == nil &&
					strings.HasPrefix(value, equipmentIDPrefix) {
					out.equipmentIDs = append(out.equipmentIDs, id)
					continue
				}
				if out.equipmentText != "" {
					return out, fmt.Errorf("equipment takes one description or equipment type ids")
				}
				out.equipmentText = value
			}
		case extraCommodityName:
			out.commodityText = strings.TrimSpace(strings.Join(texts(raw), " "))
		case extraStatusAgeDays:
			days, ok := raw.(float64)
			if !ok || days < 0 {
				return out, fmt.Errorf("status.ageDays takes a number of days")
			}
			out.inStatusSince = now - int64(days)*secondsPerDayExtra
		default:
			return out, fmt.Errorf("%q is not a shipment filter", name)
		}
	}

	return out, nil
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
