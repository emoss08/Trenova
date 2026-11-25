package shipmentquerytools

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/domaintypes"
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
	secondsPerDayExtra = int64(86400)
)

type shipmentExtras struct {
	workers, tractors, trailers, carriers, commodities, locations []pulid.ID
	place                                                         shipmentports.ShipmentStopPlace
	unassigned, late, billingReady                                bool
	held                                                          shipmentHoldFilter
}

func (e *shipmentExtras) apply(options *shipmentports.ShipmentOptions) {
	options.WorkerIDs = append(options.WorkerIDs, e.workers...)
	options.TractorIDs = append(options.TractorIDs, e.tractors...)
	options.TrailerIDs = append(options.TrailerIDs, e.trailers...)
	options.CarrierIDs = append(options.CarrierIDs, e.carriers...)
	options.CommodityIDs = append(options.CommodityIDs, e.commodities...)
	options.StopLocationIDs = append(options.StopLocationIDs, e.locations...)
	if len(e.place.Cities)+len(e.place.States)+len(e.place.Types) > 0 {
		options.StopPlaces = append(options.StopPlaces, e.place)
	}
	options.Unassigned = options.Unassigned || e.unassigned
	options.BillingTransferEligible = options.BillingTransferEligible || e.billingReady
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
		{Name: extraDriver, Kind: "id", Operators: ids, Description: "Assigned driver, primary or secondary (worker ids)."},
		{Name: extraTractor, Kind: "id", Operators: ids, Description: "Assigned tractor ids."},
		{Name: extraTrailer, Kind: "id", Operators: ids, Description: "Assigned trailer ids."},
		{Name: extraCarrier, Kind: "id", Operators: ids, Description: "Carrier covering a move (carrier ids)."},
		{Name: extraCommodity, Kind: "id", Operators: ids, Description: "Commodity carried (commodity ids)."},
		{Name: extraStopLocation, Kind: "id", Operators: ids, Description: "A live stop at these locations."},
		{Name: extraStopCity, Kind: "text", Operators: texts, Description: "A live stop in these cities; combine with stops.type."},
		{Name: extraStopState, Kind: "text", Operators: texts, Description: "A live stop in these states (two-letter codes)."},
		{Name: extraStopType, Kind: "enum", Operators: texts, Values: stopTypes, Description: "Which stops stops.city and stops.state read."},
		{Name: extraUnassigned, Kind: "boolean", Operators: flag, Description: "No live driver or carrier assignment."},
		{Name: extraLate, Kind: "boolean", Operators: flag, Description: "A stop past its window with no arrival."},
		{Name: extraOnHold, Kind: "boolean", Operators: flag, Description: "An active hold, or none when false."},
		{Name: extraHoldReason, Kind: "text", Operators: texts, Description: "Active hold with these reason codes."},
		{Name: extraHoldAgeDays, Kind: "number", Operators: []string{"gte"}, Description: "Active hold placed at least this many days ago."},
		{Name: extraBillingReady, Kind: "boolean", Operators: flag, Description: "Delivered and ready to transfer to billing."},
	}
}

func applyExtras(extras map[string]any, now int64) (shipmentExtras, error) {
	var out shipmentExtras
	for name, raw := range extras {
		switch name {
		case extraDriver:
			out.workers = append(out.workers, ids(raw)...)
		case extraTractor:
			out.tractors = append(out.tractors, ids(raw)...)
		case extraTrailer:
			out.trailers = append(out.trailers, ids(raw)...)
		case extraCarrier:
			out.carriers = append(out.carriers, ids(raw)...)
		case extraCommodity:
			out.commodities = append(out.commodities, ids(raw)...)
		case extraStopLocation:
			out.locations = append(out.locations, ids(raw)...)
		case extraStopCity:
			out.place.Cities = append(out.place.Cities, texts(raw)...)
		case extraStopState:
			out.place.States = append(out.place.States, texts(raw)...)
		case extraStopType:
			for _, value := range texts(raw) {
				stopType := shipmentvocab.StopType(value)
				if !stopType.IsValid() {
					return out, fmt.Errorf("stops.type %q is not a stop type; use one of %s", value,
						strings.Join([]string{"Pickup", "Delivery", "SplitPickup", "SplitDelivery"}, ", "))
				}
				out.place.Types = append(out.place.Types, stopType)
			}
		case extraUnassigned:
			out.unassigned = truthy(raw)
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
			out.billingReady = truthy(raw)
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

func ids(raw any) []pulid.ID {
	values := texts(raw)
	out := make([]pulid.ID, 0, len(values))
	for _, value := range values {
		if id, err := pulid.Parse(value); err == nil {
			out = append(out, id)
		}
	}

	return out
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
