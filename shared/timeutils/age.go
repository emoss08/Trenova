package timeutils

import "time"

const (
	SecondsPerDay   = 86400
	SecondsPerYear  = 31536000
	SecondsPerMonth = 2629746
)

func YearsAgoUnix(years int) int64 {
	return time.Now().AddDate(-years, 0, 0).Unix()
}

func MonthsAgoUnix(months int) int64 {
	return time.Now().AddDate(0, -months, 0).Unix()
}

func DaysAgoUnix(days int) int64 {
	return time.Now().AddDate(0, 0, -days).Unix()
}

func IsAtLeastAge(dob int64, minAge int) bool {
	if dob <= 0 {
		return false
	}
	dobTime := time.Unix(dob, 0)
	minAgeDate := time.Now().AddDate(-minAge, 0, 0)

	return !dobTime.After(minAgeDate)
}

func IsExpired(expiry int64) bool {
	if expiry <= 0 {
		return true
	}

	return expiry < NowUnix()
}

func IsOverdue(dueDate int64) bool {
	if dueDate <= 0 {
		return false
	}

	return dueDate < NowUnix()
}

func IsDueSoon(dueDate int64, warningDays int) bool {
	if dueDate <= 0 {
		return false
	}
	warningThreshold := NowUnix() + int64(warningDays*SecondsPerDay)

	return dueDate <= warningThreshold
}

func IsWithinMonths(timestamp int64, months int) bool {
	if timestamp <= 0 {
		return false
	}
	threshold := MonthsAgoUnix(months)

	return timestamp >= threshold
}

func MaxAllowedUnix(from int64, years int8) int64 {
	fromTime := time.Unix(from, 0)

	return fromTime.AddDate(int(years), 0, 0).Unix()
}
