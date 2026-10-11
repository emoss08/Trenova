package sim

import (
	"sort"
	"strings"
	"time"
)

type routeStopRef struct {
	ID          string
	Name        string
	ExternalIDs map[string]any
}

type routeRef struct {
	ID          string
	Name        string
	DriverID    string
	VehicleID   string
	ExternalIDs map[string]any
	Stops       []routeStopRef
}

type routeStopMatch struct {
	Route routeRef
	Stop  routeStopRef
}

func (r *routeRef) firstStop() (routeStopRef, bool) {
	if len(r.Stops) == 0 {
		return routeStopRef{}, false
	}
	return r.Stops[0], true
}

func (r *routeRef) lastStop() (routeStopRef, bool) {
	if len(r.Stops) == 0 {
		return routeStopRef{}, false
	}
	return r.Stops[len(r.Stops)-1], true
}

func (r *routeRef) tiny() map[string]any {
	out := map[string]any{keyID: r.ID, fieldExternalIDs: cloneMap(r.ExternalIDs)}
	if r.Name != "" {
		out[keyName] = r.Name
	}
	return out
}

func (s routeStopRef) tiny() map[string]any {
	out := map[string]any{keyID: s.ID, fieldExternalIDs: cloneMap(s.ExternalIDs)}
	if s.Name != "" {
		out[keyName] = s.Name
	}
	return out
}

func (l *LiveSimulator) routeRefs(now time.Time) []routeRef {
	routes := l.fleet().routes
	out := make([]routeRef, 0, len(routes))
	for _, route := range routes {
		routeID := recordID(route)
		if routeID == "" {
			continue
		}
		rendered, ok := l.RouteByID(now, routeID)
		if !ok {
			continue
		}
		out = append(out, routeRefFromRecord(rendered))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func routeRefFromRecord(route Record) routeRef {
	ref := routeRef{
		ID:          recordID(route),
		Name:        stringValue(route, keyName),
		DriverID:    nestedString(route, keyDriver, keyID),
		VehicleID:   nestedString(route, keyVehicle, keyID),
		ExternalIDs: renderExternalIDs(route, nil),
	}
	rawStops := listOf(route["stops"])
	ref.Stops = make([]routeStopRef, 0, len(rawStops))
	for _, raw := range rawStops {
		stop, ok := anyAsMap(raw)
		if !ok {
			continue
		}
		stopID := strings.TrimSpace(stringOf(stop[keyID]))
		if stopID == "" {
			continue
		}
		ref.Stops = append(ref.Stops, routeStopRef{
			ID:          stopID,
			Name:        strings.TrimSpace(stringOf(stop[keyName])),
			ExternalIDs: renderExternalIDs(Record(stop), nil),
		})
	}
	return ref
}

func routeRefsByDriver(refs []routeRef) map[string]routeRef {
	out := make(map[string]routeRef, len(refs))
	for idx := range refs {
		driverID := refs[idx].DriverID
		if driverID == "" {
			continue
		}
		if _, exists := out[driverID]; !exists {
			out[driverID] = refs[idx]
		}
	}
	return out
}

func findRouteStop(refs []routeRef, raw string, allowExternal bool) (routeStopMatch, bool) {
	ref := parseRecordRef(raw)
	if ref.Raw == "" || (ref.isExternal() && !allowExternal) {
		return routeStopMatch{}, false
	}
	for idx := range refs {
		for _, stop := range refs[idx].Stops {
			if ref.isExternal() {
				if value, ok := stop.ExternalIDs[ref.Key].(string); ok && value == ref.Value {
					return routeStopMatch{Route: refs[idx], Stop: stop}, true
				}
				continue
			}
			if stop.ID == ref.Raw {
				return routeStopMatch{Route: refs[idx], Stop: stop}, true
			}
		}
	}
	return routeStopMatch{}, false
}
