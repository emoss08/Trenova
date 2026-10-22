package tariff

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
)

const MaxTestNames = 200

type TestRequest struct {
	Source  string
	Context *rules.TariffContext
	Names   []string
}

func (c *Checker) RunTests(ctx context.Context, req *TestRequest) (*rules.TariffTestRun, error) {
	run, err := c.module.TestTariff(ctx, &rules.TariffTestRequest{
		Source:  req.Source,
		Context: req.Context,
		Names:   req.Names,
	})
	if err != nil {
		return nil, fmt.Errorf("run tariff tests: %w", err)
	}
	if run.Tests == nil {
		run.Tests = []rules.TariffTestResult{}
	}
	return run, nil
}

func (c *Checker) TestChecked(
	ctx context.Context,
	checked *SourceCheck,
	source string,
) (*rules.TariffTestRun, error) {
	if !checked.OK() {
		return &rules.TariffTestRun{Tests: []rules.TariffTestResult{}}, nil
	}
	return c.RunTests(ctx, &TestRequest{Source: source, Context: checked.Context})
}

const MaxNeedPasses = 4

var ErrNeedsUnmet = errors.New("the module still needed table cells after every pass")

type ScaffoldRequest struct {
	TenantInfo pagination.TenantInfo
	Source     string
	Context    *rules.TariffContext
	Record     rules.RawJSON
	Inputs     map[string]rules.RawJSON
	Name       string
}

func (c *Checker) ScaffoldTest(
	ctx context.Context,
	req *ScaffoldRequest,
) (*rules.TariffScaffold, error) {
	for pass := 0; ; pass++ {
		scaffold, err := c.module.ScaffoldTariffTest(ctx, &rules.TariffScaffoldRequest{
			Source:  req.Source,
			Context: req.Context,
			Record:  req.Record,
			Inputs:  req.Inputs,
			Name:    req.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("scaffold tariff test: %w", err)
		}
		if len(scaffold.Needs) == 0 && scaffold.Status != rules.TariffNeeds {
			return scaffold, nil
		}
		if pass >= MaxNeedPasses {
			return nil, fmt.Errorf("scaffold tariff test: %w", ErrNeedsUnmet)
		}
		if err = c.context.LoadCells(ctx, req.TenantInfo, req.Context, scaffold.Needs); err != nil {
			return nil, fmt.Errorf("load table cells for a scaffold: %w", err)
		}
	}
}
