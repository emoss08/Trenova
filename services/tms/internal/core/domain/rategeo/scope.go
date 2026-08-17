// Package rategeo owns the geography vocabulary the rating engine matches on.
//
// A scope is one way of naming a place — a specific location, a postal code, a
// three digit postal prefix, a city, a named zone, a state, a country, or
// anywhere at all. Rate zones are built out of scopes, and every rate rule
// names one scope for its origin and one for its destination.
//
// The package is deliberately free of I/O. Everything here is a pure function
// of values the caller already holds, which is what lets the same code compute
// a rule's stored key at write time and a shipment's candidate keys at rating
// time, with no chance of the two drifting apart.
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

// Scope weights decide which of two matching rules is the more specific, and
// they are fixed constants rather than anything derived from the shape of the
// data. A weight computed from, say, the land area a zone covers would let
// yesterday's rate win today's shipment because somebody added a postal code to
// a zone overnight. Rating has to be reproducible, so the ordering is stated
// once, here, and never inferred.
//
// Radius sits between city and zone on purpose: a fifty mile circle is a
// tighter promise than a market area but a looser one than a named city.
const (
	WeightAny       = int32(0)
	WeightCountry   = int32(300)
	WeightState     = int32(500)
	WeightZone      = int32(600)
	WeightRadius    = int32(650)
	WeightCityState = int32(700)
	WeightZip3      = int32(800)
	WeightZip5      = int32(900)
	WeightLocation  = int32(1000)
)

func (st ScopeType) Weight() int32 {
	switch st {
	case ScopeTypeLocation:
		return WeightLocation
	case ScopeTypeZip5:
		return WeightZip5
	case ScopeTypeZip3:
		return WeightZip3
	case ScopeTypeCityState:
		return WeightCityState
	case ScopeTypeRadius:
		return WeightRadius
	case ScopeTypeZone:
		return WeightZone
	case ScopeTypeState:
		return WeightState
	case ScopeTypeCountry:
		return WeightCountry
	case ScopeTypeAny:
		return WeightAny
	default:
		return WeightAny
	}
}

// RequiresValue reports whether a scope of this type is meaningless without an
// accompanying value. Any matches everywhere and Radius is described by a
// centre point and a distance rather than by a name, so neither carries one.
func (st ScopeType) RequiresValue() bool {
	switch st {
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
