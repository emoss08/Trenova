package sim

import "math"

const (
	fieldGeofence      = "geofence"
	fieldCircle        = "circle"
	fieldPolygon       = "polygon"
	fieldVertices      = "vertices"
	fieldRadiusMeters  = "radiusMeters"
	fieldSettings      = "settings"
	fieldShowAddresses = "showAddresses"
)

type geofenceVertex struct {
	Latitude  float64
	Longitude float64
}

func (c *geofenceCircle) contains(latitude, longitude float64) bool {
	if len(c.Vertices) >= 3 {
		return polygonContains(c.Vertices, latitude, longitude)
	}
	return haversineMeters(latitude, longitude, c.Latitude, c.Longitude) <= c.RadiusMeters
}

func polygonContains(vertices []geofenceVertex, latitude, longitude float64) bool {
	inside := false
	for current, previous := 0, len(vertices)-1; current < len(vertices); previous, current = current, current+1 {
		a := vertices[current]
		b := vertices[previous]
		if (a.Latitude > latitude) == (b.Latitude > latitude) {
			continue
		}
		crossing := (b.Longitude-a.Longitude)*(latitude-a.Latitude)/(b.Latitude-a.Latitude) +
			a.Longitude
		if longitude < crossing {
			inside = !inside
		}
	}
	return inside
}

func polygonCentroid(vertices []geofenceVertex) geofenceVertex {
	if len(vertices) == 0 {
		return geofenceVertex{}
	}
	var area, latitude, longitude float64
	for current, previous := 0, len(vertices)-1; current < len(vertices); previous, current = current, current+1 {
		a := vertices[previous]
		b := vertices[current]
		cross := a.Longitude*b.Latitude - b.Longitude*a.Latitude
		area += cross
		longitude += (a.Longitude + b.Longitude) * cross
		latitude += (a.Latitude + b.Latitude) * cross
	}
	if math.Abs(area) < 1e-12 {
		var sumLatitude, sumLongitude float64
		for _, vertex := range vertices {
			sumLatitude += vertex.Latitude
			sumLongitude += vertex.Longitude
		}
		count := float64(len(vertices))
		return geofenceVertex{Latitude: sumLatitude / count, Longitude: sumLongitude / count}
	}
	return geofenceVertex{Latitude: latitude / (3 * area), Longitude: longitude / (3 * area)}
}

func polygonBoundingRadius(vertices []geofenceVertex, center geofenceVertex) float64 {
	radius := 0.0
	for _, vertex := range vertices {
		radius = math.Max(
			radius,
			haversineMeters(center.Latitude, center.Longitude, vertex.Latitude, vertex.Longitude),
		)
	}
	return radius
}

func geofenceVerticesOf(raw any) []geofenceVertex {
	items := listOf(raw)
	out := make([]geofenceVertex, 0, len(items))
	for _, item := range items {
		vertex, ok := anyAsMap(item)
		if !ok {
			continue
		}
		out = append(out, geofenceVertex{
			Latitude:  floatFromAny(vertex[keyLatitude]),
			Longitude: floatFromAny(vertex[keyLongitude]),
		})
	}
	return out
}

func geofenceShapeFromAddress(address Record) (geofenceCircle, bool) {
	geofence, ok := anyAsMap(address[fieldGeofence])
	if !ok {
		return geofenceCircle{}, false
	}
	shape := geofenceCircle{
		AddressID:        recordID(address),
		Name:             stringValue(address, keyName),
		FormattedAddress: stringValue(address, "formattedAddress"),
		ExternalIDs:      map[string]any{},
		Geofence:         cloneMap(geofence),
	}
	if rawExternalIDs, okIDs := anyAsMap(address[fieldExternalIDs]); okIDs {
		shape.ExternalIDs = cloneMap(rawExternalIDs)
	}
	if polygon, isPolygon := anyAsMap(geofence[fieldPolygon]); isPolygon {
		vertices := geofenceVerticesOf(polygon[fieldVertices])
		if len(vertices) < 3 {
			return geofenceCircle{}, false
		}
		for _, vertex := range vertices {
			if !isReasonableCoordinate(vertex.Latitude, vertex.Longitude) {
				return geofenceCircle{}, false
			}
		}
		center := polygonCentroid(vertices)
		shape.Vertices = vertices
		shape.Latitude = center.Latitude
		shape.Longitude = center.Longitude
		shape.RadiusMeters = polygonBoundingRadius(vertices, center)
		return shape, true
	}
	circle, ok := anyAsMap(geofence[fieldCircle])
	if !ok {
		return geofenceCircle{}, false
	}
	shape.Latitude = floatFromAny(circle[keyLatitude])
	shape.Longitude = floatFromAny(circle[keyLongitude])
	shape.RadiusMeters = floatFromAny(circle[fieldRadiusMeters])
	if shape.RadiusMeters <= 0 || !isReasonableCoordinate(shape.Latitude, shape.Longitude) {
		return geofenceCircle{}, false
	}
	return shape, true
}
