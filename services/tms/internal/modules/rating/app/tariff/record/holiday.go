package record

import (
	"cmp"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
)

type Holiday struct {
	Date time.Time
	Name string
}

type HolidayWindow struct {
	From int64
	To   int64
}

func HolidayWindowFor(year int) HolidayWindow {
	return HolidayWindow{
		From: time.Date(year-1, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		To:   time.Date(year+1, time.December, 31, 0, 0, 0, 0, time.UTC).Unix(),
	}
}

func HolidaysFor(rows []*pto.OrgHoliday, year int) []Holiday {
	first, last := year-1, year+1
	seen := make(map[Holiday]struct{}, len(rows)*3)
	out := make([]Holiday, 0, len(rows)*3)
	add := func(date time.Time, name string) {
		holiday := Holiday{Date: date, Name: name}
		if _, ok := seen[holiday]; ok {
			return
		}
		seen[holiday] = struct{}{}
		out = append(out, holiday)
	}

	for _, row := range rows {
		if row == nil || row.Kind != pto.HolidayKindHoliday || row.HolidayDate <= 0 {
			continue
		}
		day := time.Unix(row.HolidayDate, 0).UTC()
		if !row.RecursAnnually {
			if day.Year() >= first && day.Year() <= last {
				add(time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC), row.Name)
			}
			continue
		}
		for y := first; y <= last; y++ {
			date := time.Date(y, day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
			if date.Month() == day.Month() {
				add(date, row.Name)
			}
		}
	}

	slices.SortFunc(out, func(a, b Holiday) int {
		return cmp.Or(a.Date.Compare(b.Date), cmp.Compare(a.Name, b.Name))
	})
	return out
}
