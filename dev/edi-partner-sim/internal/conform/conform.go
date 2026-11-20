package conform

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/sourcegraph/conc/pool"
)

const (
	DefaultWorkers = 2
	DefaultCount   = 20
	maxWorkers     = 16
)

type Options struct {
	Root     string
	Partners []string
	Seed     uint64
	Count    uint64
	Workers  int
	Kit      CarrierKit
}

type runner struct {
	client *lading.Client
	kit    CarrierKit
}

func Run(ctx context.Context, options *Options) (*Report, error) {
	if err := options.normalize(); err != nil {
		return nil, err
	}
	defs, err := open(options)
	if err != nil {
		return nil, err
	}
	runtime := x12.New(wasmmodule.Options{PoolSize: options.Workers})
	defer func() { _ = runtime.Close(context.WithoutCancel(ctx)) }()
	r := &runner{client: lading.New(runtime), kit: options.Kit}
	report := &Report{Seed: options.Seed, Count: options.Count, Workers: options.Workers}
	var subjects []*subject
	for _, def := range defs {
		s := r.prepare(ctx, def)
		report.Partners = append(report.Partners, s.report)
		if s.ready() {
			subjects = append(subjects, s)
		}
	}
	work := pool.NewWithResults[*Case]().WithContext(ctx).WithMaxGoroutines(options.Workers)
	for _, s := range subjects {
		for offset := range options.Count {
			work.Go(func(ctx context.Context) (*Case, error) {
				return r.run(ctx, s, options.Seed+offset), nil
			})
		}
	}
	cases, err := work.Wait()
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	byPartner := make(map[string]*PartnerReport, len(report.Partners))
	for _, p := range report.Partners {
		byPartner[p.Name] = p
	}
	for _, c := range cases {
		byPartner[c.Partner].Cases = append(byPartner[c.Partner].Cases, c)
	}
	for _, p := range report.Partners {
		slices.SortFunc(p.Cases, func(a, b *Case) int { return cmp.Compare(a.Seed, b.Seed) })
	}
	report.tally()
	return report, nil
}

func (o *Options) normalize() error {
	if o.Root == "" {
		o.Root = "partners"
	}
	if o.Count == 0 {
		o.Count = DefaultCount
	}
	if o.Workers == 0 {
		o.Workers = DefaultWorkers
	}
	if o.Seed == 0 {
		o.Seed = 1
	}
	if o.Kit == nil {
		o.Kit = DefaultKit{}
	}
	if o.Workers < 1 || o.Workers > maxWorkers {
		return fmt.Errorf("workers %d is not between 1 and %d", o.Workers, maxWorkers)
	}
	return nil
}

func open(options *Options) ([]*partner.Definition, error) {
	if len(options.Partners) == 0 {
		all, err := partner.OpenAll(options.Root)
		if err != nil {
			return nil, err
		}
		if len(all) == 0 {
			return nil, fmt.Errorf("no partner under %s", options.Root)
		}
		return all, nil
	}
	picked := make([]*partner.Definition, 0, len(options.Partners))
	var errs []error
	for _, name := range options.Partners {
		def, err := partner.Open(filepath.Join(options.Root, name))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		picked = append(picked, def)
	}
	return picked, errors.Join(errs...)
}

func (r *runner) run(ctx context.Context, s *subject, seed uint64) *Case {
	c := &Case{Partner: s.name(), Seed: seed}
	defer func() {
		if recovered := recover(); recovered != nil {
			c.fail(c.step("harness", "Run the case", "", ""), &Failure{
				Source:  SourceHarness,
				Rule:    "panic",
				Message: fmt.Sprint(recovered),
			})
		}
	}()
	switch s.def.Partner.Role {
	case partner.RoleCarrier:
		r.carrier(ctx, s, c)
	case partner.RoleShipper:
		r.shipper(ctx, s, c)
	}
	return c
}
