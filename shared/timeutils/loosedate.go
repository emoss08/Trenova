package timeutils

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var (
	ErrDateMissing      = errors.New("date is missing")
	ErrDateUnrecognized = errors.New("date is not recognised")
)

const (
	excelEpochOffset = 25569
	minExcelSerial   = 20000
	maxExcelSerial   = 2958465
)

const (
	layoutDateTimeSeconds = "2006-01-02 15:04:05"
	layoutDateTimeMinutes = "2006-01-02 15:04"
)

var looseDateLayouts = [...]string{
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	layoutDateTimeSeconds,
	layoutDateTimeMinutes,
	"2006-01-02",
	"01/02/2006 15:04:05",
	"01/02/2006 15:04",
	"01/02/2006 3:04:05 PM",
	"01/02/2006 3:04 PM",
	"01/02/2006",
	"1/2/2006 15:04:05",
	"1/2/2006 15:04",
	"1/2/2006 3:04:05 PM",
	"1/2/2006 3:04 PM",
	"1/2/2006",
	"01/02/06 15:04",
	"01/02/06",
	"1/2/06 15:04",
	"1/2/06",
	"2006/01/02",
	"01-02-2006",
	"1-2-2006",
	"02-Jan-2006",
	"2-Jan-2006",
	"Jan 2, 2006",
	"January 2, 2006",
	"20060102",
}

func ParseLooseDate(raw string, loc *time.Location) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, ErrDateMissing
	}
	if loc == nil {
		loc = time.UTC
	}

	for _, layout := range looseDateLayouts {
		parsed, err := time.ParseInLocation(layout, value, loc)
		if err == nil {
			return parsed.Unix(), nil
		}
	}

	if serial, err := strconv.ParseFloat(value, 64); err == nil {
		if serial >= minExcelSerial && serial <= maxExcelSerial {
			return excelSerialToUnix(serial, loc), nil
		}
	}

	return 0, fmt.Errorf("%q: %w", raw, ErrDateUnrecognized)
}

func excelSerialToUnix(serial float64, loc *time.Location) int64 {
	days := math.Floor(serial)
	fraction := serial - days
	utcDay := int64(days-excelEpochOffset) * SecondsPerDay
	seconds := int64(math.Round(fraction * SecondsPerDay))
	utc := time.Unix(utcDay+seconds, 0).UTC()
	local := time.Date(
		utc.Year(), utc.Month(), utc.Day(),
		utc.Hour(), utc.Minute(), utc.Second(), 0, loc,
	)
	return local.Unix()
}
