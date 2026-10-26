package edi

import (
	"strings"
	"time"
)

type X12DateTime struct {
	Date     string `json:"date,omitempty"`
	Time     string `json:"time,omitempty"`
	TimeCode string `json:"timeCode,omitempty"`
}

type x12Zone struct {
	name   string
	offset int
	region string
}

const (
	X12TimeCodeLocal   = "LT"
	secondsPerHour     = 3600
	secondsPerHalfHour = 1800
	x12ClockSeconds    = "150405"
)

var x12Zones = map[string]x12Zone{
	"UT": {name: "UTC"},
	"GM": {name: "GMT"},
	"ET": {region: "America/New_York"},
	"ES": {name: "EST", offset: -5 * secondsPerHour},
	"ED": {name: "EDT", offset: -4 * secondsPerHour},
	"CT": {region: "America/Chicago"},
	"CS": {name: "CST", offset: -6 * secondsPerHour},
	"CD": {name: "CDT", offset: -5 * secondsPerHour},
	"MT": {region: "America/Denver"},
	"MS": {name: "MST", offset: -7 * secondsPerHour},
	"MD": {name: "MDT", offset: -6 * secondsPerHour},
	"PT": {region: "America/Los_Angeles"},
	"PS": {name: "PST", offset: -8 * secondsPerHour},
	"PD": {name: "PDT", offset: -7 * secondsPerHour},
	"AT": {region: "America/Anchorage"},
	"AS": {name: "AKST", offset: -9 * secondsPerHour},
	"AD": {name: "AKDT", offset: -8 * secondsPerHour},
	"HT": {region: "Pacific/Honolulu"},
	"HS": {name: "HST", offset: -10 * secondsPerHour},
	"HD": {name: "HDT", offset: -9 * secondsPerHour},
	"TT": {region: "America/Halifax"},
	"TS": {name: "AST", offset: -4 * secondsPerHour},
	"TD": {name: "ADT", offset: -3 * secondsPerHour},
	"NT": {region: "America/St_Johns"},
	"NS": {name: "NST", offset: -3*secondsPerHour - secondsPerHalfHour},
	"ND": {name: "NDT", offset: -2*secondsPerHour - secondsPerHalfHour},
}

func NormalizeX12TimeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func IsKnownX12TimeCode(code string) bool {
	code = NormalizeX12TimeCode(code)
	if code == X12TimeCodeLocal {
		return true
	}
	_, ok := x12Zones[code]
	return ok
}

func (d X12DateTime) IsZero() bool {
	return strings.TrimSpace(d.Date) == ""
}

func (d X12DateTime) IsLocal() bool {
	code := NormalizeX12TimeCode(d.TimeCode)
	return code == "" || code == X12TimeCodeLocal || !IsKnownX12TimeCode(code)
}

func (d X12DateTime) Location(local *time.Location) *time.Location {
	code := NormalizeX12TimeCode(d.TimeCode)
	zone, ok := x12Zones[code]
	if !ok {
		if local != nil {
			return local
		}
		return time.UTC
	}
	if zone.region != "" {
		if loc, err := time.LoadLocation(zone.region); err == nil {
			return loc
		}
	}
	if zone.offset == 0 && zone.region == "" {
		return time.UTC
	}
	return time.FixedZone(zone.name, zone.offset)
}

func (d X12DateTime) Unix(local *time.Location) (int64, bool) {
	date := strings.TrimSpace(d.Date)
	if date == "" {
		return 0, false
	}
	dateLayout := "20060102"
	if len(date) == len("060102") {
		dateLayout = "060102"
	}

	clock := strings.TrimSpace(d.Time)
	if len(clock) > len(x12ClockSeconds) {
		clock = clock[:len(x12ClockSeconds)]
	}
	layout, value := dateLayout, date
	switch len(clock) {
	case len("1504"):
		layout, value = dateLayout+"1504", date+clock
	case len(x12ClockSeconds):
		layout, value = dateLayout+x12ClockSeconds, date+clock
	}

	parsed, err := time.ParseInLocation(layout, value, d.Location(local))
	if err != nil {
		return 0, false
	}
	return parsed.Unix(), true
}
