package operationsboardreader

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/shared/i18n"
)

const slot = "{0}"

func atPlace(ctx context.Context, stop *stopRow) string {
	where := place(stop)
	if where == "" {
		return ""
	}

	return i18n.T(ctx, "at {0}", where)
}

func lateness(ctx context.Context, route []stopRow, now int64) (boardreading.Reading, bool) {
	for idx := range route {
		stop := &route[idx]
		if stop.Status == string(shipmentvocab.StopStatusCompleted) {
			continue
		}
		due := windowEnd(stop)
		if due <= 0 {
			return boardreading.Reading{}, false
		}
		fact := &boardreading.Words{
			Text:  i18n.T(ctx, "Due {0}"),
			At:    due,
			Place: atPlace(ctx, stop),
		}
		switch {
		case now <= due:
			return ahead(ctx, stop.Estimate, due, fact), true
		case stop.Arrival != nil && *stop.Arrival > 0:
			return behind(i18n.T(ctx, "{0} at the stop"), now-due, true, fact), true
		default:
			return behind(i18n.T(ctx, "{0} behind"), now-due, false, fact), true
		}
	}

	for idx := len(route) - 1; idx >= 0; idx-- {
		stop := &route[idx]
		if pickup(stop) || stop.Arrival == nil || *stop.Arrival <= 0 {
			continue
		}
		due := windowEnd(stop)
		late := int64(0)
		if due > 0 && *stop.Arrival > due {
			late = *stop.Arrival - due
		}
		value := boardcatalog.Duration(late)
		reading := boardreading.Reading{
			Value:    value,
			Standing: boardreading.Words{Text: i18n.T(ctx, "On time")},
			Fact: &boardreading.Words{
				Text:  i18n.T(ctx, "Delivered {0}"),
				At:    *stop.Arrival,
				Place: atPlace(ctx, stop),
			},
			Tone:  boardcatalog.ToneOK,
			Track: &boardreading.TrackFacts{},
		}
		if late > 0 {
			reading.Standing = boardreading.Words{Text: i18n.T(ctx, "{0} late"), Value: &value}
			reading.Tone = boardcatalog.ToneInfo
		}
		return reading, true
	}

	return boardreading.Reading{}, false
}

func behind(text string, seconds int64, held bool, fact *boardreading.Words) boardreading.Reading {
	value := boardcatalog.Duration(seconds)

	return boardreading.Reading{
		Value:    value,
		Standing: boardreading.Words{Text: text, Value: &value},
		Fact:     fact,
		Tone:     boardcatalog.ToneWarn,
		Track:    &boardreading.TrackFacts{Behind: seconds, Held: held},
	}
}

func ahead(
	ctx context.Context,
	estimate *int64,
	due int64,
	fact *boardreading.Words,
) boardreading.Reading {
	reading := boardreading.Reading{
		Value: boardcatalog.Duration(0),
		Fact:  fact,
		Track: &boardreading.TrackFacts{},
	}
	switch {
	case estimate == nil || *estimate <= 0:
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "Not estimated")}
	case *estimate > due:
		value := boardcatalog.Duration(*estimate - due)
		reading.Value = value
		reading.Standing = boardreading.Words{
			Text:  i18n.T(ctx, "Estimated {0} late"),
			Value: &value,
		}
		reading.Tone = boardcatalog.ToneWarn
		reading.Track = &boardreading.TrackFacts{Behind: *estimate - due, Projected: true}
	default:
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "On time")}
		reading.Tone = boardcatalog.ToneOK
	}

	return reading
}
