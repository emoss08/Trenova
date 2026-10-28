package conversationschedule

import (
	"strings"
	"time"
)

// zonePattern is the clock a message names right after its time: "EST",
// "ET", "Eastern", "Eastern time", "Pacific Standard Time", "UTC".
var zonePattern = cadencePattern(
	`(?i)^([a-z]+)\b(?:\s+(?:standard|daylight))?(?:\s+time\b)?`,
)

// namedZones are the clocks a person names by abbreviation or region. An
// abbreviation is read as the region's clock whichever half of the year it
// names: "9:30 EST" in July means 9:30 in New York, as the person meant it,
// not 10:30.
var namedZones = map[string]string{
	"et":       "America/New_York",
	"est":      "America/New_York",
	"edt":      "America/New_York",
	"eastern":  "America/New_York",
	"ct":       "America/Chicago",
	"cst":      "America/Chicago",
	"cdt":      "America/Chicago",
	"central":  "America/Chicago",
	"mt":       "America/Denver",
	"mst":      "America/Denver",
	"mdt":      "America/Denver",
	"mountain": "America/Denver",
	"pt":       "America/Los_Angeles",
	"pst":      "America/Los_Angeles",
	"pdt":      "America/Los_Angeles",
	"pacific":  "America/Los_Angeles",
	"akt":      "America/Anchorage",
	"akst":     "America/Anchorage",
	"akdt":     "America/Anchorage",
	"alaska":   "America/Anchorage",
	"ht":       "Pacific/Honolulu",
	"hst":      "Pacific/Honolulu",
	"hawaii":   "Pacific/Honolulu",
	"utc":      "UTC",
	"gmt":      "UTC",
}

// leadingZone reads the clock named at the start of what follows a time and
// returns it as an IANA name with the rest of the text. A text that opens
// with no clock it knows is returned whole, with no zone.
func leadingZone(text string) (zone, rest string) {
	match := zonePattern.FindStringSubmatch(text)
	if match == nil {
		return "", text
	}
	zone, ok := namedZones[strings.ToLower(match[1])]
	if !ok {
		return "", text
	}

	return zone, text[len(match[0]):]
}

// ResolveZone reads a clock as a person or an agent names it, an IANA name
// ("America/New_York") or an abbreviation or region ("EST", "Pacific"), and
// reports whether it names one.
func ResolveZone(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	if zone, ok := namedZones[strings.ToLower(name)]; ok {
		return zone, true
	}
	if zone, rest := leadingZone(name); zone != "" && strings.TrimSpace(rest) == "" {
		return zone, true
	}
	if _, err := time.LoadLocation(name); err != nil || strings.EqualFold(name, "local") {
		return "", false
	}

	return name, true
}
