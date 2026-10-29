package pricingfixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/rules"
)

type checkedSeedSource struct {
	check     *rules.TariffCheck
	testCount int32
}

func checkSeedSource(
	ctx context.Context,
	kind rules.TariffKind,
	name, source string,
	tariffCtx *rules.TariffContext,
) (*checkedSeedSource, error) {
	if tariffCtx == nil {
		tariffCtx = &rules.TariffContext{}
	}
	check, err := rules.CheckTariff(ctx, &rules.TariffCheckRequest{
		Source:  source,
		Kind:    kind,
		Context: tariffCtx,
	})
	if err != nil {
		return nil, fmt.Errorf("check %s: %w", name, err)
	}
	if finding := check.FirstError(); !check.OK || finding != nil {
		message := "it does not check"
		if finding != nil {
			message = finding.Message.String()
		}
		return nil, fmt.Errorf("%s does not check: %s", name, message)
	}

	run, err := rules.TestTariff(ctx, &rules.TariffTestRequest{Source: source, Context: tariffCtx})
	if err != nil {
		return nil, fmt.Errorf("test %s: %w", name, err)
	}
	if !run.Passed() {
		return nil, fmt.Errorf("%s does not pass its own tests", name)
	}

	return &checkedSeedSource{
		check:     check,
		testCount: int32(len(run.Tests)),
	}, nil //nolint:gosec // a file holds far fewer tests than an int32
}
