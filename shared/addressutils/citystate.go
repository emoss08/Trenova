package addressutils

import (
	"regexp"
	"strings"
)

type CityState struct {
	City  string
	State string
}

var (
	postalSuffix = regexp.MustCompile(`\s+\d{5}(?:-\d{4})?$`)
	spaces       = regexp.MustCompile(`\s+`)
)

var stateCodes = map[string]string{
	"alabama": "AL", "alaska": "AK", "arizona": "AZ", "arkansas": "AR", "california": "CA",
	"colorado": "CO", "connecticut": "CT", "delaware": "DE", "district of columbia": "DC",
	"florida": "FL", "georgia": "GA", "hawaii": "HI", "idaho": "ID", "illinois": "IL",
	"indiana": "IN", "iowa": "IA", "kansas": "KS", "kentucky": "KY", "louisiana": "LA",
	"maine": "ME", "maryland": "MD", "massachusetts": "MA", "michigan": "MI", "minnesota": "MN",
	"mississippi": "MS", "missouri": "MO", "montana": "MT", "nebraska": "NE", "nevada": "NV",
	"new hampshire": "NH", "new jersey": "NJ", "new mexico": "NM", "new york": "NY",
	"north carolina": "NC", "north dakota": "ND", "ohio": "OH", "oklahoma": "OK", "oregon": "OR",
	"pennsylvania": "PA", "rhode island": "RI", "south carolina": "SC", "south dakota": "SD",
	"tennessee": "TN", "texas": "TX", "utah": "UT", "vermont": "VT", "virginia": "VA",
	"washington": "WA", "west virginia": "WV", "wisconsin": "WI", "wyoming": "WY",
	"puerto rico": "PR", "alberta": "AB", "british columbia": "BC", "manitoba": "MB",
	"new brunswick": "NB", "newfoundland and labrador": "NL", "nova scotia": "NS",
	"ontario": "ON", "prince edward island": "PE", "quebec": "QC", "saskatchewan": "SK",
}

var knownCodes = buildKnownCodes()

func buildKnownCodes() map[string]struct{} {
	codes := make(map[string]struct{}, len(stateCodes))
	for _, code := range stateCodes {
		codes[code] = struct{}{}
	}

	return codes
}

func StateCode(raw string) (string, bool) {
	normalized := strings.ToLower(spaces.ReplaceAllString(strings.TrimSpace(raw), " "))
	normalized = strings.TrimSuffix(normalized, ".")
	if normalized == "" {
		return "", false
	}
	if code, ok := stateCodes[normalized]; ok {
		return code, true
	}

	upper := strings.ToUpper(strings.ReplaceAll(normalized, ".", ""))
	if _, ok := knownCodes[upper]; ok {
		return upper, true
	}

	return "", false
}

func ParseCityState(raw string) (CityState, bool) {
	text := strings.TrimSpace(spaces.ReplaceAllString(raw, " "))
	text = strings.TrimSpace(postalSuffix.ReplaceAllString(text, ""))
	text = strings.TrimRight(text, ", ")
	if text == "" {
		return CityState{}, false
	}

	if comma := strings.LastIndex(text, ","); comma > 0 {
		city := strings.TrimSpace(text[:comma])
		if state, ok := StateCode(text[comma+1:]); ok && city != "" {
			return CityState{City: city, State: state}, true
		}
	}

	words := strings.Fields(text)
	for take := min(3, len(words)-1); take >= 1; take-- {
		state, ok := StateCode(strings.Join(words[len(words)-take:], " "))
		if !ok {
			continue
		}
		city := strings.TrimRight(strings.Join(words[:len(words)-take], " "), ",")
		if city != "" {
			return CityState{City: city, State: state}, true
		}
	}

	return CityState{}, false
}

func (c CityState) SameAs(other CityState) bool {
	return c.State == other.State && strings.EqualFold(c.City, other.City)
}
