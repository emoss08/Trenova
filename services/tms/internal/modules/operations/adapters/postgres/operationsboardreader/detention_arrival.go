package operationsboardreader

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionboard"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

const (
	arrivalWindowHours   = 2
	arrivalWindows       = 24 / arrivalWindowHours
	arrivalWindowMinimum = 10
	arrivalHourLayout    = "3 PM"
)

type DetentionByArrivalHour struct {
	intelligence detentionports.DetentionIntelligenceRepository
}

func NewDetentionByArrivalHour(p DetentionParams) *DetentionByArrivalHour {
	return &DetentionByArrivalHour{intelligence: p.Intelligence}
}

func (*DetentionByArrivalHour) Metric() string {
	return detentionboard.MetricDetentionByArrivalHour
}

func (r *DetentionByArrivalHour) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	ask := &detentionAsk{
		tenant:   req.Tenant,
		period:   req.Period,
		now:      req.Now,
		location: req.Location,
	}
	window, ok := ask.window()
	if !ok {
		return &boardcatalog.MetricRead{}, nil
	}
	cells, err := r.intelligence.ArrivalFigures(ctx, window.req)
	if err != nil {
		return nil, err
	}

	return arrivalShares(ctx, req.Period, cells), nil
}

type arrivalWindow struct {
	stops    int64
	detained int64
}

func (w arrivalWindow) share() decimal.Decimal {
	return decimal.NewFromInt(w.detained).Div(decimal.NewFromInt(w.stops))
}

func arrivalShares(
	ctx context.Context,
	period boardcatalog.Period,
	cells []detentionports.ArrivalCell,
) *boardcatalog.MetricRead {
	windows := make([]arrivalWindow, arrivalWindows)
	var all arrivalWindow
	for _, cell := range cells {
		if cell.Hour < 0 || cell.Hour >= arrivalWindows*arrivalWindowHours {
			continue
		}
		at := &windows[cell.Hour/arrivalWindowHours]
		at.stops += int64(cell.StopCount)
		at.detained += int64(cell.DetainedCount)
		all.stops += int64(cell.StopCount)
		all.detained += int64(cell.DetainedCount)
	}

	read := &boardcatalog.MetricRead{
		Title: i18n.T(ctx, "Arrivals detained, by the hour they arrived"),
		Source: i18n.T(
			ctx,
			"{0, plural, one {# stop} other {# stops}} at every facility together, {1}",
			all.stops,
			detentionPeriodWords(ctx, period),
		),
		Figures: map[string]*boardcatalog.Value{
			"worstWindow":      nil,
			"worstWindowShare": nil,
			"detainedShare":    nil,
		},
	}
	if all.stops == 0 {
		return read
	}

	overall := boardcatalog.Percent(all.share())
	read.Figures["detainedShare"] = &overall
	chart := &boardcatalog.Chart{
		Unit:   boardcatalog.ValuePercent,
		X:      make([]boardcatalog.Tick, arrivalWindows),
		Series: []boardcatalog.Series{{Name: i18n.T(ctx, "Detained"), Role: boardcatalog.RoleNow}},
		Ref:    &boardcatalog.Line{Value: overall.Number, Label: i18n.T(ctx, "Every arrival")},
	}
	chart.Series[0].Points = make([]*decimal.Decimal, arrivalWindows)
	worst := worstArrivalWindow(windows)
	for idx := range windows {
		chart.X[idx] = boardcatalog.Tick{Label: arrivalHour(idx), Marked: idx == worst}
		if windows[idx].stops == 0 {
			continue
		}
		share := windows[idx].share()
		chart.Series[0].Points[idx] = &share
	}
	if worst >= 0 {
		chart.Marked = []int{worst}
		label := boardcatalog.Value{
			Kind: boardcatalog.ValueText,
			Text: i18n.T(ctx, "{0} to {1}", arrivalHour(worst), arrivalHour(worst+1)),
		}
		share := boardcatalog.Percent(windows[worst].share())
		read.Figures["worstWindow"] = &label
		read.Figures["worstWindowShare"] = &share
	}
	read.Chart = chart

	return read
}

func worstArrivalWindow(windows []arrivalWindow) int {
	worst := -1
	for _, floor := range []int64{arrivalWindowMinimum, 1} {
		for idx := range windows {
			if windows[idx].stops < floor {
				continue
			}
			if worst < 0 || windows[idx].share().GreaterThan(windows[worst].share()) {
				worst = idx
			}
		}
		if worst >= 0 {
			return worst
		}
	}

	return worst
}

func arrivalHour(window int) string {
	hour := (window * arrivalWindowHours) % (arrivalWindows * arrivalWindowHours)

	return time.Date(2000, time.January, 1, hour, 0, 0, 0, time.UTC).Format(arrivalHourLayout)
}
