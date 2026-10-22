// Package rategeo owns the geography vocabulary the rating engine matches on.
//
// A scope is one way of naming a place — a specific location, a postal code, a
// three digit postal prefix, a city, a named zone, a state, a country, or
// anywhere at all. Rate zones are built out of scopes, and every rate rule
// names one scope for its origin and one for its destination.
//
// The keys a scope or a place produces, and how specific a lane is, are
// computed by the rules module (lane.keys, lane.specificity), the one
// implementation the server and the browser share.
package rategeo

// ScopeType names the granularity at which a place is described.
type ScopeType string

const (
	ScopeTypeAny       = ScopeType("Any")
	ScopeTypeCountry   = ScopeType("Country")
	ScopeTypeState     = ScopeType("State")
	ScopeTypeZone      = ScopeType("Zone")
	ScopeTypeRadius    = ScopeType("Radius")
	ScopeTypeCityState = ScopeType("CityState")
	ScopeTypeZip3      = ScopeType("Zip3")
	ScopeTypeZip5      = ScopeType("Zip5")
	ScopeTypeLocation  = ScopeType("Location")
)

func (st ScopeType) String() string {
	return string(st)
}

func ScopeTypeValues() []ScopeType {
	return []ScopeType{
		ScopeTypeAny,
		ScopeTypeCountry,
		ScopeTypeState,
		ScopeTypeZone,
		ScopeTypeRadius,
		ScopeTypeCityState,
		ScopeTypeZip3,
		ScopeTypeZip5,
		ScopeTypeLocation,
	}
}

func (st ScopeType) IsValid() bool {
	switch st {
	case ScopeTypeAny,
		ScopeTypeCountry,
		ScopeTypeState,
		ScopeTypeZone,
		ScopeTypeRadius,
		ScopeTypeCityState,
		ScopeTypeZip3,
		ScopeTypeZip5,
		ScopeTypeLocation:
		return true
	default:
		return false
	}
}

// RequiresValue reports whether a scope of this type is meaningless without an
// accompanying value. Any matches everywhere and Radius is described by a
// centre point and a distance rather than by a name, so neither carries one.
func (st ScopeType) RequiresValue() bool {
	switch st { //nolint:exhaustive // every other scope names a place
	case ScopeTypeAny, ScopeTypeRadius:
		return false
	default:
		return true
	}
}

// IsKeyed reports whether a scope of this type participates in lane key
// matching. Radius rules are found through a geospatial index instead, so they
// are the one shape that cannot be reduced to a string.
func (st ScopeType) IsKeyed() bool {
	return st != ScopeTypeRadius
}
