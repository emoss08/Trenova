package utils

import (
	"fmt"
	"strconv"
	"time"
)

func NowUnix() int64 {
	return time.Now().Unix()
}

func DaysToSeconds(days int) int64 {
	return int64(days * 24 * 60 * 60)
}

func ParseTimeValue(value any) (time.Time, error) {
	switch v := value.(type) {
	case string:
		return time.Parse(time.RFC3339, v)
	case time.Time:
		return v, nil
	case int64:
		return time.Unix(v, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported time value type: %T", value)
	}
}

func GetISOWeek(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

func GetYearString(t time.Time, digits int) string {
	year := t.Year()
	switch digits {
	case 2:
		return fmt.Sprintf("%02d", year%100)
	case 4:
		return fmt.Sprintf("%04d", year)
	default:
		yearStr := strconv.Itoa(year)
		if len(yearStr) > digits {
			return yearStr[len(yearStr)-digits:]
		}
		return yearStr
	}
}

func GetCurrentYear() int {
	return time.Now().Year()
}

func IsAtLeastAge(dob int64, age int) bool {
	return dob <= YearsAgoUnix(age)
}

func YearsFromNowUnix(years int) int64 {
	return time.Now().AddDate(years, 0, 0).Unix()
}

func YearsAgoUnix(years int) int64 {
	return time.Now().AddDate(-years, 0, 0).Unix()
}

func YearsAgoToSeconds(years int) int64 {
	return YearsAgoUnix(years) * 60 * 60 * 24 * 365
}

func MaxAllowedUnix(now int64, years int8) int64 {
	nowTime := time.Unix(now, 0)
	maxAllowedTime := nowTime.AddDate(int(years), 0, 0).Unix()
	return maxAllowedTime
}
