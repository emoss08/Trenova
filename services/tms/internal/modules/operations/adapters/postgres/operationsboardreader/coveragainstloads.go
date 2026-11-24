package operationsboardreader

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carriercoverports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
)

const lapseTickDays = 7

type CoverAgainstLoads struct {
	cover carriercoverports.Reader
}

func NewCoverAgainstLoads(p CoverParams) *CoverAgainstLoads {
	return &CoverAgainstLoads{cover: p.Cover}
}

func (*CoverAgainstLoads) Metric() string { return operationsboard.MetricCoverAgainstLoads }

type lapseWindow struct {
	now   int64
	until int64
	last  time.Time
	loc   *time.Location
}

func lapseWindowOf(req *boardreaderports.MetricRequest) lapseWindow {
	loc := serviceZone(req.Location)
	period := req.Period
	if period == "" {
		period = boardcatalog.PeriodNext30Days
	}
	_, last, ok := period.Days(req.Now, loc)
	if !ok {
		_, last, _ = boardcatalog.PeriodNext30Days.Days(req.Now, loc)
	}

	return lapseWindow{
		now:   req.Now,
		until: last.AddDate(0, 0, 1).Unix() - 1,
		last:  last,
		loc:   loc,
	}
}

func (r *CoverAgainstLoads) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	window := lapseWindowOf(req)
	covers, err := r.cover.Ending(ctx, &carriercoverports.EndingRequest{
		Now:   window.now,
		Until: window.until,
		Limit: carriercoverports.MaxCarriers,
		Loads: true,
	})
	if err != nil {
		return nil, err
	}
	drawn := covers[:min(len(covers), operationsboard.MaxCoverCarriers)]
	ids := make([]pulid.ID, 0, len(drawn))
	for _, cover := range drawn {
		ids = append(ids, cover.CarrierID)
	}
	loads, err := r.cover.Loads(ctx, &carriercoverports.LoadsRequest{
		CarrierIDs: ids,
		PerCarrier: operationsboard.MaxCoverLoads + 1,
	})
	if err != nil {
		return nil, err
	}

	return lapseRead(ctx, &lapseInput{window: window, covers: covers, loads: loads}), nil
}

type lapseInput struct {
	window lapseWindow
	covers []*carriercoverports.Cover
	loads  map[pulid.ID][]carriercoverports.Load
}

type lapseTally struct {
	lapsing int64
	lapsed  int64
	loads   int64
	past    int64
	drawn   int64
	whole   bool
}

func lapseRead(ctx context.Context, in *lapseInput) *boardcatalog.MetricRead {
	tally := lapseTally{whole: len(in.covers) < carriercoverports.MaxCarriers}
	for _, cover := range in.covers {
		if cover.EndsAt() < in.window.now {
			tally.lapsed++
			continue
		}
		tally.lapsing++
	}
	drawn := in.covers[:min(len(in.covers), operationsboard.MaxCoverCarriers)]
	if len(in.covers) > len(drawn) {
		tally.whole = false
	}
	rows := make([]boardcatalog.ChartRow, 0, len(drawn))
	for _, cover := range drawn {
		rows = append(rows, lapseRow(ctx, cover, in.loads[cover.CarrierID], in.window, &tally))
	}

	read := &boardcatalog.MetricRead{
		Title:   i18n.T(ctx, "Their loads against the end of cover"),
		Source:  lapseSource(ctx, &tally, in.window),
		Figures: lapseFigures(&tally, len(in.covers) < carriercoverports.MaxCarriers),
		Chart: &boardcatalog.Chart{
			Unit:   boardcatalog.ValueCount,
			X:      lapseTicks(in.window),
			Series: []boardcatalog.Series{},
			Rows:   rows,
		},
	}

	return read
}

func lapseRow(
	ctx context.Context,
	cover *carriercoverports.Cover,
	loads []carriercoverports.Load,
	window lapseWindow,
	tally *lapseTally,
) boardcatalog.ChartRow {
	ends := cover.EndsAt()
	row := boardcatalog.ChartRow{
		Label: cover.Name,
		Ref:   &boardcatalog.Ref{Kind: carrierKind, ID: cover.CarrierID},
		Spans: []boardcatalog.Span{{
			From: min(window.now, ends),
			To:   ends,
			Kind: boardcatalog.SpanCover,
		}},
		Dots: []boardcatalog.Dot{},
	}
	if !cover.LoadsReadable() || int64(len(loads)) != cover.ReadableLoads ||
		len(loads) > operationsboard.MaxCoverLoads {
		tally.whole = false
	}
	tally.loads += cover.OpenLoads
	for idx, load := range loads {
		if idx >= operationsboard.MaxCoverLoads {
			break
		}
		if load.DeliversAt <= 0 {
			tally.whole = false
			continue
		}
		tally.drawn++
		dot := boardcatalog.Dot{At: load.DeliversAt}
		if load.DeliversAt > ends {
			tally.past++
			dot.Marked = true
			dot.Label = i18n.T(ctx, "{0} delivers {1}", load.ProNumber,
				i18n.Day(i18n.FromContext(ctx), time.Unix(load.DeliversAt, 0).In(window.loc),
					time.Unix(window.now, 0).In(window.loc)))
		}
		row.Dots = append(row.Dots, dot)
	}

	return row
}

func lapseFigures(tally *lapseTally, allCarriers bool) map[string]*boardcatalog.Value {
	figures := map[string]*boardcatalog.Value{
		operationsboard.FigureLapsingCarriers: nil,
		operationsboard.FigureLapsedCarriers:  nil,
		operationsboard.FigurePastCover:       nil,
		operationsboard.FigureLoadsWithThem:   nil,
	}
	if allCarriers {
		figures[operationsboard.FigureLapsingCarriers] = new(boardcatalog.Count(tally.lapsing))
		figures[operationsboard.FigureLapsedCarriers] = new(boardcatalog.Count(tally.lapsed))
	}
	if tally.whole {
		figures[operationsboard.FigurePastCover] = new(boardcatalog.Count(tally.past))
		figures[operationsboard.FigureLoadsWithThem] = new(boardcatalog.Count(tally.loads))
	}

	return figures
}

func lapseTicks(window lapseWindow) []boardcatalog.Tick {
	ticks := make([]boardcatalog.Tick, 0, 5)
	today := time.Unix(window.now, 0).In(window.loc)
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, window.loc)
	for day := start.AddDate(0, 0, lapseTickDays); day.Before(window.last); day = day.AddDate(
		0, 0, lapseTickDays,
	) {
		ticks = append(ticks, boardcatalog.Tick{At: day.Unix()})
	}

	return append(ticks, boardcatalog.Tick{At: window.last.Unix()})
}

func lapseSource(ctx context.Context, tally *lapseTally, window lapseWindow) string {
	return i18n.T(ctx,
		"{0, plural, one {# load} other {# loads}} with carriers whose cover ends by {1}",
		tally.drawn,
		i18n.Day(i18n.FromContext(ctx), window.last, time.Unix(window.now, 0).In(window.loc)))
}
