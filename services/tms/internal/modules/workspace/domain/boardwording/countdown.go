package boardwording

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/timeutils"
)

const countdownWarnDays = 7

type Countdown struct {
	Ends     int64
	Span     int64
	Date     bool
	Location *time.Location
	Ending   string
	Ended    string
}

func (c *Countdown) Read(ctx context.Context, now int64) boardreading.Reading {
	left := c.Ends - now
	value := boardcatalog.Duration(left)
	lapsed := left < 0
	if c.Date {
		days := timeutils.WholeDaysBetween(
			timeutils.LocalDateAsUTC(now, c.Location),
			timeutils.LocalDateAsUTC(c.Ends, c.Location),
		)
		left = days * timeutils.SecondsPerDay
		value = boardcatalog.Days(days)
		lapsed = days < 0
	}
	reading := boardreading.Reading{
		Value:  value,
		Window: &boardreading.WindowFacts{Left: left, Span: max(c.Span, 0)},
		Fact:   &boardreading.Words{Text: c.Ending, At: c.Ends, Day: c.Date},
	}
	switch {
	case lapsed:
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "{0} lapsed"), Value: &value}
		reading.Fact.Text = c.Ended
		reading.Tone = boardcatalog.ToneBad
	case c.Date && left == 0:
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "Ends today")}
		reading.Tone = boardcatalog.ToneWarn
	default:
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "{0} left"), Value: &value}
		if left < countdownWarnDays*timeutils.SecondsPerDay {
			reading.Tone = boardcatalog.ToneWarn
		}
	}

	return reading
}
