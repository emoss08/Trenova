package volume

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/seeds/volume/adjustmentvolume"
	"github.com/emoss08/trenova/internal/seeds/volume/billingvolume"
	"github.com/emoss08/trenova/internal/seeds/volume/capturevolume"
	"github.com/emoss08/trenova/internal/seeds/volume/cashvolume"
	"github.com/emoss08/trenova/internal/seeds/volume/factoringvolume"
	"github.com/emoss08/trenova/internal/seeds/volume/fuelvolume"
	"github.com/emoss08/trenova/internal/seeds/volume/masterdatavolume"
	"github.com/emoss08/trenova/internal/seeds/volume/payrollvolume"
	"github.com/emoss08/trenova/internal/seeds/volume/servicefailurevolume"
	"github.com/emoss08/trenova/internal/seeds/volume/settlementvolume"
	"github.com/emoss08/trenova/internal/seeds/volume/shipmentvolume"
	"github.com/emoss08/trenova/internal/seeds/volume/stopetavolume"
	"github.com/emoss08/trenova/internal/seeds/volume/telematicsvolume"
	"github.com/emoss08/trenova/internal/seeds/volume/tendervolume"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/pkg/dbscope"
)

const (
	systemReason = "find the organization a volume data set fills"
)

func Run(ctx context.Context, db *keel.DB, opts volumegen.Options) (*volumegen.Report, error) {
	if err := opts.Normalize(); err != nil {
		return nil, err
	}

	now := time.Now().UTC().Truncate(time.Second)
	g := &volumegen.Generator{
		DB:      db,
		Opts:    opts,
		Scale:   volumegen.ScaleFor(opts.Shipments),
		Fake:    volumegen.NewFaker(opts.Seed),
		Report:  &volumegen.Report{Seed: opts.Seed},
		Now:     now,
		Start:   now.AddDate(0, -opts.HistoryMonths, 0),
		Numbers: volumegen.NewNumberer(db),
	}

	began := time.Now()
	refs, err := volumegen.LoadTenantRefs(
		dbscope.WithSystem(ctx, systemReason),
		db,
		opts.OrganizationID,
	)
	if err != nil {
		return nil, err
	}
	g.Refs = refs
	g.Numbers.OrgID = refs.OrgID
	g.Numbers.BuID = refs.BuID
	g.Numbers.BuCode = refs.BuCode
	ctx = dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: refs.OrgID,
		BusinessUnitID: refs.BuID,
	})
	if err = volumegen.LoadTenantData(ctx, db, refs); err != nil {
		return nil, err
	}

	phases := []struct {
		name string
		run  func(context.Context, *volumegen.Generator) error
	}{
		{name: "master data", run: masterdatavolume.Seed},
		{name: "shipments", run: shipmentvolume.Seed},
		{name: "billing", run: billingvolume.Seed},
		{name: "invoice adjustments", run: adjustmentvolume.Seed},
		{name: "factoring", run: factoringvolume.Seed},
		{name: "carrier settlements", run: settlementvolume.Seed},
		{name: "driver payroll", run: payrollvolume.Seed},
		{name: "fuel and IFTA", run: fuelvolume.Seed},
		{name: "cash, collections and accounting", run: cashvolume.Seed},
		{name: "service failures", run: servicefailurevolume.Seed},
		{name: "carrier tenders", run: tendervolume.Seed},
		{name: "telematics", run: telematicsvolume.Seed},
		{name: "stop ETAs", run: stopetavolume.Seed},
		{name: "document capture", run: capturevolume.Seed},
	}
	for _, phase := range phases {
		phaseStart := time.Now()
		g.Opts.Logf("→ Seeding %s", phase.name)
		if err = phase.run(ctx, g); err != nil {
			return g.Report, fmt.Errorf("seed %s: %w", phase.name, err)
		}
		g.Opts.Logf("✓ Seeded %s in %s", phase.name, time.Since(phaseStart).Round(time.Millisecond))
	}

	g.Report.Duration = time.Since(began)

	return g.Report, nil
}
