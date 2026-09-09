package countryutils

import "strings"

const (
	ISO2UnitedStates = "US"
	ISO2Canada       = "CA"
	ISO2Mexico       = "MX"
)

var iso3ToISO2 = map[string]string{
	"USA": ISO2UnitedStates,
	"CAN": ISO2Canada,
	"MEX": ISO2Mexico,
}

func ISO3ToISO2(iso3 string) (string, bool) {
	code, ok := iso3ToISO2[strings.ToUpper(strings.TrimSpace(iso3))]
	return code, ok
}

func ISO3ToISO2OrDefault(iso3, fallback string) string {
	if code, ok := ISO3ToISO2(iso3); ok {
		return code
	}
	return fallback
}
