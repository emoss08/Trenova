/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package timeutils

import (
	"fmt"
	"time"

	"github.com/rotisserie/eris"
)

// NowUnix returns the current time as a Unix timestamp.
// The Unix timestamp represents the number of seconds elapsed since January 1, 1970 UTC.
func NowUnix() int64 {
	return time.Now().Unix()
}

func TimeZoneAwareNow(location string) int64 {
	loc, err := time.LoadLocation(location)
	if err != nil {
		return NowUnix()
	}
	now := time.Now().In(loc)
	return now.Unix()
}

// YearsToSeconds converts a number of years into the equivalent number of seconds.
// It assumes a year has 365 days and does not account for leap years.
func YearsToSeconds(years int) int64 {
	return int64(years * 365 * 24 * 60 * 60)
}

// YearsAgoUnix returns the Unix timestamp corresponding to the current time minus a specified number of years.
// The function uses the AddDate method to subtract the specified number of years.
func YearsAgoUnix(years int) int64 {
	return time.Now().AddDate(-years, 0, 0).Unix()
}

// IsAtLeastAge determines if a given date of birth (in Unix timestamp format) represents
// a person who is at least the specified age in years.
// The function compares the date of birth against a timestamp calculated as the current time
// minus the specified age in years.
func IsAtLeastAge(dob int64, age int) bool {
	return dob <= YearsAgoUnix(age)
}

// MonthsAgoUnix returns the Unix timestamp corresponding to the current time minus a specified number of months.
// The function uses the AddDate method to subtract the specified number of months.
func MonthsAgoUnix(months int) int64 {
	return time.Now().AddDate(0, -months, 0).Unix()
}

// MonthsAgoUnixPointer returns a pointer to the Unix timestamp corresponding to the current time minus a specified number of months.
// The function uses the AddDate method to subtract the specified number of months.
func MonthsAgoUnixPointer(months int) *int64 {
	now := time.Now().AddDate(0, -months, 0).Unix()
	return &now
}

// MonthsFromNowUnix returns the Unix timestamp corresponding to the current time plus a specified number of months.
// The function uses the AddDate method to add the specified number of months.
func MonthsFromNowUnix(months int) int64 {
	return time.Now().AddDate(0, months, 0).Unix()
}

// YearsFromNowUnix returns the Unix timestamp corresponding to the current time plus a specified number of years.
// The function uses the AddDate method to add the specified number of years.
func YearsFromNowUnix(years int) int64 {
	return time.Now().AddDate(years, 0, 0).Unix()
}

// SecondsPerYear returns the number of seconds in a year.
// It assumes a year has 365 days and does not account for leap years.
func SecondsPerYear() int64 {
	return int64(365 * 24 * 60 * 60)
}

// DaysToSeconds converts a number of days into the equivalent number of seconds.
func DaysToSeconds(days int) int64 {
	return int64(days * 24 * 60 * 60)
}

// ParseTimeValue parses a given value into a time.Time object.
// Supported input types:
// - string: Parses the string as an RFC3339 formatted timestamp.
// - time.Time: Returns the input as-is.
// - int64: Treats the value as a Unix timestamp and converts it to time.Time.
// Returns an error if the input type is unsupported.
func ParseTimeValue(value any) (time.Time, error) {
	switch v := value.(type) {
	case string:
		return time.Parse(time.RFC3339, v)
	case time.Time:
		return v, nil
	case int64:
		return time.Unix(v, 0), nil
	default:
		return time.Time{}, eris.New(fmt.Sprintf("unsupported time value type: %T", value))
	}
}

// NowUnixPointer returns a pointer to the current Unix timestamp.
// This is useful for cases where a pointer to the timestamp is needed.
func NowUnixPointer() *int64 {
	now := NowUnix()
	return &now
}

// YearsFromNowUnixPointer returns a pointer to the Unix timestamp corresponding to the current time plus a specified number of years.
// The function uses the AddDate method to add the specified number of years.
func YearsFromNowUnixPointer(years int) *int64 {
	now := time.Now().AddDate(years, 0, 0).Unix()
	return &now
}

// YearsAgoUnixPointer returns a pointer to the Unix timestamp corresponding to the current time minus a specified number of years.
// The function uses the AddDate method to subtract the specified number of years.
func YearsAgoUnixPointer(years int) *int64 {
	now := time.Now().AddDate(-years, 0, 0).Unix()
	return &now
}

// CurrentDateInTimezone returns the current date in the specified timezone as a formatted string (YYYY-MM-DD).
// If the timezone cannot be loaded, it falls back to UTC.
func CurrentDateInTimezone(location string) string {
	loc, err := time.LoadLocation(location)
	if err != nil {
		return time.Now().UTC().Format("2006-01-02")
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func CurrentYear() int {
	return time.Now().Year()
}
