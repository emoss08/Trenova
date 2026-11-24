package weekhoursrepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerteam"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet/timesheetdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave/workerleavedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule/workerscheduledb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/weekhoursports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	assignmentsPerWorker = 4
	entriesPerWorker     = 64
	spansPerWorker       = 8
	roomCandidates       = 3
	rankSlackMinutes     = int64(24 * 60)
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) weekhoursports.Reader {
	return &repository{db: p.DB}
}

type headRow struct {
	ID     pulid.ID
	Name   string
	Status string
}

type planRow struct {
	WorkerID         pulid.ID
	CycleOffsetWeeks int16
	From             int64
	To               *int64
	Name             string
	DaysOfWeek       string
	StartMinute      int16
	DurationMinutes  int16
	CycleWeeks       int16
}

type sheetRow struct {
	WorkerID  pulid.ID
	Threshold int32
}

type entryRow struct {
	WorkerID     pulid.ID
	ClockedInAt  int64
	ClockedOutAt *int64
	BreakMinutes int32
}

type spanRow struct {
	WorkerID pulid.ID
	From     int64
	To       int64
}

type rankRow struct {
	WorkerID pulid.ID
	Key      int64
}

type countRow struct {
	Workers int64
}

type window struct {
	start  int64
	end    int64
	period int64
}

func windowOf(now int64, req *weekhoursports.WorkersRequest) window {
	start, end := weekhours.Bounds(now, req.Location)

	return window{start: start, end: end, period: weekhours.TimesheetPeriod(start, req.Location)}
}

func distinct(ids []pulid.ID) []pulid.ID {
	out := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		if id.IsNotNil() && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	if len(out) > weekhoursports.MaxWorkers {
		out = out[:weekhoursports.MaxWorkers]
	}

	return out
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func (r *repository) Workers(
	ctx context.Context,
	req *weekhoursports.WorkersRequest,
) (map[pulid.ID]*weekhoursports.WorkerWeek, error) {
	ids := distinct(req.WorkerIDs)
	if len(ids) == 0 {
		return map[pulid.ID]*weekhoursports.WorkerWeek{}, nil
	}
	win := windowOf(req.Now, req)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*weekhoursports.WorkerWeek, error) {
			var (
				heads   []headRow
				plans   []planRow
				sheets  []sheetRow
				entries []entryRow
				pto     []spanRow
				leave   []spanRow
			)
			if err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.IntoSlice(headsQuery(ids, req.ManagerIDs), &heads),
				keel.IntoSlice(plansQuery(ids, win), &plans),
				keel.IntoSlice(sheetsQuery(ids, win), &sheets),
				keel.IntoSlice(entriesQuery(ids, win), &entries),
				keel.IntoSlice(ptoQuery(ids, win), &pto),
				keel.IntoSlice(leaveQuery(ids, win), &leave),
			); err != nil {
				return nil, fmt.Errorf("read the week's hours: %w", err)
			}

			return assemble(&evidence{
				req:     req,
				win:     win,
				heads:   heads,
				plans:   plans,
				sheets:  sheets,
				entries: entries,
				away:    slices.Concat(pto, leave),
			}), nil
		},
	)
}

func headsQuery(ids, managerIDs []pulid.ID) keel.Query[*worker.Worker] {
	w := workerdb.Workers

	return w.Pick(w.ID, text(keel.ConcatWS(" ", w.FirstName, w.LastName)), text(w.Status)).
		Where(w.ID.In(ids...), workerteam.Where(w.ID.Col, managerIDs)).
		Limit(len(ids))
}

func plansQuery(ids []pulid.ID, win window) keel.Query[*workerschedule.WorkerShiftAssignment] {
	sa := workerscheduledb.WorkerShiftAssignments
	st := workerscheduledb.ShiftTemplates

	return sa.Join(st, st.ID.EqCol(sa.ShiftTemplateID)).
		Pick(
			sa.WorkerID, sa.CycleOffsetWeeks, sa.EffectiveFrom, sa.EffectiveTo,
			st.Name, st.DaysOfWeek, st.StartMinute, st.DurationMinutes, st.CycleWeeks,
		).
		Where(
			sa.WorkerID.In(ids...),
			sa.EffectiveFrom.Lt(win.end),
			keel.Or(sa.EffectiveTo.IsNull(), sa.EffectiveTo.Gte(win.start)),
		).
		Order(sa.WorkerID.Asc(), sa.EffectiveFrom.Asc()).
		Limit(len(ids) * assignmentsPerWorker)
}

func sheetsQuery(ids []pulid.ID, win window) keel.Query[*timesheet.Timesheet] {
	ts := timesheetdb.Timesheets

	return ts.Pick(ts.WorkerID, ts.OvertimeThresholdMinutes).
		Where(ts.WorkerID.In(ids...), ts.PeriodStart.Eq(win.period)).
		Limit(len(ids))
}

func entriesQuery(ids []pulid.ID, win window) keel.Query[*timesheet.TimeClockEntry] {
	e := timesheetdb.TimeClockEntries

	return e.Pick(e.WorkerID, e.ClockedInAt, e.ClockedOutAt, e.BreakMinutes).
		Where(e.WorkerID.In(ids...), e.ClockedInAt.Gte(win.start), e.ClockedInAt.Lt(win.end)).
		Order(e.WorkerID.Asc(), e.ClockedInAt.Asc()).
		Limit(len(ids) * entriesPerWorker)
}

func ptoQuery(ids []pulid.ID, win window) keel.Query[*worker.WorkerPTO] {
	p := workerdb.WorkerPTOs

	return p.Pick(p.WorkerID, p.StartDate, p.EndDate).
		Where(
			p.WorkerID.In(ids...),
			p.Status.Eq(worker.PTOStatusApproved),
			p.StartDate.Lt(win.end),
			p.EndDate.Gte(win.start),
		).
		Limit(len(ids) * spansPerWorker)
}

func leaveQuery(ids []pulid.ID, win window) keel.Query[*workerleave.WorkerLeaveCase] {
	c := workerleavedb.WorkerLeaveCases

	return c.Pick(c.WorkerID, c.StartsAt, c.EndsAt.Coalesce(int64(0))).
		Where(
			c.WorkerID.In(ids...),
			c.Status.Eq(workerleave.LeaveCaseApproved),
			c.StartsAt.Lt(win.end),
			keel.Or(c.EndsAt.IsNull(), c.EndsAt.Gte(win.start)),
		).
		Limit(len(ids) * spansPerWorker)
}

type evidence struct {
	req     *weekhoursports.WorkersRequest
	win     window
	heads   []headRow
	plans   []planRow
	sheets  []sheetRow
	entries []entryRow
	away    []spanRow
}

func assemble(ev *evidence) map[pulid.ID]*weekhoursports.WorkerWeek {
	inputs := make(map[pulid.ID]*weekhours.Input, len(ev.heads))
	for idx := range ev.heads {
		inputs[ev.heads[idx].ID] = &weekhours.Input{
			WeekStart: ev.win.start,
			Location:  ev.req.Location,
			Now:       ev.req.Now,
		}
	}
	for idx := range ev.plans {
		plan := &ev.plans[idx]
		if in, ok := inputs[plan.WorkerID]; ok {
			in.Assignments = append(in.Assignments, weekhours.Assignment{
				Template: &workerschedule.ShiftTemplate{
					Name:            plan.Name,
					DaysOfWeek:      plan.DaysOfWeek,
					StartMinute:     plan.StartMinute,
					DurationMinutes: plan.DurationMinutes,
					CycleWeeks:      plan.CycleWeeks,
				},
				CycleOffsetWeeks: plan.CycleOffsetWeeks,
				From:             plan.From,
				To:               plan.To,
			})
		}
	}
	for idx := range ev.sheets {
		if in, ok := inputs[ev.sheets[idx].WorkerID]; ok {
			in.Threshold = ev.sheets[idx].Threshold
		}
	}
	for idx := range ev.entries {
		entry := &ev.entries[idx]
		if in, ok := inputs[entry.WorkerID]; ok {
			in.Entries = append(in.Entries, timesheet.TimeClockEntry{
				ClockedInAt:  entry.ClockedInAt,
				ClockedOutAt: entry.ClockedOutAt,
				BreakMinutes: entry.BreakMinutes,
			})
		}
	}
	for idx := range ev.away {
		span := &ev.away[idx]
		if in, ok := inputs[span.WorkerID]; ok {
			in.Away = append(in.Away, weekhours.Span{From: span.From, To: span.To})
		}
	}

	out := make(map[pulid.ID]*weekhoursports.WorkerWeek, len(ev.heads))
	for idx := range ev.heads {
		head := &ev.heads[idx]
		out[head.ID] = &weekhoursports.WorkerWeek{
			WorkerID: head.ID,
			Name:     strings.TrimSpace(head.Name),
			Status:   head.Status,
			Week:     weekhours.Project(inputs[head.ID]),
		}
	}

	return out
}

func rotaQuery(
	win window,
	managerIDs []pulid.ID,
) (keel.Query[*workerschedule.WorkerShiftAssignment], keel.Int64Expr) {
	sa := workerscheduledb.WorkerShiftAssignments
	st := workerscheduledb.ShiftTemplates
	w := workerdb.Workers
	ts := timesheetdb.Timesheets

	key := keel.Int64Of(keel.Raw(
		"coalesce(? + ?, 0) + ? * (7 - length(replace(?, '1', ''))) - coalesce(?, ?)",
		ts.RegularMinutes, ts.OvertimeMinutes, st.DurationMinutes, st.DaysOfWeek,
		ts.OvertimeThresholdMinutes, timesheet.DefaultOvertimeThresholdMinutes,
	))
	query := sa.Join(st, st.ID.EqCol(sa.ShiftTemplateID)).
		Join(w, w.ID.EqCol(sa.WorkerID)).
		LeftJoin(ts, ts.WorkerID.EqCol(sa.WorkerID), ts.PeriodStart.Eq(win.period)).
		Where(
			sa.EffectiveFrom.Lt(win.end),
			keel.Or(sa.EffectiveTo.IsNull(), sa.EffectiveTo.Gte(win.start)),
			w.Status.Eq(domaintypes.StatusActive),
			workerteam.Where(w.ID.Col, managerIDs),
		)

	return query, key
}

func (r *repository) Leading(
	ctx context.Context,
	req *weekhoursports.LeadingRequest,
) (*weekhoursports.Leading, error) {
	workersReq := &weekhoursports.WorkersRequest{
		Tenant:     req.Tenant,
		ManagerIDs: req.ManagerIDs,
		Now:        req.Now,
		Location:   req.Location,
	}
	win := windowOf(req.Now, workersReq)
	topLimit := weekhoursports.MaxWorkers - roomCandidates

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*weekhoursports.Leading, error) {
		var (
			top    []rankRow
			low    []rankRow
			counts []countRow
		)
		topQuery, key := rotaQuery(win, req.ManagerIDs)
		lowQuery, _ := rotaQuery(win, req.ManagerIDs)
		countQuery, _ := rotaQuery(win, req.ManagerIDs)
		sa := workerscheduledb.WorkerShiftAssignments
		if err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoSlice(topQuery.Pick(sa.WorkerID, key).
				Order(key.Desc(), sa.WorkerID.Asc()).
				Limit(topLimit), &top),
			keel.IntoSlice(lowQuery.Pick(sa.WorkerID, key).
				Order(key.Asc(), sa.WorkerID.Asc()).
				Limit(roomCandidates), &low),
			keel.IntoSlice(countQuery.Pick(keel.CountOf(sa.WorkerID).Distinct()), &counts),
		); err != nil {
			return nil, fmt.Errorf("read who is on this week's rota: %w", err)
		}

		ids := make([]pulid.ID, 0, len(top)+len(low))
		for _, row := range slices.Concat(top, low) {
			ids = append(ids, row.WorkerID)
		}
		workersReq.WorkerIDs = ids
		weeks, err := r.Workers(ctx, workersReq)
		if err != nil {
			return nil, err
		}

		out := &weekhoursports.Leading{Start: win.start, End: win.end}
		if len(counts) > 0 {
			out.OnRota = counts[0].Workers
		}
		out.Exact = len(top) < topLimit || top[len(top)-1].Key < -rankSlackMinutes

		return rank(out, weeks, req), nil
	})
}

func rank(
	out *weekhoursports.Leading,
	weeks map[pulid.ID]*weekhoursports.WorkerWeek,
	req *weekhoursports.LeadingRequest,
) *weekhoursports.Leading {
	all := make([]*weekhoursports.WorkerWeek, 0, len(weeks))
	for _, week := range weeks {
		if week.Week.Planned() == 0 && week.Week.Shift == "" {
			continue
		}
		all = append(all, week)
		if week.Week.Over() {
			out.Over++
		}
		if week.Week.OnLine() {
			out.OnLine++
		}
	}
	slices.SortFunc(all, func(a, b *weekhoursports.WorkerWeek) int {
		ah := a.Week.Planned() - a.Week.Threshold
		bh := b.Week.Planned() - b.Week.Threshold
		if ah != bh {
			return int(bh - ah)
		}

		return strings.Compare(a.Name, b.Name)
	})

	limit := min(max(req.Limit, 1), weekhoursports.MaxLeading)
	room := min(max(req.Room, 0), limit-1)
	leadCount := min(len(all), limit-room)
	out.Rows = all[:leadCount]
	rest := slices.Clone(all[leadCount:])
	slices.Reverse(rest)
	for _, week := range rest {
		if len(out.Room) >= room {
			break
		}
		if week.Week.Room() > 0 {
			out.Room = append(out.Room, week)
		}
	}
	return out
}
