package sim

import (
	"strings"
	"time"
)

const (
	routeStopEventArrival   = "RouteStopArrival"
	routeStopEventDeparture = "RouteStopDeparture"

	routeStopUpdateType        = "route tracking"
	routeStopOperationArrived  = "stop arrived"
	routeStopOperationDeparted = "stop departed"
	routeStopStateArrived      = "arrived"
	routeStopStateDeparted     = "departed"
)

type routeWebhookContext struct {
	RouteID           string
	RouteName         string
	RouteExternalIDs  map[string]any
	DriverID          string
	DriverName        string
	DriverExternalIDs map[string]any
}

func (l *LiveSimulator) RouteStopWebhookEmissions(
	now time.Time,
	windowStart time.Time,
	windowEnd time.Time,
	vehicleIDs []string,
) []WebhookEmission {
	transitions := l.geofenceTransitions(now, windowStart, windowEnd, vehicleIDs)
	if len(transitions) == 0 {
		return []WebhookEmission{}
	}
	routeByVehicle := l.routeContextByVehicleMap()
	if len(routeByVehicle) == 0 {
		return []WebhookEmission{}
	}

	assets := l.loadAssetMetadata()
	out := make([]WebhookEmission, 0, len(transitions))
	for idx := range transitions {
		transition := &transitions[idx]
		route, ok := routeByVehicle[transition.VehicleID]
		if !ok {
			continue
		}
		arrival := transition.EventType == geofenceEventEntry
		eventType := routeStopEventDeparture
		if arrival {
			eventType = routeStopEventArrival
		}
		out = append(out, WebhookEmission{
			EventType: eventType,
			UniqueKey: strings.Join([]string{
				eventType,
				transition.VehicleID,
				transition.Circle.AddressID,
				transition.At.UTC().Format(time.RFC3339),
			}, "|"),
			Data: l.routeStopWebhookData(transition, &route, assets, arrival),
		})
	}
	return out
}

func (l *LiveSimulator) routeContextByVehicleMap() map[string]routeWebhookContext {
	routes, err := l.store.List(ResourceRoutes)
	if err != nil {
		return map[string]routeWebhookContext{}
	}
	roster := l.loadDriverRoster()
	out := make(map[string]routeWebhookContext, len(routes))
	for _, route := range routes {
		vehicleID := nestedString(route, "vehicle", "id")
		if vehicleID == "" {
			continue
		}
		if _, exists := out[vehicleID]; exists {
			continue
		}
		driverID := nestedString(route, "driver", "id")
		context := routeWebhookContext{
			RouteID:           recordID(route),
			RouteName:         stringValue(route, "name"),
			RouteExternalIDs:  externalIDsFromRecord(route),
			DriverID:          driverID,
			DriverName:        firstNonEmpty(nestedString(route, "driver", "name"), roster[driverID].Name, driverID),
			DriverExternalIDs: nestedExternalIDs(route, "driver"),
		}
		out[vehicleID] = context
	}
	return out
}

func (l *LiveSimulator) routeStopWebhookData(
	transition *geofenceTransition,
	route *routeWebhookContext,
	assets map[string]Record,
	arrival bool,
) map[string]any {
	circle := &transition.Circle
	stopID := strings.Join([]string{route.RouteID, "stop", circle.AddressID}, "-")

	operation := routeStopOperationDeparted
	state := routeStopStateDeparted
	if arrival {
		operation = routeStopOperationArrived
		state = routeStopStateArrived
	}

	dwell := time.Duration(
		(15 + 30*l.hashFraction("route-stop-dwell", route.RouteID, circle.AddressID)) *
			float64(time.Minute),
	)
	enRouteLead := time.Duration(
		(20 + 30*l.hashFraction("route-stop-enroute", route.RouteID, circle.AddressID)) *
			float64(time.Minute),
	)

	routeStopDetails := map[string]any{
		"id":          stopID,
		"state":       state,
		"externalIds": cloneMap(circle.ExternalIDs),
		"orders":      []any{},
	}
	if arrival {
		arrivalTime := transition.At.UTC()
		enRouteTime := arrivalTime.Add(-enRouteLead)
		routeStopDetails["enRouteTime"] = enRouteTime.Format(time.RFC3339)
		routeStopDetails["eta"] = arrivalTime.Format(time.RFC3339)
		routeStopDetails["actualArrivalTime"] = arrivalTime.Format(time.RFC3339)
	} else {
		departureTime := transition.At.UTC()
		arrivalTime := departureTime.Add(-dwell)
		enRouteTime := arrivalTime.Add(-enRouteLead)
		routeStopDetails["enRouteTime"] = enRouteTime.Format(time.RFC3339)
		routeStopDetails["eta"] = arrivalTime.Format(time.RFC3339)
		routeStopDetails["actualArrivalTime"] = arrivalTime.Format(time.RFC3339)
		routeStopDetails["actualDepartureTime"] = departureTime.Format(time.RFC3339)
	}

	return map[string]any{
		"operation":        operation,
		"type":             routeStopUpdateType,
		"time":             transition.At.UTC().Format(time.RFC3339),
		"assignedToRoute":  route.RouteID,
		"driver":           routeStopDriverPayload(route),
		"vehicle":          vehicleWebhookPayload(transition.VehicleID, assets),
		"route":            routeStopRoutePayload(route),
		"routeStopDetails": routeStopDetails,
	}
}

func routeStopDriverPayload(route *routeWebhookContext) map[string]any {
	if strings.TrimSpace(route.DriverID) == "" {
		return map[string]any{"externalIds": map[string]any{}}
	}
	return map[string]any{
		"id":          route.DriverID,
		"name":        route.DriverName,
		"externalIds": cloneMap(route.DriverExternalIDs),
	}
}

func routeStopRoutePayload(route *routeWebhookContext) map[string]any {
	return map[string]any{
		"id":          route.RouteID,
		"name":        route.RouteName,
		"externalIds": cloneMap(route.RouteExternalIDs),
	}
}

func externalIDsFromRecord(record Record) map[string]any {
	if externalIDs, ok := anyAsMap(record["externalIds"]); ok {
		return cloneMap(externalIDs)
	}
	return map[string]any{}
}

func nestedExternalIDs(record Record, key string) map[string]any {
	if nested, ok := anyAsMap(record[key]); ok {
		if externalIDs, okIDs := anyAsMap(nested["externalIds"]); okIDs {
			return cloneMap(externalIDs)
		}
	}
	return map[string]any{}
}
