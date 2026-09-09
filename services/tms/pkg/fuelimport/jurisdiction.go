package fuelimport

import "strings"

const (
	countryUS = "US"
	countryCA = "CA"
	countryMX = "MX"
)

var usJurisdictions = map[string]string{
	"AL": "alabama", "AK": "alaska", "AZ": "arizona", "AR": "arkansas", "CA": "california",
	"CO": "colorado", "CT": "connecticut", "DE": "delaware", "DC": "district of columbia",
	"FL": "florida", "GA": "georgia", "HI": "hawaii", "ID": "idaho", "IL": "illinois",
	"IN": "indiana", "IA": "iowa", "KS": "kansas", "KY": "kentucky", "LA": "louisiana",
	"ME": "maine", "MD": "maryland", "MA": "massachusetts", "MI": "michigan",
	"MN": "minnesota", "MS": "mississippi", "MO": "missouri", "MT": "montana",
	"NE": "nebraska", "NV": "nevada", "NH": "new hampshire", "NJ": "new jersey",
	"NM": "new mexico", "NY": "new york", "NC": "north carolina", "ND": "north dakota",
	"OH": "ohio", "OK": "oklahoma", "OR": "oregon", "PA": "pennsylvania",
	"RI": "rhode island", "SC": "south carolina", "SD": "south dakota", "TN": "tennessee",
	"TX": "texas", "UT": "utah", "VT": "vermont", "VA": "virginia", "WA": "washington",
	"WV": "west virginia", "WI": "wisconsin", "WY": "wyoming",
}

var caJurisdictions = map[string]string{
	"AB": "alberta", "BC": "british columbia", "MB": "manitoba", "NB": "new brunswick",
	"NL": "newfoundland and labrador", "NS": "nova scotia", "NT": "northwest territories",
	"NU": "nunavut", "ON": "ontario", "PE": "prince edward island", "QC": "quebec",
	"SK": "saskatchewan", "YT": "yukon",
}

var jurisdictionAliases = map[string]string{
	"washington dc":         "DC",
	"washington d c":        "DC",
	"district columbia":     "DC",
	"newfoundland":          "NL",
	"pei":                   "PE",
	"quebec province":       "QC",
	"québec":                "QC",
	"qu bec":                "QC",
	"yukon territory":       "YT",
	"nwt":                   "NT",
	"prince edward isl":     "PE",
	"n carolina":            "NC",
	"s carolina":            "SC",
	"n dakota":              "ND",
	"s dakota":              "SD",
	"w virginia":            "WV",
	"british col":           "BC",
	"newfoundland labrador": "NL",
}

var jurisdictionNames = buildJurisdictionNames()

func buildJurisdictionNames() map[string]string {
	names := make(
		map[string]string,
		len(usJurisdictions)+len(caJurisdictions)+len(jurisdictionAliases),
	)
	for code, name := range usJurisdictions {
		names[name] = code
	}
	for code, name := range caJurisdictions {
		names[name] = code
	}
	for name, code := range jurisdictionAliases {
		names[normalizeHeader(name)] = code
	}
	return names
}

func countryOf(code string) (string, bool) {
	if _, ok := usJurisdictions[code]; ok {
		return countryUS, true
	}
	if _, ok := caJurisdictions[code]; ok {
		return countryCA, true
	}
	return "", false
}

func NormalizeJurisdiction(raw string) (country, code string, ok bool) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return "", "", false
	}

	explicit := ""
	for _, prefix := range [...]string{"US", "CA", "MX", "USA", "CAN", "MEX"} {
		for _, sep := range [...]string{"-", "_", " ", "/", ":"} {
			if strings.HasPrefix(value, prefix+sep) && len(value) > len(prefix)+len(sep) {
				explicit = prefix[:2]
				value = strings.TrimSpace(value[len(prefix)+len(sep):])
				break
			}
		}
		if explicit != "" {
			break
		}
	}

	if len(value) == 2 && isLetters(value) {
		if explicit == countryMX {
			return countryMX, value, true
		}
		if explicit != "" {
			if found, known := countryOf(value); known && found == explicit {
				return found, value, true
			}
			return "", "", false
		}
		if found, known := countryOf(value); known {
			return found, value, true
		}
		return "", "", false
	}

	if mapped, found := jurisdictionNames[normalizeHeader(value)]; found {
		country, _ = countryOf(mapped)
		if explicit != "" && explicit != country {
			return "", "", false
		}
		return country, mapped, true
	}

	return "", "", false
}

func isLetters(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 'A' || value[i] > 'Z' {
			return false
		}
	}
	return len(value) > 0
}
