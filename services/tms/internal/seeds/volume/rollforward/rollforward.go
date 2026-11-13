package rollforward

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	systemReason = "find the organization whose open freight rolls forward"
	minShift     = int64(time.Hour / time.Second)
)

var ErrEnvironment = errors.New(
	"roll-forward runs only against a database marked development or test",
)

var openStatuses = []shipmentvocab.Status{
	shipmentvocab.StatusNew,
	shipmentvocab.StatusPartiallyAssigned,
	shipmentvocab.StatusAssigned,
	shipmentvocab.StatusInTransit,
	shipmentvocab.StatusDelayed,
	shipmentvocab.StatusPartiallyCompleted,
}

var Command = keelcli.Command{
	Name:  "roll-forward",
	Usage: "[-org id]",
	Summary: "slide the freight a volume data set left open, and the telematics feed it " +
		"stopped at, forward to now, so loads in transit when it was seeded are in transit " +
		"today, drivers are busy until their drops and their clocks and positions are current",
	Run: runCommand,
}

type Result struct {
	Shift       int64
	Shipments   int64
	Stops       int64
	Assignments int64

	TelematicsShift int64
	Clocks          int64
	Positions       int64
	DutyLogs        int64
	Violations      int64
}

func (r *Result) Rolled() bool {
	return r.Shift > 0 || r.TelematicsShift > 0
}

func runCommand(ctx context.Context, env *keelcli.Env, args []string) error {
	fs := env.FlagSet("roll-forward")
	org := fs.String(
		"org",
		"",
		"organization to roll forward (default: the development organization)",
	)
	if err := env.Parse(fs, args); err != nil {
		return err
	}

	orgID := pulid.Nil
	if *org != "" {
		parsed, err := pulid.Parse(*org)
		if err != nil {
			return keelcli.Usagef("roll-forward: -org %q: %v", *org, err)
		}
		orgID = parsed
	}

	db, err := env.DB(ctx)
	if err != nil {
		return err
	}
	mark, err := seed.Environment(ctx, db)
	if err != nil {
		return fmt.Errorf("read the database environment: %w", err)
	}
	if mark != seed.Development && mark != seed.Test {
		return ErrEnvironment
	}

	result, err := Run(ctx, db, orgID, time.Now().Unix())
	if err != nil {
		return err
	}
	if !result.Rolled() {
		_, _ = fmt.Fprintln(
			env.Stdout,
			"The open freight and the telematics feed are already current; nothing rolled.",
		)
		return nil
	}
	if result.Shift > 0 {
		_, _ = fmt.Fprintf(
			env.Stdout,
			"Rolled freight forward %s: %d shipments, %d stops, %d driver assignments\n",
			time.Duration(result.Shift)*time.Second,
			result.Shipments,
			result.Stops,
			result.Assignments,
		)
	}
	if result.TelematicsShift > 0 {
		_, _ = fmt.Fprintf(
			env.Stdout,
			"Rolled telematics forward %s: %d hours-of-service clocks, %d vehicle positions, "+
				"%d duty logs, %d violations\n",
			time.Duration(result.TelematicsShift)*time.Second,
			result.Clocks,
			result.Positions,
			result.DutyLogs,
			result.Violations,
		)
	}

	return nil
}

func Run(ctx context.Context, db *keel.DB, orgID pulid.ID, now int64) (*Result, error) {
	refs, err := volumegen.LoadTenantRefs(dbscope.WithSystem(ctx, systemReason), db, orgID)
	if err != nil {
		return nil, err
	}
	ctx = dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: refs.OrgID,
		BusinessUnitID: refs.BuID,
	})

	result := &Result{}
	if result.Shift, err = freightShift(ctx, db, now); err != nil {
		return nil, err
	}
	if result.TelematicsShift, err = telematicsShift(ctx, db, now); err != nil {
		return nil, err
	}
	if !result.Rolled() {
		return result, nil
	}

	err = keel.Write(ctx, db, func(ctx context.Context) error {
		if result.Shift > 0 {
			if err := roll(ctx, db, result.Shift, result); err != nil {
				return err
			}
		}
		if result.TelematicsShift > 0 {
			return rollTelematics(ctx, db, result.TelematicsShift, result)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func freightShift(ctx context.Context, db *keel.DB, now int64) (int64, error) {
	sp := shipmentdb.Shipments
	latest, err := sp.Order(sp.CreatedAt.Desc()).First(ctx, db)
	if errors.Is(err, keel.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("find the newest shipment: %w", err)
	}

	return behind(now, latest.CreatedAt), nil
}

func telematicsShift(ctx context.Context, db *keel.DB, now int64) (int64, error) {
	hos := telematicsdb.WorkerHOSStates
	latest, err := hos.Order(hos.RecordedAt.Desc()).First(ctx, db)
	if errors.Is(err, keel.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("find the newest hours-of-service clock: %w", err)
	}

	return behind(now, latest.RecordedAt), nil
}

func behind(now, latest int64) int64 {
	if shift := now - latest; shift >= minShift {
		return shift
	}

	return 0
}

func rollTelematics(ctx context.Context, db *keel.DB, shift int64, result *Result) error {
	hos, pos := telematicsdb.WorkerHOSStates, telematicsdb.VehiclePositions
	logs, violations := telematicsdb.WorkerHOSLogs, telematicsdb.WorkerHOSViolations
	by := keel.Int64(shift)
	var err error

	result.Clocks, err = hos.Where(keel.True()).Update(ctx, db,
		hos.RecordedAt.ToExpr(hos.RecordedAt.Add(by)),
		hos.ReceivedAt.ToExpr(hos.ReceivedAt.Add(by)),
		hos.CycleStartedAt.ToExpr(hos.CycleStartedAt.Add(by)),
	)
	if err != nil {
		return fmt.Errorf("roll hours-of-service clocks forward: %w", err)
	}
	result.Positions, err = pos.Where(keel.True()).Update(ctx, db,
		pos.RecordedAt.ToExpr(pos.RecordedAt.Add(by)),
		pos.ReceivedAt.ToExpr(pos.ReceivedAt.Add(by)),
	)
	if err != nil {
		return fmt.Errorf("roll vehicle positions forward: %w", err)
	}
	result.DutyLogs, err = logs.Where(keel.True()).Update(ctx, db,
		logs.LogStartAt.ToExpr(logs.LogStartAt.Add(by)),
		logs.LogEndAt.ToExpr(logs.LogEndAt.Add(by)),
		logs.ReceivedAt.ToExpr(logs.ReceivedAt.Add(by)),
	)
	if err != nil {
		return fmt.Errorf("roll duty logs forward: %w", err)
	}
	result.Violations, err = violations.Where(keel.True()).Update(ctx, db,
		violations.ViolationStartAt.ToExpr(violations.ViolationStartAt.Add(by)),
		violations.DayStartAt.ToExpr(violations.DayStartAt.Add(by)),
		violations.DayEndAt.ToExpr(violations.DayEndAt.Add(by)),
		violations.DetectedAt.ToExpr(violations.DetectedAt.Add(by)),
	)
	if err != nil {
		return fmt.Errorf("roll hours-of-service violations forward: %w", err)
	}

	return nil
}

func roll(ctx context.Context, db *keel.DB, shift int64, result *Result) error {
	sp, sm, stp, asn := shipmentdb.Shipments, shipmentdb.ShipmentMoves, shipmentdb.Stops,
		shipmentdb.Assignments
	by := keel.Int64(shift)
	openShipments := sp.Where(sp.Status.In(openStatuses...)).Pick(sp.ID)
	openMoves := sm.Where(sm.ShipmentID.InQuery(openShipments)).Pick(sm.ID)

	stops, err := stp.Where(stp.ShipmentMoveID.InQuery(openMoves)).Update(ctx, db,
		stp.ScheduledWindowStart.ToExpr(stp.ScheduledWindowStart.Add(by)),
		stp.ScheduledWindowEnd.ToExpr(stp.ScheduledWindowEnd.Add(by)),
		stp.ActualArrival.ToExpr(stp.ActualArrival.Add(by)),
		stp.ActualDeparture.ToExpr(stp.ActualDeparture.Add(by)),
	)
	if err != nil {
		return fmt.Errorf("roll stops forward: %w", err)
	}
	assignments, err := asn.Where(asn.ShipmentMoveID.InQuery(openMoves)).Update(ctx, db,
		asn.CreatedAt.ToExpr(asn.CreatedAt.Add(by)),
	)
	if err != nil {
		return fmt.Errorf("roll assignments forward: %w", err)
	}
	shipments, err := sp.Where(sp.Status.In(openStatuses...)).Update(ctx, db,
		sp.ActualShipDate.ToExpr(sp.ActualShipDate.Add(by)),
		sp.ActualDeliveryDate.ToExpr(sp.ActualDeliveryDate.Add(by)),
		sp.CreatedAt.ToExpr(sp.CreatedAt.Add(by)),
	)
	if err != nil {
		return fmt.Errorf("roll shipments forward: %w", err)
	}

	result.Stops, result.Assignments, result.Shipments = stops, assignments, shipments
	return nil
}
