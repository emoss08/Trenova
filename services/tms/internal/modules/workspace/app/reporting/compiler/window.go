package compiler

import (
	"time"

	"github.com/emoss08/trenova/pkg/dbtype"
)

func ParameterWindow(
	op dbtype.Operator,
	value any,
	loc *time.Location,
	now time.Time,
) (start, end int64, ok bool) {
	if loc == nil {
		loc = time.UTC
	}

	//nolint:exhaustive // only operators that span a window have one
	switch op {
	case dbtype.OpDateRange:
		rangeStart, rangeEnd, err := coerceDateRange(value, loc)
		if err != nil {
			return 0, 0, false
		}
		return rangeStart, rangeEnd, true
	case dbtype.OpLastNDays, dbtype.OpNextNDays:
		n, err := coerceInteger(value)
		if err != nil || n <= 0 {
			return 0, 0, false
		}
		start, end = relativeNDayRange(op, n, now.In(loc), loc)
		return start, end, true
	default:
		return 0, 0, false
	}
}

func relativeNDayRange(
	op dbtype.Operator,
	n int64,
	now time.Time,
	loc *time.Location,
) (start, end int64) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	if op == dbtype.OpNextNDays {
		return today.Unix(), today.AddDate(0, 0, int(n)+1).Unix()
	}
	return today.AddDate(0, 0, -int(n)).Unix(), today.AddDate(0, 0, 1).Unix()
}
