// Package volumeseed fills one organization with a large, varied, internally
// consistent data set so the product can be exercised at realistic volume:
// list performance, pagination, filters, loading states and dashboards.
//
// It is never part of the registered seeds. It runs only from
// `trenova db seed-volume`, refuses production and staging, and adds to
// whatever the development seeds created, so it can be run more than once to
// grow the data set. Shipment, order and invoice numbers come from the
// tenant's real sequence formats, exactly as the application issues them.
package volumeseed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	DefaultShipments     = 10_000
	DefaultHistoryMonths = 12
	DefaultBatchSize     = 500
	MaxShipments         = 250_000
)

var (
	ErrShipmentCount = errors.New("shipment count must be between 1 and 250000")
	ErrHistoryMonths = errors.New("history must be between 1 and 36 months")
)

type Options struct {
	Shipments      int
	HistoryMonths  int
	Seed           uint64
	OrganizationID pulid.ID
	BatchSize      int
	Logf           func(format string, args ...any)
}

func (o *Options) normalize() error {
	if o.Shipments == 0 {
		o.Shipments = DefaultShipments
	}
	if o.Shipments < 1 || o.Shipments > MaxShipments {
		return ErrShipmentCount
	}
	if o.HistoryMonths == 0 {
		o.HistoryMonths = DefaultHistoryMonths
	}
	if o.HistoryMonths < 1 || o.HistoryMonths > 36 {
		return ErrHistoryMonths
	}
	if o.BatchSize <= 0 {
		o.BatchSize = DefaultBatchSize
	}
	if o.Seed == 0 {
		o.Seed = uint64(time.Now().UnixNano()) //nolint:gosec // any seed is fine
	}
	if o.Logf == nil {
		o.Logf = func(string, ...any) {}
	}

	return nil
}

type scale struct {
	customers   int
	locations   int
	drivers     int
	tractors    int
	trailers    int
	carriers    int
	fleetCodes  int
	commentRate float64
}

func scaleFor(shipments int) scale {
	return scale{
		customers:   clamp(shipments/40, 40, 5_000),
		locations:   clamp(shipments/8, 150, 25_000),
		drivers:     clamp(shipments/25, 30, 10_000),
		tractors:    clamp(shipments*9/250, 25, 9_000),
		trailers:    clamp(shipments*3/50, 40, 15_000),
		carriers:    clamp(shipments/80, 20, 3_000),
		fleetCodes:  clamp(shipments/800, 6, 40),
		commentRate: 2.4,
	}
}

type Report struct {
	Seed     uint64
	Counts   []Count
	Duration time.Duration
}

type Count struct {
	Entity string
	Rows   int
}

func (r *Report) add(entity string, rows int) {
	for i := range r.Counts {
		if r.Counts[i].Entity == entity {
			r.Counts[i].Rows += rows
			return
		}
	}
	r.Counts = append(r.Counts, Count{Entity: entity, Rows: rows})
}

type generator struct {
	db      *bun.DB
	opts    Options
	scale   scale
	fake    *faker
	numbers *numberer
	refs    *tenantRefs
	report  *Report
	now     time.Time
	start   time.Time

	customers   []*customerRecord
	locations   []*locationRecord
	commodities []*commodityRecord
	drivers     []*driverRecord
	crews       []*crew
	carriers    []pulid.ID
	shipments   []*shipmentRecord
	payments    []*customerpayment.Payment
	factoring   *facState
}

func Run(ctx context.Context, db *bun.DB, opts Options) (*Report, error) {
	if err := opts.normalize(); err != nil {
		return nil, err
	}

	conn := postgres.WrapDB(db)
	logger := zap.NewNop()
	now := time.Now().UTC().Truncate(time.Second)
	g := &generator{
		db:     db,
		opts:   opts,
		scale:  scaleFor(opts.Shipments),
		fake:   newFaker(opts.Seed),
		report: &Report{Seed: opts.Seed},
		now:    now,
		start:  now.AddDate(0, -opts.HistoryMonths, 0),
		numbers: &numberer{
			generator: seqgen.NewGenerator(seqgen.GeneratorParams{
				Store: seqgen.NewSequenceStore(
					seqgen.SequenceStoreParams{DB: conn, Logger: logger},
				),
				Provider: seqgen.NewFormatProvider(
					seqgen.FormatProviderParams{DB: conn, Logger: logger},
				),
				Logger: logger,
			}),
			formats: seqgen.NewFormatProvider(
				seqgen.FormatProviderParams{DB: conn, Logger: logger},
			),
		},
	}

	began := time.Now()
	refs, err := loadTenantRefs(ctx, db, opts.OrganizationID)
	if err != nil {
		return nil, err
	}
	g.refs = refs
	g.numbers.orgID = refs.orgID
	g.numbers.buID = refs.buID
	g.numbers.buCode = refs.buCode

	phases := []struct {
		name string
		run  func(context.Context) error
	}{
		{name: "master data", run: g.seedMasterData},
		{name: "shipments", run: g.seedShipments},
		{name: "billing", run: g.seedBilling},
		{name: "invoice adjustments", run: g.seedAdjustments},
		{name: "factoring", run: g.seedFactoring},
		{name: "carrier settlements", run: g.seedCarrierSettlements},
		{name: "driver payroll", run: g.seedPayroll},
		{name: "fuel and IFTA", run: g.seedFuel},
		{name: "cash, collections and accounting", run: g.seedCash},
	}
	for _, phase := range phases {
		phaseStart := time.Now()
		g.opts.Logf("→ Seeding %s", phase.name)
		if err = phase.run(ctx); err != nil {
			return g.report, fmt.Errorf("seed %s: %w", phase.name, err)
		}
		g.opts.Logf("✓ Seeded %s in %s", phase.name, time.Since(phaseStart).Round(time.Millisecond))
	}

	g.report.Duration = time.Since(began)

	return g.report, nil
}

func clamp(value, low, high int) int {
	return max(low, min(value, high))
}
