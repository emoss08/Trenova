package desktranscript

import (
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
)

type TimeframePreset string

const (
	TimeframeLast7Days    = TimeframePreset("Last7Days")
	TimeframeLast30Days   = TimeframePreset("Last30Days")
	TimeframeLast90Days   = TimeframePreset("Last90Days")
	TimeframeLast12Months = TimeframePreset("Last12Months")
	TimeframeAllTime      = TimeframePreset("AllTime")
	TimeframeCustom       = TimeframePreset("Custom")
)

const DateLayout = "2006-01-02"

func AllTimeframePresets() []TimeframePreset {
	return []TimeframePreset{
		TimeframeLast7Days,
		TimeframeLast30Days,
		TimeframeLast90Days,
		TimeframeLast12Months,
		TimeframeAllTime,
		TimeframeCustom,
	}
}

func (p TimeframePreset) IsValid() bool {
	for _, known := range AllTimeframePresets() {
		if p == known {
			return true
		}
	}

	return false
}

func (p TimeframePreset) days() int {
	switch p {
	case TimeframeLast7Days:
		return 7
	case TimeframeLast30Days:
		return 30
	case TimeframeLast90Days:
		return 90
	case TimeframeLast12Months, TimeframeAllTime, TimeframeCustom:
		return 0
	default:
		return 0
	}
}

type Timeframe struct {
	Preset   TimeframePreset `json:"preset"`
	FromDate string          `json:"fromDate,omitempty"`
	ToDate   string          `json:"toDate,omitempty"`
}

type Window struct {
	From   int64 `json:"from,omitempty"`
	Before int64 `json:"before,omitempty"`
}

func (w Window) Bounded() bool {
	return w.From > 0 || w.Before > 0
}

func AllTime() Timeframe {
	return Timeframe{Preset: TimeframeAllTime}
}

func (t *Timeframe) Validate(field string, multiErr *errortypes.MultiError) {
	if !t.Preset.IsValid() {
		multiErr.Add(field+".preset", errortypes.ErrInvalid, "Choose a timeframe")
		return
	}
	if t.Preset != TimeframeCustom {
		if t.FromDate != "" || t.ToDate != "" {
			multiErr.Add(
				field+".fromDate",
				errortypes.ErrInvalid,
				"Only a custom timeframe takes dates",
			)
		}
		return
	}

	from, fromErr := time.Parse(DateLayout, t.FromDate)
	if fromErr != nil {
		multiErr.Add(field+".fromDate", errortypes.ErrInvalid, "Choose the first day of the range")
	}
	to, toErr := time.Parse(DateLayout, t.ToDate)
	if toErr != nil {
		multiErr.Add(field+".toDate", errortypes.ErrInvalid, "Choose the last day of the range")
	}
	if fromErr == nil && toErr == nil && to.Before(from) {
		multiErr.Add(
			field+".toDate",
			errortypes.ErrInvalid,
			"The last day of the range cannot be before its first",
		)
	}
}

func (t *Timeframe) Resolve(now time.Time, loc *time.Location) Window {
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)

	switch t.Preset {
	case TimeframeLast7Days, TimeframeLast30Days, TimeframeLast90Days:
		return Window{From: today.AddDate(0, 0, 1-t.Preset.days()).Unix()}
	case TimeframeLast12Months:
		return Window{From: today.AddDate(-1, 0, 1).Unix()}
	case TimeframeCustom:
		from, fromErr := time.ParseInLocation(DateLayout, t.FromDate, loc)
		to, toErr := time.ParseInLocation(DateLayout, t.ToDate, loc)
		if fromErr != nil || toErr != nil {
			return Window{}
		}

		return Window{From: from.Unix(), Before: to.AddDate(0, 0, 1).Unix()}
	case TimeframeAllTime:
		return Window{}
	default:
		return Window{}
	}
}
