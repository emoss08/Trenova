package workforceboardreader

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workforceboard"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/weekhoursports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
)

const (
	rotaRows   = 8
	rotaRoom   = 2
	figureOver = "over"
	figureRoom = "room"
	figureRota = "onRota"
	noTime     = "–"
)

type RotaWeek struct {
	weeks  weekhoursports.Reader
	narrow weekhoursports.TeamNarrower
}

func NewRotaWeek(p WeekParams) *RotaWeek {
	return &RotaWeek{weeks: p.Weeks, narrow: p.Narrow}
}

func (*RotaWeek) Metric() string { return workforceboard.MetricRotaWeek }

func (r *RotaWeek) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	read := &boardcatalog.MetricRead{
		Title: i18n.T(ctx, "The week, day by day"),
		Figures: map[string]*boardcatalog.Value{
			figureOver: nil, figureRoom: nil, figureRota: nil,
		},
	}
	managers, err := r.narrow.Narrow(ctx, req.Actor, req.Tenant)
	if errors.Is(err, weekhoursports.ErrNoTeam) {
		read.Source = i18n.T(ctx, "no one on your team's rota")

		return read, nil
	}
	if err != nil {
		return nil, err
	}
	leading, err := r.weeks.Leading(ctx, &weekhoursports.LeadingRequest{
		Tenant:     req.Tenant,
		ManagerIDs: managers,
		Now:        req.Now,
		Location:   req.Location,
		Limit:      rotaRows,
		Room:       rotaRoom,
	})
	if err != nil {
		return nil, err
	}
	read.Source = i18n.T(
		ctx,
		"this week's rota for {0, plural, one {# worker} other {# workers}}",
		leading.OnRota,
	)
	onRota := boardcatalog.Count(leading.OnRota)
	read.Figures[figureRota] = &onRota
	if leading.Exact {
		over := boardcatalog.Count(leading.Over)
		room := boardcatalog.Count(max(0, leading.OnRota-leading.Over-leading.OnLine))
		read.Figures[figureOver] = &over
		read.Figures[figureRoom] = &room
	}
	read.Table = rotaTable(ctx, leading, req.Location)

	return read, nil
}

func weekdayLabel(ctx context.Context, day time.Time) string {
	date := day.Day()
	switch day.Weekday() {
	case time.Sunday:
		return i18n.T(ctx, "Sun {0}", date)
	case time.Monday:
		return i18n.T(ctx, "Mon {0}", date)
	case time.Tuesday:
		return i18n.T(ctx, "Tue {0}", date)
	case time.Wednesday:
		return i18n.T(ctx, "Wed {0}", date)
	case time.Thursday:
		return i18n.T(ctx, "Thu {0}", date)
	case time.Friday:
		return i18n.T(ctx, "Fri {0}", date)
	case time.Saturday:
		return i18n.T(ctx, "Sat {0}", date)
	}

	return ""
}

func rotaTable(
	ctx context.Context,
	leading *weekhoursports.Leading,
	loc *time.Location,
) *boardcatalog.Table {
	if loc == nil {
		loc = time.UTC
	}
	start := time.Unix(leading.Start, 0).In(loc)
	columns := make([]boardcatalog.Column, 0, weekhours.DaysInWeek+2)
	columns = append(columns, boardcatalog.Column{
		Label: i18n.T(ctx, "Person"),
		Kind:  boardcatalog.ColumnText,
	})
	for offset := range weekhours.DaysInWeek {
		columns = append(columns, boardcatalog.Column{
			Label: weekdayLabel(ctx, start.AddDate(0, 0, offset)),
			Kind:  boardcatalog.ColumnValue,
		})
	}
	columns = append(columns, boardcatalog.Column{
		Label: i18n.T(ctx, "Week"),
		Kind:  boardcatalog.ColumnValue,
	})

	shown := make([]*weekhoursports.WorkerWeek, 0, len(leading.Rows)+len(leading.Room))
	shown = append(shown, leading.Rows...)
	shown = append(shown, leading.Room...)
	rows := make([][]boardcatalog.TableCell, 0, len(shown))
	for _, worker := range shown {
		rows = append(rows, rotaRow(ctx, worker))
	}

	return &boardcatalog.Table{Columns: columns, Rows: rows}
}

func rotaRow(ctx context.Context, worker *weekhoursports.WorkerWeek) []boardcatalog.TableCell {
	week := &worker.Week
	row := make([]boardcatalog.TableCell, 0, weekhours.DaysInWeek+2)
	row = append(row, boardcatalog.TableCell{Text: worker.Name})
	for idx := range week.Days {
		row = append(row, dayCell(ctx, &week.Days[idx]))
	}
	total := minutes(week.Planned())
	weekCell := boardcatalog.TableCell{Value: &total}
	if week.Over() {
		weekCell.Mark = boardcatalog.MarkW
	}

	return append(row, weekCell)
}

func dayCell(ctx context.Context, day *weekhours.Day) boardcatalog.TableCell {
	cell := boardcatalog.TableCell{}
	switch day.State {
	case weekhours.DayWorked:
		value := minutes(day.Minutes)
		cell.Value, cell.Mark = &value, boardcatalog.MarkHot
	case weekhours.DayScheduled:
		value := minutes(day.Minutes)
		cell.Value = &value
	case weekhours.DayAway:
		cell.Text, cell.Mark = i18n.T(ctx, "Time off"), boardcatalog.MarkOff
	case weekhours.DayMissed:
		cell.Text, cell.Mark = i18n.T(ctx, "Not worked"), boardcatalog.MarkOff
	case weekhours.DayOff:
		cell.Text, cell.Mark = noTime, boardcatalog.MarkOff
	}
	if day.Crosses {
		cell.Mark = boardcatalog.MarkW
	}

	return cell
}
