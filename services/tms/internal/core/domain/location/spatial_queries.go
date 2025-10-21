package location

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// SpatialQueryHelper provides useful PostGIS spatial query functions for locations
type SpatialQueryHelper struct {
	db *bun.DB
}

// NewSpatialQueryHelper creates a new spatial query helper
func NewSpatialQueryHelper(db *bun.DB) *SpatialQueryHelper {
	return &SpatialQueryHelper{db: db}
}

// DistanceResult represents the result of a distance query
type DistanceResult struct {
	Location
	DistanceMeters float64 `bun:"distance_meters" json:"distanceMeters"`
	DistanceMiles  float64 `bun:"distance_miles"  json:"distanceMiles"`
	DistanceKM     float64 `bun:"distance_km"     json:"distanceKm"`
}

// FindLocationsWithinRadius finds all locations within a specified radius (in meters) of a point
// Example: Find all warehouses within 50 miles (80467 meters) of a pickup location
func (h *SpatialQueryHelper) FindLocationsWithinRadius(
	ctx context.Context,
	longitude, latitude float64,
	radiusMeters float64,
	orgID, businessUnitID string,
) ([]DistanceResult, error) {
	var results []DistanceResult

	err := h.db.NewSelect().
		Model(&results).
		ColumnExpr("loc.*").
		ColumnExpr("ST_Distance(loc.geom, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) AS distance_meters", longitude, latitude).
		ColumnExpr("ST_Distance(loc.geom, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) * 0.000621371 AS distance_miles", longitude, latitude).
		ColumnExpr("ST_Distance(loc.geom, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) / 1000 AS distance_km", longitude, latitude).
		Where("loc.organization_id = ?", orgID).
		Where("loc.business_unit_id = ?", businessUnitID).
		Where("loc.geom IS NOT NULL").
		Where("ST_DWithin(loc.geom, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)", longitude, latitude, radiusMeters).
		Order("distance_meters ASC").
		Scan(ctx)

	return results, err
}

// FindNearestLocations finds the N nearest locations to a given point
// Example: Find the 5 closest distribution centers to a delivery location
func (h *SpatialQueryHelper) FindNearestLocations(
	ctx context.Context,
	longitude, latitude float64,
	limit int,
	orgID, businessUnitID string,
) ([]DistanceResult, error) {
	var results []DistanceResult

	err := h.db.NewSelect().
		Model(&results).
		ColumnExpr("loc.*").
		ColumnExpr("ST_Distance(loc.geom, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) AS distance_meters", longitude, latitude).
		ColumnExpr("ST_Distance(loc.geom, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) * 0.000621371 AS distance_miles", longitude, latitude).
		ColumnExpr("ST_Distance(loc.geom, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) / 1000 AS distance_km", longitude, latitude).
		Where("loc.organization_id = ?", orgID).
		Where("loc.business_unit_id = ?", businessUnitID).
		Where("loc.geom IS NOT NULL").
		Order("distance_meters ASC").
		Limit(limit).
		Scan(ctx)

	return results, err
}

// CalculateDistance calculates the distance between two locations in meters, miles, and kilometers
// Example: Calculate the distance between a pickup and delivery location for routing
func (h *SpatialQueryHelper) CalculateDistance(
	ctx context.Context,
	locationID1, locationID2 string,
) (*struct {
	Location1ID    string  `json:"location1Id"`
	Location2ID    string  `json:"location2Id"`
	DistanceMeters float64 `json:"distanceMeters"`
	DistanceMiles  float64 `json:"distanceMiles"`
	DistanceKM     float64 `json:"distanceKm"`
}, error,
) {
	var result struct {
		Location1ID    string  `bun:"location1_id"`
		Location2ID    string  `bun:"location2_id"`
		DistanceMeters float64 `bun:"distance_meters"`
		DistanceMiles  float64 `bun:"distance_miles"`
		DistanceKM     float64 `bun:"distance_km"`
	}

	err := h.db.NewRaw(`
		SELECT 
			? AS location1_id,
			? AS location2_id,
			ST_Distance(l1.geom, l2.geom) AS distance_meters,
			ST_Distance(l1.geom, l2.geom) * 0.000621371 AS distance_miles,
			ST_Distance(l1.geom, l2.geom) / 1000 AS distance_km
		FROM locations l1
		CROSS JOIN locations l2
		WHERE l1.id = ? AND l2.id = ?
			AND l1.geom IS NOT NULL AND l2.geom IS NOT NULL
	`, locationID1, locationID2, locationID1, locationID2).Scan(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &struct {
		Location1ID    string  `json:"location1Id"`
		Location2ID    string  `json:"location2Id"`
		DistanceMeters float64 `json:"distanceMeters"`
		DistanceMiles  float64 `json:"distanceMiles"`
		DistanceKM     float64 `json:"distanceKm"`
	}{
		Location1ID:    result.Location1ID,
		Location2ID:    result.Location2ID,
		DistanceMeters: result.DistanceMeters,
		DistanceMiles:  result.DistanceMiles,
		DistanceKM:     result.DistanceKM,
	}, nil
}

// IsLocationWithinServiceArea checks if a location is within a service area (defined by a polygon)
// Example: Check if a pickup location is within your operational service area
func (h *SpatialQueryHelper) IsLocationWithinServiceArea(
	ctx context.Context,
	locationID string,
	serviceAreaWKT string, // Well-Known Text format for polygon
) (bool, error) {
	var result struct {
		IsWithin bool `bun:"is_within"`
	}

	err := h.db.NewRaw(`
		SELECT ST_Within(
			l.geom::geometry,
			ST_GeomFromText(?, 4326)
		) AS is_within
		FROM locations l
		WHERE l.id = ? AND l.geom IS NOT NULL
	`, serviceAreaWKT, locationID).Scan(ctx, &result)

	return result.IsWithin, err
}

// FindLocationsAlongRoute finds locations within a buffer distance of a route line
// Example: Find all fuel stations within 5 miles of a planned route
func (h *SpatialQueryHelper) FindLocationsAlongRoute(
	ctx context.Context,
	routeLineWKT string, // Well-Known Text format for line string (route)
	bufferMeters float64,
	orgID, businessUnitID string,
) ([]DistanceResult, error) {
	var results []DistanceResult

	err := h.db.NewSelect().
		Model(&results).
		ColumnExpr("loc.*").
		ColumnExpr("ST_Distance(loc.geom, ST_GeomFromText(?, 4326)::geography) AS distance_meters", routeLineWKT).
		ColumnExpr("ST_Distance(loc.geom, ST_GeomFromText(?, 4326)::geography) * 0.000621371 AS distance_miles", routeLineWKT).
		ColumnExpr("ST_Distance(loc.geom, ST_GeomFromText(?, 4326)::geography) / 1000 AS distance_km", routeLineWKT).
		Where("loc.organization_id = ?", orgID).
		Where("loc.business_unit_id = ?", businessUnitID).
		Where("loc.geom IS NOT NULL").
		Where("ST_DWithin(loc.geom, ST_GeomFromText(?, 4326)::geography, ?)", routeLineWKT, bufferMeters).
		Order("distance_meters ASC").
		Scan(ctx)

	return results, err
}

// GetLocationBounds returns the bounding box for a set of locations
// Example: Get the geographic bounds of all your warehouse locations for map display
func (h *SpatialQueryHelper) GetLocationBounds(
	ctx context.Context,
	orgID, businessUnitID string,
	categoryID *string, // Optional: filter by location category
) (*struct {
	MinLongitude float64 `json:"minLongitude"`
	MinLatitude  float64 `json:"minLatitude"`
	MaxLongitude float64 `json:"maxLongitude"`
	MaxLatitude  float64 `json:"maxLatitude"`
	CenterLon    float64 `json:"centerLongitude"`
	CenterLat    float64 `json:"centerLatitude"`
}, error,
) {
	var result struct {
		MinLongitude float64 `bun:"min_longitude"`
		MinLatitude  float64 `bun:"min_latitude"`
		MaxLongitude float64 `bun:"max_longitude"`
		MaxLatitude  float64 `bun:"max_latitude"`
		CenterLon    float64 `bun:"center_lon"`
		CenterLat    float64 `bun:"center_lat"`
	}

	query := h.db.NewSelect().
		Model((*Location)(nil)).
		ColumnExpr("ST_XMin(ST_Extent(geom::geometry)) AS min_longitude").
		ColumnExpr("ST_YMin(ST_Extent(geom::geometry)) AS min_latitude").
		ColumnExpr("ST_XMax(ST_Extent(geom::geometry)) AS max_longitude").
		ColumnExpr("ST_YMax(ST_Extent(geom::geometry)) AS max_latitude").
		ColumnExpr("ST_X(ST_Centroid(ST_Extent(geom::geometry))) AS center_lon").
		ColumnExpr("ST_Y(ST_Centroid(ST_Extent(geom::geometry))) AS center_lat").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", businessUnitID).
		Where("geom IS NOT NULL")

	if categoryID != nil {
		query = query.Where("location_category_id = ?", *categoryID)
	}

	err := query.Scan(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &struct {
		MinLongitude float64 `json:"minLongitude"`
		MinLatitude  float64 `json:"minLatitude"`
		MaxLongitude float64 `json:"maxLongitude"`
		MaxLatitude  float64 `json:"maxLatitude"`
		CenterLon    float64 `json:"centerLongitude"`
		CenterLat    float64 `json:"centerLatitude"`
	}{
		MinLongitude: result.MinLongitude,
		MinLatitude:  result.MinLatitude,
		MaxLongitude: result.MaxLongitude,
		MaxLatitude:  result.MaxLatitude,
		CenterLon:    result.CenterLon,
		CenterLat:    result.CenterLat,
	}, nil
}

// ClusterLocations returns clustered locations within a grid cell size
// Example: Cluster nearby locations for map markers at different zoom levels
func (h *SpatialQueryHelper) ClusterLocations(
	ctx context.Context,
	cellSizeMeters float64,
	orgID, businessUnitID string,
) ([]struct {
	ClusterID     string  `bun:"cluster_id" json:"clusterId"`
	CenterLon     float64 `bun:"center_lon" json:"centerLongitude"`
	CenterLat     float64 `bun:"center_lat" json:"centerLatitude"`
	LocationCount int     `bun:"location_count" json:"locationCount"`
	LocationIDs   string  `bun:"location_ids" json:"locationIds"` // Comma-separated list
}, error,
) {
	type clusterResult struct {
		ClusterID     string  `bun:"cluster_id"`
		CenterLon     float64 `bun:"center_lon"`
		CenterLat     float64 `bun:"center_lat"`
		LocationCount int     `bun:"location_count"`
		LocationIDs   string  `bun:"location_ids"`
	}

	var results []clusterResult

	err := h.db.NewRaw(`
		SELECT 
			ST_SnapToGrid(geom::geometry, ?)::text AS cluster_id,
			ST_X(ST_Centroid(ST_Collect(geom::geometry))) AS center_lon,
			ST_Y(ST_Centroid(ST_Collect(geom::geometry))) AS center_lat,
			COUNT(*) AS location_count,
			STRING_AGG(id, ',') AS location_ids
		FROM locations
		WHERE organization_id = ?
			AND business_unit_id = ?
			AND geom IS NOT NULL
		GROUP BY ST_SnapToGrid(geom::geometry, ?)
		ORDER BY location_count DESC
	`, cellSizeMeters, orgID, businessUnitID, cellSizeMeters).Scan(ctx, &results)
	if err != nil {
		return nil, err
	}

	// Convert to return type
	finalResults := make([]struct {
		ClusterID     string  `bun:"cluster_id" json:"clusterId"`
		CenterLon     float64 `bun:"center_lon" json:"centerLongitude"`
		CenterLat     float64 `bun:"center_lat" json:"centerLatitude"`
		LocationCount int     `bun:"location_count" json:"locationCount"`
		LocationIDs   string  `bun:"location_ids" json:"locationIds"`
	}, len(results))

	for i, r := range results {
		finalResults[i].ClusterID = r.ClusterID
		finalResults[i].CenterLon = r.CenterLon
		finalResults[i].CenterLat = r.CenterLat
		finalResults[i].LocationCount = r.LocationCount
		finalResults[i].LocationIDs = r.LocationIDs
	}

	return finalResults, nil
}

// Example usage comments for common TMS scenarios:

// Distance conversion constants
const (
	MetersToMiles      = 0.000621371
	MetersToKilometers = 0.001
	MilesToMeters      = 1609.34
	KilometersToMeters = 1000
)

// Example: Find warehouses within 50 miles of a pickup
func ExampleFindWarehousesNearPickup() string {
	return fmt.Sprintf(`
		// Find warehouses within 50 miles (%.0f meters) of pickup
		warehouses, err := helper.FindLocationsWithinRadius(
			ctx,
			pickupLongitude,
			pickupLatitude,
			%.0f, // 50 miles in meters
			organizationID,
			businessUnitID,
		)
	`, 50*MilesToMeters, 50*MilesToMeters)
}

// Example: Find 5 nearest distribution centers
func ExampleFindNearestDCs() string {
	return `
		// Find 5 nearest distribution centers
		dcs, err := helper.FindNearestLocations(
			ctx,
			deliveryLongitude,
			deliveryLatitude,
			5,
			organizationID,
			businessUnitID,
		)
	`
}

// Example: Calculate route distance for pricing
func ExampleCalculateRouteDistance() string {
	return `
		// Calculate distance between pickup and delivery for pricing
		distance, err := helper.CalculateDistance(
			ctx,
			pickupLocationID,
			deliveryLocationID,
		)
		// Use distance.DistanceMiles for rate calculation
	`
}
