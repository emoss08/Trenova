package sim

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

var fleetSnapshotResources = []Resource{
	ResourceAddresses,
	ResourceAssets,
	ResourceDrivers,
	ResourceRoutes,
	ResourceVehicleStats,
	ResourceHOSClocks,
	ResourceHOSLogs,
	ResourceTags,
	ResourceDriverVehicleAssignments,
	ResourceDriverSignOuts,
	ResourceDriverWorkflows,
}

type fleetSnapshot struct {
	version        uint64
	locationVer    uint64
	addresses      []Record
	addressByID    map[string]Record
	assets         []Record
	assetByID      map[string]Record
	drivers        []Record
	driverByID     map[string]Record
	routes         []Record
	tagRecords     []Record
	tags           *tagIndex
	waypoints      map[string][]routePoint
	geometries     map[string]*routeGeometry
	templates      map[string]Record
	clockTemplates map[string]Record
	logTemplates   map[string]Record
	geofences      []geofenceCircle
	baseRoster     map[string]driverRoster
	assignments    []apiAssignment
	signOuts       []driverSignOut
	workflows      []Record
	vehicleIDs     []string
}

type fleetCache struct {
	mu      sync.Mutex
	current atomic.Pointer[fleetSnapshot]
}

func (l *LiveSimulator) SetClock(clock *SimClock) {
	l.clock = clock
}

func (l *LiveSimulator) now() time.Time {
	if l.clock == nil {
		return time.Now().UTC()
	}
	return l.clock.Now()
}

func (l *LiveSimulator) fleet() *fleetSnapshot {
	version := l.store.DataVersion()
	if snapshot := l.cache.current.Load(); snapshot != nil && snapshot.version == version {
		return snapshot
	}
	l.cache.mu.Lock()
	defer l.cache.mu.Unlock()
	if snapshot := l.cache.current.Load(); snapshot != nil &&
		snapshot.version == l.store.DataVersion() {
		return snapshot
	}
	snapshot := l.buildFleetSnapshot()
	l.cache.current.Store(snapshot)
	return snapshot
}

func (l *LiveSimulator) buildFleetSnapshot() *fleetSnapshot {
	previous := l.cache.current.Load()
	knownLocation := uint64(0)
	if previous != nil {
		knownLocation = previous.locationVer
	}
	read := l.store.Collections(fleetSnapshotResources, knownLocation, previous != nil)
	collections := read.Records
	snapshot := &fleetSnapshot{
		version:        read.Version,
		locationVer:    read.LocationVersion,
		addresses:      collections[ResourceAddresses],
		addressByID:    indexRecordsByID(collections[ResourceAddresses]),
		assets:         collections[ResourceAssets],
		assetByID:      indexRecordsByID(collections[ResourceAssets]),
		drivers:        collections[ResourceDrivers],
		driverByID:     indexRecordsByID(collections[ResourceDrivers]),
		routes:         collections[ResourceRoutes],
		tagRecords:     collections[ResourceTags],
		tags:           newTagIndex(collections[ResourceTags]),
		templates:      indexRecordsByID(collections[ResourceVehicleStats]),
		clockTemplates: indexRecordsByDriver(collections[ResourceHOSClocks]),
		logTemplates:   indexRecordsByDriver(collections[ResourceHOSLogs]),
		geofences:      geofenceCirclesFromAddresses(collections[ResourceAddresses]),
		assignments: mergeAssignments(
			parseAPIAssignments(collections[ResourceDriverVehicleAssignments]),
			staticAssignmentsFromDrivers(collections[ResourceDrivers]),
		),
		signOuts:  parseSignOuts(collections[ResourceDriverSignOuts]),
		workflows: collections[ResourceDriverWorkflows],
	}
	if read.Waypoints == nil && previous != nil {
		snapshot.waypoints = previous.waypoints
		snapshot.geometries = previous.geometries
	} else {
		snapshot.waypoints = read.Waypoints
		snapshot.geometries = make(map[string]*routeGeometry, len(snapshot.waypoints))
		for assetID, points := range snapshot.waypoints {
			geometry := l.buildRouteGeometry(assetID, points)
			snapshot.geometries[assetID] = &geometry
		}
	}
	snapshot.vehicleIDs = snapshot.knownVehicleIDs()
	snapshot.baseRoster = buildBaseRoster(snapshot)
	return snapshot
}

func indexRecordsByID(records []Record) map[string]Record {
	out := make(map[string]Record, len(records))
	for _, record := range records {
		if id := recordID(record); id != "" {
			out[id] = record
		}
	}
	return out
}

func indexRecordsByDriver(records []Record) map[string]Record {
	out := make(map[string]Record, len(records))
	for _, record := range records {
		if driverID := nestedString(record, keyDriver, keyID); driverID != "" {
			out[driverID] = record
		}
	}
	return out
}

func parseWaypoints(records []Record) map[string][]routePoint {
	out := map[string][]routePoint{}
	for _, record := range records {
		assetID := nestedString(record, keyAsset, keyID)
		if assetID == "" {
			continue
		}
		location, ok := anyAsMap(record[keyLocation])
		if !ok {
			continue
		}
		point := routePoint{
			Latitude:  floatFromAny(location[keyLatitude]),
			Longitude: floatFromAny(location[keyLongitude]),
			Heading:   floatFromAny(location[keyHeadingDegrees]),
			SpeedMPS:  defaultAssetSpeedMPS,
		}
		if address, okAddress := anyAsMap(location["address"]); okAddress {
			point.Address = cloneMap(address)
		}
		if speed, okSpeed := anyAsMap(record[keySpeed]); okSpeed {
			point.SpeedMPS = maxFloat64(
				floatFromAny(speed["gpsSpeedMetersPerSecond"]),
				floatFromAny(speed["ecuSpeedMetersPerSecond"]),
			)
		}
		if point.SpeedMPS <= 0 {
			point.SpeedMPS = defaultAssetSpeedMPS
		}
		out[assetID] = append(out[assetID], point)
	}
	return out
}

func (f *fleetSnapshot) knownVehicleIDs() []string {
	candidates := make([]string, 0, len(f.assets)+len(f.waypoints)+len(f.templates))
	for _, record := range f.assets {
		if strings.EqualFold(stringValue(record, keyType), assetTypeVehicle) {
			candidates = append(candidates, recordID(record))
		}
	}
	for id := range f.waypoints {
		candidates = append(candidates, id)
	}
	for id := range f.templates {
		candidates = append(candidates, id)
	}
	sort.Strings(candidates)
	return sliceutils.DedupeStrings(candidates)
}

func buildBaseRoster(snapshot *fleetSnapshot) map[string]driverRoster {
	roster := make(map[string]driverRoster, len(snapshot.drivers))
	for _, driver := range snapshot.drivers {
		if id := recordID(driver); id != "" {
			roster[id] = driverRoster{Name: stringValue(driver, keyName)}
		}
	}
	for driverID, record := range snapshot.clockTemplates {
		entry := roster[driverID]
		if entry.Name == "" {
			entry.Name = nestedString(record, keyDriver, keyName)
		}
		entry.VehicleID = stringutils.FirstNonEmptyTrimmed(
			entry.VehicleID,
			nestedString(record, "currentVehicle", keyID),
		)
		roster[driverID] = entry
	}
	for _, route := range snapshot.routes {
		driverID := nestedString(route, keyDriver, keyID)
		if driverID == "" {
			continue
		}
		entry := roster[driverID]
		entry.VehicleID = stringutils.FirstNonEmptyTrimmed(
			entry.VehicleID,
			nestedString(route, keyVehicle, keyID),
		)
		entry.Name = stringutils.FirstNonEmptyTrimmed(
			entry.Name,
			nestedString(route, keyDriver, keyName),
		)
		roster[driverID] = entry
	}
	for driverID, entry := range roster {
		if strings.TrimSpace(entry.Name) == "" {
			entry.Name = driverID
			roster[driverID] = entry
		}
	}
	assignVehiclesToUnassignedDrivers(roster, snapshot.vehicleIDs)
	return roster
}

func assignVehiclesToUnassignedDrivers(roster map[string]driverRoster, vehicleIDs []string) {
	if len(roster) == 0 {
		return
	}
	assigned := make(map[string]struct{}, len(roster))
	for _, entry := range roster {
		if vehicleID := strings.TrimSpace(entry.VehicleID); vehicleID != "" {
			assigned[vehicleID] = struct{}{}
		}
	}
	candidates := make([]string, 0, len(vehicleIDs))
	for _, vehicleID := range vehicleIDs {
		if _, taken := assigned[vehicleID]; !taken {
			candidates = append(candidates, vehicleID)
		}
	}
	if len(candidates) == 0 {
		return
	}
	driverIDs := make([]string, 0, len(roster))
	for driverID, entry := range roster {
		if strings.TrimSpace(entry.VehicleID) == "" {
			driverIDs = append(driverIDs, driverID)
		}
	}
	sort.Strings(driverIDs)
	for idx, driverID := range driverIDs {
		if idx >= len(candidates) {
			break
		}
		entry := roster[driverID]
		entry.VehicleID = candidates[idx]
		roster[driverID] = entry
	}
}

func (l *LiveSimulator) loadAssetWaypoints() map[string][]routePoint {
	return l.fleet().waypoints
}

func (l *LiveSimulator) loadAssetMetadata() map[string]Record {
	return l.fleet().assetByID
}

func (l *LiveSimulator) loadVehicleStatsTemplates() map[string]Record {
	return l.fleet().templates
}

func (l *LiveSimulator) loadDriverTemplateRecords(resource Resource) map[string]Record {
	if resource == ResourceHOSClocks {
		return l.fleet().clockTemplates
	}
	if resource == ResourceHOSLogs {
		return l.fleet().logTemplates
	}
	return map[string]Record{}
}

func (l *LiveSimulator) loadGeofenceCircles() []geofenceCircle {
	return l.fleet().geofences
}

func (l *LiveSimulator) loadDriverRoster() map[string]driverRoster {
	snapshot := l.fleet()
	roster := make(map[string]driverRoster, len(snapshot.baseRoster))
	for driverID, entry := range snapshot.baseRoster {
		roster[driverID] = entry
	}
	active := snapshot.activeAPIAssignments(l.now())
	for idx := range active {
		assignment := &active[idx]
		entry, ok := roster[assignment.DriverID]
		if !ok {
			continue
		}
		entry.VehicleID = assignment.VehicleID
		roster[assignment.DriverID] = entry
	}
	return roster
}

func (l *LiveSimulator) driverByVehicleMap() map[string]string {
	snapshot := l.fleet()
	driverIDs := make([]string, 0, len(snapshot.baseRoster))
	for driverID := range snapshot.baseRoster {
		driverIDs = append(driverIDs, driverID)
	}
	sort.Strings(driverIDs)
	out := make(map[string]string, len(driverIDs))
	for _, driverID := range driverIDs {
		if driver, ok := snapshot.driverByID[driverID]; ok && driverDeactivated(driver) {
			continue
		}
		vehicleID := strings.TrimSpace(snapshot.baseRoster[driverID].VehicleID)
		if vehicleID == "" {
			continue
		}
		if _, exists := out[vehicleID]; !exists {
			out[vehicleID] = driverID
		}
	}
	active := snapshot.activeAPIAssignments(l.now())
	for idx := range active {
		assignment := &active[idx]
		for vehicleID, driverID := range out {
			if driverID == assignment.DriverID && vehicleID != assignment.VehicleID {
				delete(out, vehicleID)
			}
		}
		out[assignment.VehicleID] = assignment.DriverID
	}
	return out
}

func (l *LiveSimulator) buildRouteGeometry(assetID string, points []routePoint) routeGeometry {
	geometry := routeGeometry{Points: points}
	if len(points) < 2 {
		return geometry
	}
	segments, period := buildRouteSegments(points)
	if len(segments) == 0 || period <= 0 {
		return geometry
	}
	geometry.Segments, geometry.Period = l.tuneRouteSegmentsForDuration(assetID, segments, period)
	return geometry
}

func (l *LiveSimulator) routeGeometryForAsset(assetID string, points []routePoint) routeGeometry {
	if cached, ok := l.fleet().geometries[assetID]; ok && samePoints(cached.Points, points) {
		return *cached
	}
	return l.buildRouteGeometry(assetID, points)
}

func samePoints(left, right []routePoint) bool {
	if len(left) != len(right) {
		return false
	}
	return len(left) == 0 || &left[0] == &right[0]
}

func geofenceCirclesFromAddresses(addresses []Record) []geofenceCircle {
	out := make([]geofenceCircle, 0, len(addresses))
	for _, address := range addresses {
		if shape, ok := geofenceShapeFromAddress(address); ok {
			out = append(out, shape)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].AddressID < out[j].AddressID
	})
	return out
}

func (f *fleetSnapshot) geofenceAt(latitude, longitude float64) *geofenceCircle {
	for idx := range f.geofences {
		circle := &f.geofences[idx]
		if circle.contains(latitude, longitude) {
			return circle
		}
	}
	return nil
}
