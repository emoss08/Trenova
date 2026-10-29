package x12field

import (
	"math"
	"strconv"
	"strings"
)

const (
	maxLatitude      = 90.0
	maxLongitude     = 180.0
	minutesPerDegree = 60.0
	secondsPerDegree = 3600.0
	dmsTrailerDigits = 4
)

func ParseX12Latitude(value, direction string) *float64 {
	return parseX12Coordinate(value, direction, maxLatitude, "S")
}

func ParseX12Longitude(value, direction string) *float64 {
	return parseX12Coordinate(value, direction, maxLongitude, "W")
}

func parseX12Coordinate(value, direction string, limit float64, negative string) *float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	sign := 1.0
	switch value[0] {
	case '-':
		sign, value = -1, value[1:]
	case '+':
		value = value[1:]
	}
	if strings.EqualFold(strings.TrimSpace(direction), negative) {
		sign = -1
	}

	degrees, ok := coordinateDegrees(value)
	if !ok || degrees > limit {
		return nil
	}
	result := sign * degrees
	return &result
}

func coordinateDegrees(value string) (float64, bool) {
	if strings.Contains(value, ".") {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < 0 {
			return 0, false
		}
		return parsed, true
	}

	if len(value) <= dmsTrailerDigits {
		return 0, false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	split := len(value) - dmsTrailerDigits
	degrees, _ := strconv.Atoi(value[:split])
	minutes, _ := strconv.Atoi(value[split : split+2])
	seconds, _ := strconv.Atoi(value[split+2:])
	if minutes >= int(minutesPerDegree) || seconds >= int(minutesPerDegree) {
		return 0, false
	}
	return float64(
		degrees,
	) + float64(
		minutes,
	)/minutesPerDegree + float64(
		seconds,
	)/secondsPerDegree, true
}
