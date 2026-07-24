package sim

import (
	"sort"
	"strings"
	"time"
)

const (
	geofenceEventEntry        = "GeofenceEntry"
	geofenceEventExit         = "GeofenceExit"
	geofenceMaxWindowLookback = 2 * time.Hour
)

type geofenceCircle struct {
	AddressID        string
	Name             string
	FormattedAddress string
	ExternalIDs      map[string]any
	Latitude         float64
	Longitude        float64
	RadiusMeters     float64
}

type geofenceTransition struct {
	EventType string
	At        time.Time
	VehicleID string
	Circle    geofenceCircle
}

func (l *LiveSimulator) loadGeofenceCircles() []geofenceCircle {
	addresses, err := l.store.List(ResourceAddresses)
	if err != nil {
		return []geofenceCircle{}
	}

	out := make([]geofenceCircle, 0, len(addresses))
	for _, address := range addresses {
		geofence, ok := anyAsMap(address["geofence"])
		if !ok {
			continue
		}
		circle, ok := anyAsMap(geofence["circle"])
		if !ok {
			continue
		}

		latitude := floatFromAny(circle["latitude"])
		longitude := floatFromAny(circle["longitude"])
		radius := floatFromAny(circle["radiusMeters"])
		if radius <= 0 || !isReasonableCoordinate(latitude, longitude) {
			continue
		}

		externalIDs := map[string]any{}
		if rawExternalIDs, okIDs := anyAsMap(address["externalIds"]); okIDs {
			externalIDs = cloneMap(rawExternalIDs)
		}
		out = append(out, geofenceCircle{
			AddressID:        recordID(address),
			Name:             stringValue(address, "name"),
			FormattedAddress: stringValue(address, "formattedAddress"),
			ExternalIDs:      externalIDs,
			Latitude:         latitude,
			Longitude:        longitude,
			RadiusMeters:     radius,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].AddressID < out[j].AddressID
	})
	return out
}

func (l *LiveSimulator) GeofenceWebhookEmissions(
	now time.Time,
	windowStart time.Time,
	windowEnd time.Time,
	vehicleIDs []string,
) []WebhookEmission {
	transitions := l.geofenceTransitions(now, windowStart, windowEnd, vehicleIDs)
	if len(transitions) == 0 {
		return []WebhookEmission{}
	}

	assets := l.loadAssetMetadata()
	out := make([]WebhookEmission, 0, len(transitions))
	for idx := range transitions {
		transition := &transitions[idx]
		out = append(out, WebhookEmission{
			EventType: transition.EventType,
			UniqueKey: strings.Join([]string{
				transition.EventType,
				transition.VehicleID,
				transition.Circle.AddressID,
				transition.At.UTC().Format(time.RFC3339),
			}, "|"),
			Data: geofenceWebhookData(transition, assets),
		})
	}
	return out
}

func (l *LiveSimulator) geofenceTransitions(
	now time.Time,
	windowStart time.Time,
	windowEnd time.Time,
	vehicleIDs []string,
) []geofenceTransition {
	circles := l.loadGeofenceCircles()
	if len(circles) == 0 {
		return []geofenceTransition{}
	}

	waypoints := l.loadAssetWaypoints()
	selected := selectAssetIDs(vehicleIDs, waypoints)
	if len(selected) == 0 {
		return []geofenceTransition{}
	}

	times := geofenceSampleTimes(windowStart, windowEnd)
	if len(times) < 2 {
		return []geofenceTransition{}
	}

	driverByVehicle := l.driverByVehicleMap()
	eventsByVehicle := l.vehicleEventsForWindow(
		selected,
		driverByVehicle,
		times[0].Add(-2*time.Hour),
		times[len(times)-1].Add(2*time.Hour),
	)

	out := make([]geofenceTransition, 0, len(selected))
	for _, vehicleID := range selected {
		points := waypoints[vehicleID]
		if len(points) == 0 {
			continue
		}
		out = append(out, l.vehicleGeofenceTransitions(
			vehicleID,
			points,
			eventsByVehicle[vehicleID],
			circles,
			times,
			now,
		)...)
	}

	sort.Slice(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.Before(out[j].At)
		}
		if out[i].VehicleID != out[j].VehicleID {
			return out[i].VehicleID < out[j].VehicleID
		}
		return out[i].Circle.AddressID < out[j].Circle.AddressID
	})
	return out
}

func (l *LiveSimulator) vehicleGeofenceTransitions(
	vehicleID string,
	points []routePoint,
	events []SimEvent,
	circles []geofenceCircle,
	times []time.Time,
	now time.Time,
) []geofenceTransition {
	out := make([]geofenceTransition, 0, 4)
	previous := make([]bool, len(circles))
	for timeIndex, sampleTime := range times {
		state := l.routeStateForSample(vehicleID, points, sampleTime, now, now)
		state = l.applyVehicleEventsToRouteState(
			vehicleID,
			points,
			events,
			sampleTime,
			now,
			now,
			state,
		)
		for circleIndex := range circles {
			circle := &circles[circleIndex]
			inside := haversineMeters(
				state.Latitude,
				state.Longitude,
				circle.Latitude,
				circle.Longitude,
			) <= circle.RadiusMeters
			if timeIndex > 0 && inside != previous[circleIndex] {
				out = append(out, geofenceTransition{
					EventType: ternary(inside, geofenceEventEntry, geofenceEventExit),
					At:        sampleTime.UTC(),
					VehicleID: vehicleID,
					Circle:    *circle,
				})
			}
			previous[circleIndex] = inside
		}
	}
	return out
}

func geofenceSampleTimes(windowStart, windowEnd time.Time) []time.Time {
	if windowEnd.Before(windowStart) {
		windowStart, windowEnd = windowEnd, windowStart
	}

	gridStart := windowStart.UTC().Truncate(defaultAssetSampleStep)
	gridEnd := windowEnd.UTC().Truncate(defaultAssetSampleStep)
	if !gridEnd.After(gridStart) {
		return []time.Time{}
	}

	steps := int(gridEnd.Sub(gridStart)/defaultAssetSampleStep) + 1
	if steps > maxAssetSamplesPerAsset+1 {
		gridStart = gridEnd.Add(-time.Duration(maxAssetSamplesPerAsset) * defaultAssetSampleStep)
		steps = maxAssetSamplesPerAsset + 1
	}

	times := make([]time.Time, 0, steps)
	for cursor := gridStart; !cursor.After(gridEnd); cursor = cursor.Add(defaultAssetSampleStep) {
		times = append(times, cursor)
	}
	return times
}

func geofenceWebhookData(
	transition *geofenceTransition,
	assets map[string]Record,
) map[string]any {
	circle := &transition.Circle
	vehicle := vehicleWebhookPayload(transition.VehicleID, assets)

	return map[string]any{
		"address": map[string]any{
			"id":               circle.AddressID,
			"name":             circle.Name,
			"formattedAddress": circle.FormattedAddress,
			"externalIds":      cloneMap(circle.ExternalIDs),
			"geofence": map[string]any{
				"circle": map[string]any{
					"latitude":     circle.Latitude,
					"longitude":    circle.Longitude,
					"radiusMeters": circle.RadiusMeters,
				},
			},
		},
		"vehicle": vehicle,
	}
}
