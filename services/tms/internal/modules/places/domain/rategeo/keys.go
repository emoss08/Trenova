package rategeo

import (
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
)

const (
	prefixLocation  = "LOC:"
	prefixZip5      = "Z5:"
	prefixZip3      = "Z3:"
	prefixCityState = "CS:"
	prefixZone      = "ZN:"
	prefixState     = "ST:"
	prefixCountry   = "CT:"

	// KeyAny is the key a rule stores when it is willing to match any place. It
	// carries no prefix because there is nothing to qualify.
	KeyAny = "ANY"

	// LaneKeySeparator joins an origin key to a destination key. It is a
	// character that cannot appear in any encoded value, so a lane key can
	// always be split back into its two halves.
	LaneKeySeparator = ">"

	// cityStateSeparator joins a state to a city inside a CityState key. The
	// state leads so that keys for one state sort together.
	cityStateSeparator = "|"

	zip3Length = 3
)

// Scope is one side of a lane: a way of naming a place.
//
// Value carries whatever the type needs — a location id, a postal code, a zone
// id, a state id, a country code — and is empty for the two types that do not
// name anything.
type Scope struct {
	Type  ScopeType
	Value string
	// City is only read for CityState scopes, where Value holds the state id.
	City string
}

// Key renders the scope as the string a rule stores and a shipment is matched
// against. A radius scope has no key: it is found through the geospatial index
// instead, and callers must handle it separately.
func (s Scope) Key() (string, bool) {
	switch s.Type {
	case ScopeTypeAny:
		return KeyAny, true
	case ScopeTypeLocation:
		return prefixLocation + s.Value, s.Value != ""
	case ScopeTypeZip5:
		return prefixZip5 + normalizePostalCode(s.Value), s.Value != ""
	case ScopeTypeZip3:
		return prefixZip3 + zip3Of(s.Value), len(normalizePostalCode(s.Value)) >= zip3Length
	case ScopeTypeCityState:
		return CityStateKey(s.Value, s.City), s.Value != "" && s.City != ""
	case ScopeTypeZone:
		return prefixZone + s.Value, s.Value != ""
	case ScopeTypeState:
		return prefixState + s.Value, s.Value != ""
	case ScopeTypeCountry:
		return prefixCountry + normalizeCountry(s.Value), s.Value != ""
	case ScopeTypeRadius:
		return "", false
	default:
		return "", false
	}
}

// CityStateKey renders a city within a state. City names arrive from address
// entry and from imported rate sheets, so they are folded to a single case
// before they become part of a key — "Saint Louis" typed three different ways
// has to reach the same rate.
func CityStateKey(stateID, city string) string {
	return prefixCityState + stateID + cityStateSeparator + normalizeCity(city)
}

// Place is everything the engine knows about one end of a shipment's lane.
//
// It is assembled once per rating from a stop's location and handed to
// MatchKeys, which turns it into the set of keys any rule could have stored to
// match it.
type Place struct {
	LocationID  pulid.ID
	PostalCode  string
	City        string
	StateID     pulid.ID
	CountryISO3 string
	ZoneIDs     []pulid.ID
}

// MatchKeys returns every key that describes this place, from the most specific
// downward, always ending with KeyAny.
//
// This is the other half of the lane key scheme: a rule stores exactly one key,
// and a shipment produces the handful of keys that a rule could have stored to
// reach it. Matching is then a set membership test the database can answer from
// an index, rather than a disjunction of comparisons per scope type.
//
// The result is ordered most specific first. Nothing depends on that ordering —
// the winner is chosen by stored specificity, not by position — but it keeps
// the keys readable in a trace.
func (p *Place) MatchKeys() []string {
	keys := make([]string, 0, 6+len(p.ZoneIDs))

	if !p.LocationID.IsNil() {
		keys = append(keys, prefixLocation+p.LocationID.String())
	}

	postal := normalizePostalCode(p.PostalCode)
	if postal != "" {
		keys = append(keys, prefixZip5+postal)
		if len(postal) >= zip3Length {
			keys = append(keys, prefixZip3+postal[:zip3Length])
		}
	}

	if !p.StateID.IsNil() && p.City != "" {
		keys = append(keys, CityStateKey(p.StateID.String(), p.City))
	}

	for _, zoneID := range p.ZoneIDs {
		if !zoneID.IsNil() {
			keys = append(keys, prefixZone+zoneID.String())
		}
	}

	if !p.StateID.IsNil() {
		keys = append(keys, prefixState+p.StateID.String())
	}

	if p.CountryISO3 != "" {
		keys = append(keys, prefixCountry+normalizeCountry(p.CountryISO3))
	}

	return append(keys, KeyAny)
}

// LaneKey joins an origin key to a destination key.
func LaneKey(originKey, destinationKey string) string {
	return originKey + LaneKeySeparator + destinationKey
}

// LaneKeyCandidates returns every lane key that could match this origin and
// destination — the cross product of the two key sets.
//
// The result is bounded by the number of scope types plus the number of zones a
// place belongs to, squared, which in practice is a few dozen entries. That is
// small enough to hand to the database as a single array and let it probe an
// index once per entry.
func LaneKeyCandidates(originKeys, destinationKeys []string) []string {
	candidates := make([]string, 0, len(originKeys)*len(destinationKeys))

	for _, originKey := range originKeys {
		for _, destinationKey := range destinationKeys {
			candidates = append(candidates, LaneKey(originKey, destinationKey))
		}
	}

	return candidates
}

// LaneSpecificity scores a lane by its two scopes. Geography outweighs every
// other condition a rule can carry, so this value is scaled well clear of the
// condition weights it is summed with.
func LaneSpecificity(origin, destination ScopeType) int32 {
	return origin.Weight() + destination.Weight()
}

func normalizePostalCode(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	// Postal codes reach us as "60601", "60601-1234" and occasionally with
	// stray spacing. Only the leading segment identifies the delivery area, and
	// it is the only part a rate is ever keyed on.
	if dash := strings.IndexByte(trimmed, '-'); dash > 0 {
		trimmed = trimmed[:dash]
	}

	return strings.ToUpper(trimmed)
}

func zip3Of(value string) string {
	postal := normalizePostalCode(value)
	if len(postal) < zip3Length {
		return postal
	}

	return postal[:zip3Length]
}

func normalizeCity(value string) string {
	return strings.ToUpper(strings.Join(strings.Fields(value), " "))
}

func normalizeCountry(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
