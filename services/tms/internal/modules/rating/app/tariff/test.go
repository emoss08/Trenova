package tariff

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
)

type TestRequest struct {
	TenantInfo pagination.TenantInfo
	Source     string
	Context    *rules.TariffContext
	Names      []string
}

func (c *Checker) RunTests(ctx context.Context, req *TestRequest) (*rules.TariffTestRun, error) {
	loads := &cellLoad{contexts: c.context, tenant: req.TenantInfo, tariffCtx: req.Context}
	var run *rules.TariffTestRun
	err := loads.until(ctx, func() ([]rules.TariffNeed, error) {
		var testErr error
		run, testErr = c.module.TestTariff(ctx, &rules.TariffTestRequest{
			Source:  req.Source,
			Context: req.Context,
			Names:   req.Names,
		})
		if testErr != nil {
			return nil, fmt.Errorf("run tariff tests: %w", testErr)
		}
		return run.Needs, nil
	})
	if err != nil {
		return nil, fmt.Errorf("load table cells for tariff tests: %w", err)
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
	return c.RunTests(ctx, &TestRequest{
		TenantInfo: checked.TenantInfo,
		Source:     source,
		Context:    checked.Context,
	})
}

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
	loads := &cellLoad{contexts: c.context, tenant: req.TenantInfo, tariffCtx: req.Context}
	var scaffold *rules.TariffScaffold
	err := loads.until(ctx, func() ([]rules.TariffNeed, error) {
		var scaffoldErr error
		scaffold, scaffoldErr = c.module.ScaffoldTariffTest(ctx, &rules.TariffScaffoldRequest{
			Source:  req.Source,
			Context: req.Context,
			Record:  req.Record,
			Inputs:  req.Inputs,
			Name:    req.Name,
		})
		if scaffoldErr != nil {
			return nil, fmt.Errorf("scaffold tariff test: %w", scaffoldErr)
		}
		return scaffold.Needs, nil
	})
	if err != nil {
		return nil, fmt.Errorf("load table cells for a scaffold: %w", err)
	}
	return scaffold, nil
}
