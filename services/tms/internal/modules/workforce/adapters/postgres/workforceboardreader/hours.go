package workforceboardreader

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workforceboard"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/weekhoursports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

const secondsPerMinute = int64(60)

type WeekParams struct {
	fx.In

	Weeks  weekhoursports.Reader
	Narrow weekhoursports.TeamNarrower
}

type HoursThisWeek struct {
	weeks  weekhoursports.Reader
	narrow weekhoursports.TeamNarrower
}

func NewHoursThisWeek(p WeekParams) *HoursThisWeek {
	return &HoursThisWeek{weeks: p.Weeks, narrow: p.Narrow}
}

func (*HoursThisWeek) Kind() recordkind.Kind { return workerKind }

func (*HoursThisWeek) Measure() string { return workforceboard.MeasureHoursThisWeek }

func (r *HoursThisWeek) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Reading{}, nil
	}
	managers, err := r.narrow.Narrow(ctx, req.Actor, req.Tenant)
	if errors.Is(err, weekhoursports.ErrNoTeam) {
		return map[pulid.ID]boardreading.Reading{}, nil
	}
	if err != nil {
		return nil, err
	}
	weeks, err := r.weeks.Workers(ctx, &weekhoursports.WorkersRequest{
		Tenant:     req.Tenant,
		WorkerIDs:  req.IDs,
		ManagerIDs: managers,
		Now:        req.Now,
		Location:   req.Location,
	})
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]boardreading.Reading, len(weeks))
	for id, week := range weeks {
		if week.Week.Shift == "" && week.Week.Worked == 0 {
			continue
		}
		out[id] = hoursReading(ctx, &week.Week)
	}

	return out, nil
}

func minutes(n int32) boardcatalog.Value {
	return boardcatalog.Hours(int64(n) * secondsPerMinute)
}

func hoursReading(ctx context.Context, week *weekhours.Week) boardreading.Reading {
	overtime := minutes(week.Overtime())
	worked := minutes(week.Worked)
	reading := boardreading.Reading{
		Value: overtime,
		Limit: &boardreading.LimitFacts{
			Used:    decimal.NewFromInt32(week.Worked),
			Planned: decimal.NewFromInt32(week.Planned()),
			Bound:   decimal.NewFromInt32(week.Threshold),
		},
		Fact: &boardreading.Words{Text: i18n.T(ctx, "{0} worked"), Value: &worked},
	}
	if week.Shift != "" {
		reading.Fact.Place = i18n.T(ctx, "on {0}", week.Shift)
	}
	switch {
	case week.Over():
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "{0} over"), Value: &overtime}
		reading.Tone = boardcatalog.ToneWarn
		if week.AlreadyOver() {
			reading.Tone = boardcatalog.ToneBad
		}
	case week.OnLine():
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "On the line")}
	default:
		room := minutes(week.Room())
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "{0} under"), Value: &room}
	}

	return reading
}
