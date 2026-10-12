package sim

import (
	"fmt"
	"math"
	"sync"
	"time"
)

var (
	centralZoneOnce sync.Once
	centralZone     *time.Location
)

const (
	fieldSimCoupledVehicleID = "simCoupledVehicleId"
	fieldSimMountedOnAssetID = "simMountedOnAssetId"
	fieldSimParkedAddressID  = "simParkedAddressId"
	fieldSimParkedLocation   = "simParkedLocation"
	fieldSimYardLoop         = "simYardLoop"

	trailerHitchOffsetMeters = 12.0
	parkedSpreadMeters       = 70.0
	yardLoopRadiusMeters     = 120.0
	yardLoopSpeedMPS         = 3.6
	yardShiftStartHour       = 6
	yardShiftEndHour         = 18
	reverseGeoCityRadiusMi   = 1.5
	reverseGeoMaxRangeMi     = 75.0
	maxHostChainDepth        = 4
	trackEventPadding        = 2 * time.Hour
)

type trackKind uint8

const (
	trackNone trackKind = iota
	trackRoute
	trackParked
	trackYardLoop
)

type positionTrack struct {
	AssetID      string
	HostID       string
	Kind         trackKind
	Geometry     *routeGeometry
	Points       []routePoint
	Events       []SimEvent
	EventsFrom   time.Time
	EventsTo     time.Time
	Parked       routeState
	Yard         routeState
	HitchOffsetM float64
}

type gazetteerCity struct {
	Name      string
	Latitude  float64
	Longitude float64
}

var texasGazetteer = []gazetteerCity{
	{"Abilene", 32.4487, -99.7331}, {"Alice", 27.7522, -98.0697},
	{"Amarillo", 35.2220, -101.8313}, {"Arlington", 32.7357, -97.1081},
	{"Austin", 30.2672, -97.7431}, {"Baytown", 29.7355, -94.9774},
	{"Beaumont", 30.0802, -94.1266}, {"Big Spring", 32.2504, -101.4787},
	{"Brownsville", 25.9017, -97.4975}, {"Bryan", 30.6744, -96.3700},
	{"Canton", 32.5565, -95.8633}, {"Childress", 34.4265, -100.2040},
	{"Conroe", 30.3119, -95.4561}, {"Corpus Christi", 27.8006, -97.3964},
	{"Corsicana", 32.0954, -96.4689}, {"Dallas", 32.7767, -96.7970},
	{"Denton", 33.2148, -97.1331}, {"El Paso", 31.7619, -106.4850},
	{"Ennis", 32.3293, -96.6253}, {"Fort Stockton", 30.8940, -102.8793},
	{"Fort Worth", 32.7555, -97.3308}, {"George West", 28.3325, -98.1175},
	{"Georgetown", 30.6333, -97.6780}, {"Harlingen", 26.1906, -97.6961},
	{"Hillsboro", 32.0107, -97.1300}, {"Houston", 29.7604, -95.3698},
	{"Humble", 29.9988, -95.2622}, {"Katy", 29.7858, -95.8245},
	{"Killeen", 31.1171, -97.7278}, {"Kingsville", 27.5159, -97.8561},
	{"Laredo", 27.5306, -99.4803}, {"Lindale", 32.5157, -95.4094},
	{"Longview", 32.5007, -94.7405}, {"Lubbock", 33.5779, -101.8552},
	{"McAllen", 26.2034, -98.2300}, {"Memphis", 34.7248, -100.5340},
	{"Mesquite", 32.7668, -96.5992}, {"Midland", 31.9973, -102.0779},
	{"Mineral Wells", 32.8085, -98.1128}, {"New Braunfels", 29.7030, -98.1245},
	{"Odessa", 31.8457, -102.3676}, {"Pasadena", 29.6911, -95.2091},
	{"Pecos", 31.4229, -103.4932}, {"Plainview", 34.1848, -101.7068},
	{"Pleasanton", 28.9672, -98.4786}, {"Raymondville", 26.4815, -97.7831},
	{"Round Rock", 30.5083, -97.6789}, {"San Antonio", 29.4241, -98.4936},
	{"San Marcos", 29.8833, -97.9414}, {"Seguin", 29.5688, -97.9647},
	{"Sinton", 28.0367, -97.5092}, {"Snyder", 32.7179, -100.9176},
	{"Sweetwater", 32.4709, -100.4059}, {"Temple", 31.0982, -97.3428},
	{"Terrell", 32.7360, -96.2753}, {"Tyler", 32.3513, -95.3011},
	{"Van Horn", 31.0399, -104.8308}, {"Vernon", 34.1545, -99.2651},
	{"Victoria", 28.8053, -97.0036}, {"Waco", 31.5493, -97.1467},
	{"Weatherford", 32.7593, -97.7973}, {"Wichita Falls", 33.9137, -98.4934},
}

func (v *fleetView) trackFor(assetID string, windowStart, windowEnd time.Time) *positionTrack {
	return v.live.trackFor(v.snap, assetID, windowStart, windowEnd)
}

func (l *LiveSimulator) trackFor(
	snapshot *fleetSnapshot,
	assetID string,
	windowStart time.Time,
	windowEnd time.Time,
) *positionTrack {
	track := &positionTrack{AssetID: assetID}
	hostID := assetID
	for depth := 0; depth < maxHostChainDepth; depth++ {
		asset, ok := snapshot.assetByID[hostID]
		if !ok {
			return track
		}
		if parked, isParked := snapshot.parkedState(asset); isParked {
			track.Kind = trackParked
			if asset[fieldSimYardLoop] == true {
				track.Kind = trackYardLoop
				track.Yard = snapshot.yardCenter(asset)
			}
			track.Parked = parked
			track.HostID = hostID
			if hostID != assetID {
				track.Parked = offsetBehind(parked, track.HitchOffsetM)
			}
			return track
		}
		if stringValue(asset, keyType) == assetTypeVehicle {
			points := snapshot.waypoints[hostID]
			if len(points) == 0 {
				return track
			}
			track.Kind = trackRoute
			track.HostID = hostID
			track.Points = points
			track.Geometry = snapshot.geometries[hostID]
			track.EventsFrom = windowStart.Add(-trackEventPadding)
			track.EventsTo = windowEnd.Add(trackEventPadding)
			track.Events = l.vehicleEventsAround(hostID, track.EventsFrom, track.EventsTo)
			return track
		}
		next := firstNonEmptyField(asset, fieldSimCoupledVehicleID, fieldSimMountedOnAssetID)
		if next == "" {
			return track
		}
		if stringValue(asset, fieldSimCoupledVehicleID) != "" {
			track.HitchOffsetM += trailerHitchOffsetMeters
		}
		hostID = next
	}
	return track
}

func firstNonEmptyField(record Record, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(record, key); value != "" {
			return value
		}
	}
	return ""
}

func (f *fleetSnapshot) parkedState(asset Record) (routeState, bool) {
	if location, ok := anyAsMap(asset[fieldSimParkedLocation]); ok {
		latitude := floatFromAny(location[keyLatitude])
		longitude := floatFromAny(location[keyLongitude])
		if isReasonableCoordinate(latitude, longitude) && (latitude != 0 || longitude != 0) {
			return routeState{
				Latitude:  latitude,
				Longitude: longitude,
				Heading:   normalizeHeading(floatFromAny(location[keyHeadingDegrees])),
			}, true
		}
	}
	addressID := stringValue(asset, fieldSimParkedAddressID)
	if addressID == "" {
		return routeState{}, false
	}
	address, ok := f.addressByID[addressID]
	if !ok {
		return routeState{}, false
	}
	latitude := floatFromAny(address[keyLatitude])
	longitude := floatFromAny(address[keyLongitude])
	if !isReasonableCoordinate(latitude, longitude) || (latitude == 0 && longitude == 0) {
		return routeState{}, false
	}
	angle := 2 * math.Pi * fractionFromHash("parked-angle|"+recordID(asset))
	distance := parkedSpreadMeters * (0.3 + 0.7*fractionFromHash("parked-distance|"+recordID(asset)))
	latitude += distance * math.Cos(angle) / metersPerDegreeLatitude
	if perDegree := metersPerDegreeLongitude(latitude); perDegree > 0 {
		longitude += distance * math.Sin(angle) / perDegree
	}
	return routeState{
		Latitude:  latitude,
		Longitude: longitude,
		Heading:   math.Round(360 * fractionFromHash("parked-heading|"+recordID(asset))),
	}, true
}

func fractionFromHash(key string) float64 {
	return float64(fnvHash64(key)%10000) / 10000.0
}

func (l *LiveSimulator) trackStateAt(track *positionTrack, at time.Time) (routeState, bool) {
	switch track.Kind {
	case trackParked:
		return track.Parked, true
	case trackRoute:
		geometry := track.Geometry
		if geometry == nil {
			built := l.buildRouteGeometry(track.HostID, track.Points)
			geometry = &built
		}
		events := track.Events
		if at.Before(track.EventsFrom) || at.After(track.EventsTo) {
			events = l.vehicleEventsAround(
				track.HostID,
				at.Add(-trackEventPadding),
				at.Add(trackEventPadding),
			)
		}
		state := l.routeStateForGeometry(track.HostID, geometry, at, at, at)
		state = l.applyVehicleEventsToGeometryState(
			track.HostID,
			geometry,
			events,
			at,
			at,
			at,
			state,
		)
		if track.HitchOffsetM > 0 {
			state = offsetBehind(state, track.HitchOffsetM)
		}
		return state, true
	case trackYardLoop:
		return yardLoopState(track, at), true
	case trackNone:
		return routeState{}, false
	default:
		return routeState{}, false
	}
}

func (l *LiveSimulator) vehicleEventsAround(vehicleID string, from, to time.Time) []SimEvent {
	return l.vehicleEventsForWindow([]string{vehicleID}, l.driverByVehicleMap(), from, to)[vehicleID]
}

func offsetBehind(state routeState, meters float64) routeState {
	if meters <= 0 {
		return state
	}
	heading := radians(state.Heading)
	state.Latitude -= meters * math.Cos(heading) / metersPerDegreeLatitude
	if perDegree := metersPerDegreeLongitude(state.Latitude); perDegree > 0 {
		state.Longitude -= meters * math.Sin(heading) / perDegree
	}
	return state
}

func (f *fleetSnapshot) reverseGeocode(state routeState) string {
	if circle := f.geofenceAt(state.Latitude, state.Longitude); circle != nil &&
		circle.FormattedAddress != "" {
		return circle.FormattedAddress
	}
	if formatted := formattedLocationFromAddress(state.Address); formatted != "" {
		return formatted
	}
	return nearestPlaceDescription(state.Latitude, state.Longitude)
}

func nearestPlaceDescription(latitude, longitude float64) string {
	if !isReasonableCoordinate(latitude, longitude) {
		return ""
	}
	best := -1
	bestMeters := math.MaxFloat64
	for idx := range texasGazetteer {
		city := &texasGazetteer[idx]
		meters := haversineMeters(latitude, longitude, city.Latitude, city.Longitude)
		if meters < bestMeters {
			best, bestMeters = idx, meters
		}
	}
	if best < 0 {
		return ""
	}
	city := &texasGazetteer[best]
	miles := bestMeters / metersPerMile
	if miles > reverseGeoMaxRangeMi {
		return ""
	}
	if miles <= reverseGeoCityRadiusMi {
		return city.Name + ", TX"
	}
	direction := compassPoint(bearingDegrees(city.Latitude, city.Longitude, latitude, longitude))
	return fmt.Sprintf("%.1f mi %s of %s, TX", miles, direction, city.Name)
}

func compassPoint(heading float64) string {
	points := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	index := int(math.Round(normalizeHeading(heading)/45)) % len(points)
	return points[index]
}

func (f *fleetSnapshot) addressRef(state routeState) map[string]any {
	circle := f.geofenceAt(state.Latitude, state.Longitude)
	if circle == nil {
		return nil
	}
	return map[string]any{keyID: circle.AddressID, keyName: circle.Name}
}

func ambientTemperatureMilliC(latitude float64, at time.Time, assetKey string) int64 {
	dayOfYear := float64(at.UTC().YearDay())
	seasonal := 20.5 + 9.5*math.Sin(2*math.Pi*(dayOfYear-105)/365)
	localHour := float64(at.UTC().Add(-6*time.Hour).Hour()) + float64(at.UTC().Minute())/60
	diurnal := 6.5 * math.Sin(2*math.Pi*(localHour-9)/24)
	latitudeAdjust := (30.0 - latitude) * 0.55
	jitter := (fractionFromHash(assetKey+"|"+at.UTC().Truncate(10*time.Minute).Format(time.RFC3339)) - 0.5) * 0.8
	return int64(math.Round((seasonal + diurnal + latitudeAdjust + jitter) * 1000))
}

func speedMilesPerHour(state routeState) float64 {
	return round(state.SpeedMPS*2.23694, 2)
}

func formatSampleTime(at time.Time) string {
	return at.UTC().Format(time.RFC3339)
}

func (f *fleetSnapshot) yardCenter(asset Record) routeState {
	address, ok := f.addressByID[stringValue(asset, fieldSimParkedAddressID)]
	if !ok {
		return routeState{}
	}
	return routeState{
		Latitude:  floatFromAny(address[keyLatitude]),
		Longitude: floatFromAny(address[keyLongitude]),
	}
}

func yardShiftActive(at time.Time) bool {
	local := at.In(centralTime())
	if local.Weekday() == time.Sunday {
		return false
	}
	return local.Hour() >= yardShiftStartHour && local.Hour() < yardShiftEndHour
}

func yardLoopState(track *positionTrack, at time.Time) routeState {
	if !yardShiftActive(at) {
		return track.Parked
	}
	circumference := 2 * math.Pi * yardLoopRadiusMeters
	phase := math.Mod(float64(at.Unix())*yardLoopSpeedMPS, circumference) / circumference
	angle := 2 * math.Pi * phase
	state := routeState{
		Latitude: track.Yard.Latitude + yardLoopRadiusMeters*math.Cos(
			angle,
		)/metersPerDegreeLatitude,
		Heading:  normalizeHeading(degrees(angle) + 90),
		SpeedMPS: yardLoopSpeedMPS,
	}
	state.Longitude = track.Yard.Longitude
	if perDegree := metersPerDegreeLongitude(state.Latitude); perDegree > 0 {
		state.Longitude += yardLoopRadiusMeters * math.Sin(angle) / perDegree
	}
	return state
}

func centralTime() *time.Location {
	centralZoneOnce.Do(func() {
		location, err := time.LoadLocation("America/Chicago")
		if err != nil {
			location = time.FixedZone("CST", -6*60*60)
		}
		centralZone = location
	})
	return centralZone
}
