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
	Summary: "slide the freight a volume data set left open forward to now, so loads in " +
		"transit when it was seeded are in transit today and drivers are busy until their drops",
	Run: runCommand,
}

type Result struct {
	Shift       int64
	Shipments   int64
	Stops       int64
	Assignments int64
}

func runCommand(ctx context.Context, env *keelcli.Env, args []string) error {
	fs := env.FlagSet("roll-forward")
	org := fs.String("org", "", "organization to roll forward (default: the development organization)")
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
	if result.Shift == 0 {
		_, _ = fmt.Fprintln(env.Stdout, "The open freight is already current; nothing rolled.")
		return nil
	}
	_, _ = fmt.Fprintf(
		env.Stdout,
		"Rolled forward %s: %d shipments, %d stops, %d driver assignments\n",
		time.Duration(result.Shift)*time.Second,
		result.Shipments,
		result.Stops,
		result.Assignments,
	)

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

	sp := shipmentdb.Shipments
	latest, err := sp.Order(sp.CreatedAt.Desc()).First(ctx, db)
	if errors.Is(err, keel.ErrNotFound) {
		return &Result{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find the newest shipment: %w", err)
	}
	shift := now - latest.CreatedAt
	if shift < minShift {
		return &Result{}, nil
	}

	result := &Result{Shift: shift}
	err = keel.Write(ctx, db, func(ctx context.Context) error {
		return roll(ctx, db, shift, result)
	})
	if err != nil {
		return nil, err
	}

	return result, nil
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
