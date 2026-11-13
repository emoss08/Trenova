package demandforecastrepository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/driverstart"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverready"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave/workerleavedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule/workerscheduledb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxDriverStarts = 10_000
	millisPerSecond = int64(1000)
)

type bookedRow struct {
	CustomerID pulid.ID
	Latitude   float64
	Longitude  float64
	SlotStart  int64
	Total      int64
	Open       int64
}

func (r *repository) ListBooked(
	ctx context.Context,
	req *demandforecastports.BookedRequest,
) ([]demandforecastports.BookedSlot, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]demandforecastports.BookedSlot, error) {
			l := loadsBetween(req.From, req.Until)
			loc := locationdb.Locations.As("bk_loc")
			slotStart := l.st.ScheduledWindowStart
			open := keel.Count().Filter(l.sm.Status.Eq(shipmentvocab.MoveStatusNew))

			rows, err := keel.Into[bookedRow](ctx, r.db.Keel(ctx), l.q.
				Join(loc, loc.ID.EqCol(l.st.LocationID)).
				Pick(text(l.sp.CustomerID), loc.Latitude, loc.Longitude, slotStart, keel.Count(), open).
				Where(loc.Latitude.IsNotNull(), loc.Longitude.IsNotNull()).
				Group(l.sp.CustomerID, loc.Latitude, loc.Longitude, slotStart))
			if err != nil {
				return nil, fmt.Errorf("read booked loads: %w", err)
			}

			out := make([]demandforecastports.BookedSlot, 0, len(rows))
			for _, row := range rows {
				out = append(out, demandforecastports.BookedSlot(row))
			}

			return out, nil
		},
	)
}

type startRow struct {
	WorkerID        pulid.ID
	FirstName       string
	LastName        string
	Latitude        *float64
	Longitude       *float64
	FreeAt          int64
	PositioningID   *string
	ClocksAt        *int64
	DriveMs         *int64
	ShiftMs         *int64
	CycleMs         *int64
	CycleTomorrowMs *int64
	TimeOffUntil    *int64
	LeaveFrom       *int64
	LeaveUntil      *int64
	ShiftDays       *string
	ShiftCycleWeeks *int16
	ShiftStart      *int16
	ShiftOffset     *int16
}

func (r *startRow) onOpenLeave() bool {
	return r.LeaveFrom != nil && r.LeaveUntil == nil
}

func (r *startRow) readyAt(now int64, location *time.Location) int64 {
	in := driverready.Input{Now: now, FreeAt: r.FreeAt, Location: location}
	if r.TimeOffUntil != nil {
		in.TimeOffUntil = *r.TimeOffUntil
	}
	if r.LeaveUntil != nil {
		in.TimeOffUntil = max(in.TimeOffUntil, *r.LeaveUntil)
	}
	if r.ShiftDays != nil {
		in.Pattern = &driverready.Pattern{Template: &workerschedule.ShiftTemplate{
			DaysOfWeek: *r.ShiftDays,
		}}
		if r.ShiftCycleWeeks != nil {
			in.Pattern.Template.CycleWeeks = *r.ShiftCycleWeeks
		}
		if r.ShiftStart != nil {
			in.Pattern.Template.StartMinute = *r.ShiftStart
		}
		if r.ShiftOffset != nil {
			in.Pattern.CycleOffsetWeeks = *r.ShiftOffset
		}
	}
	if r.ClocksAt != nil && r.DriveMs != nil && r.ShiftMs != nil && r.CycleMs != nil {
		in.Clocks = &driverready.Clocks{
			RecordedAt:   *r.ClocksAt,
			DriveSeconds: *r.DriveMs / millisPerSecond,
			ShiftSeconds: *r.ShiftMs / millisPerSecond,
			CycleSeconds: *r.CycleMs / millisPerSecond,
		}
		if r.CycleTomorrowMs != nil {
			in.Clocks.CycleTomorrowSeconds = *r.CycleTomorrowMs / millisPerSecond
		}
	}

	return driverready.At(in)
}

func (r *repository) ListDriverStarts(
	ctx context.Context,
	req *demandforecastports.DriverStartsRequest,
) ([]demandforecastports.DriverStart, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]demandforecastports.DriverStart, error) {
			w := workerdb.Workers
			wt := tractordb.Tractors.As("ds_trac")
			primary := wt.
				Pick(wt.ID).
				Where(wt.PrimaryWorkerID.EqCol(w.ID)).
				Order(keel.Desc(wt.Status.Eq(domaintypes.EquipmentStatusAvailable)), wt.Code.Asc()).
				Limit(1)

			q, start := driverstart.Join(w.LeftJoinLateral(primary), w, wt, req.Now)
			hos := telematicsdb.WorkerHOSStates.As("ds_hos")
			pto := workerdb.WorkerPTOs.As("ds_pto")
			off := pto.
				Pick(pto.EndDate).
				Where(
					pto.WorkerID.EqCol(w.ID),
					pto.Status.Eq(worker.PTOStatusApproved),
					start.FreeAt.GteExpr(pto.StartDate),
					start.FreeAt.LtExpr(pto.EndDate),
				).
				Order(pto.EndDate.Desc()).
				Limit(1)
			leave := workerleavedb.WorkerLeaveCases.As("ds_leave")
			away := leave.
				Pick(leave.StartsAt, leave.EndsAt).
				Where(
					leave.WorkerID.EqCol(w.ID),
					leave.Status.Eq(workerleave.LeaveCaseApproved),
					start.FreeAt.GteExpr(leave.StartsAt),
					keel.Or(leave.EndsAt.IsNull(), start.FreeAt.LtExpr(leave.EndsAt)),
				).
				Order(keel.Desc(leave.EndsAt).NullsFirst()).
				Limit(1)
			shift := workerscheduledb.WorkerShiftAssignments.As("ds_shift")
			pattern := workerscheduledb.ShiftTemplates.As("ds_pattern")
			rostered := shift.
				Pick(shift.ShiftTemplateID, shift.CycleOffsetWeeks).
				Where(
					shift.WorkerID.EqCol(w.ID),
					shift.EffectiveFrom.Lte(req.Now),
					keel.Or(shift.EffectiveTo.IsNull(), shift.EffectiveTo.Gte(req.Now)),
				).
				Order(shift.EffectiveFrom.Desc()).
				Limit(1)
			rows, err := keel.Into[startRow](ctx, r.db.Keel(ctx), q.
				LeftJoin(hos,
					hos.WorkerID.EqCol(w.ID),
					hos.RecordedAt.Gte(req.Now-driverready.ClocksMaxAgeSeconds),
				).
				LeftJoinLateral(off).
				LeftJoinLateral(away).
				LeftJoinLateral(rostered).
				LeftJoin(pattern, pattern.ID.EqCol(shift.ShiftTemplateID)).
				Pick(
					w.ID,
					w.FirstName,
					w.LastName,
					start.Latitude,
					start.Longitude,
					start.FreeAt,
					keel.StringOf(start.PositioningID),
					hos.RecordedAt,
					hos.DriveRemainingMs,
					hos.ShiftRemainingMs,
					hos.CycleRemainingMs,
					hos.CycleTomorrowMs,
					pto.EndDate,
					leave.StartsAt,
					leave.EndsAt,
					pattern.DaysOfWeek,
					pattern.CycleWeeks,
					pattern.StartMinute,
					shift.CycleOffsetWeeks,
				).
				Where(
					keel.When(len(req.WorkerIDs) > 0, w.ID.In(req.WorkerIDs...)),
					w.Status.Eq(domaintypes.StatusActive),
					w.CanBeAssigned.IsTrue(),
					w.AvailableForDispatch.IsTrue(),
				).
				Order(w.ID.Asc()).
				Limit(maxDriverStarts))
			if err != nil {
				return nil, fmt.Errorf("read driver starts: %w", err)
			}

			out := make([]demandforecastports.DriverStart, 0, len(rows))
			for _, row := range rows {
				if row.Latitude == nil || row.Longitude == nil || row.onOpenLeave() {
					continue
				}
				start := demandforecastports.DriverStart{
					WorkerID:  row.WorkerID,
					Name:      strings.TrimSpace(row.FirstName + " " + row.LastName),
					Latitude:  *row.Latitude,
					Longitude: *row.Longitude,
					FreeAt:    row.readyAt(req.Now, req.Location),
				}
				if row.PositioningID != nil {
					start.PositioningID = pulid.ID(*row.PositioningID)
				}
				out = append(out, start)
			}

			return out, nil
		},
	)
}

type customerNameRow struct {
	ID   pulid.ID
	Name string
}

func (r *repository) CustomerNames(
	ctx context.Context,
	req *demandforecastports.CustomerNamesRequest,
) (map[pulid.ID]string, error) {
	if len(req.CustomerIDs) == 0 {
		return map[pulid.ID]string{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		c := customerdb.Customers
		rows, err := keel.Into[customerNameRow](ctx, r.db.Keel(ctx), c.
			Pick(c.ID, c.Name).
			Where(c.ID.In(req.CustomerIDs...)).
			Limit(len(req.CustomerIDs)))
		if err != nil {
			return nil, fmt.Errorf("read customer names: %w", err)
		}
		names := make(map[pulid.ID]string, len(rows))
		for _, row := range rows {
			names[row.ID] = row.Name
		}

		return names, nil
	})
}
