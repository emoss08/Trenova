package aipricing

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/shopspring/decimal"
)

type AdvisorUsage struct {
	Model    string       `json:"model"`
	Calls    int          `json:"calls"`
	Tokens   TokenUsage   `json:"tokens"`
	Executor []TokenUsage `json:"executor,omitempty"`
}

func (u *AdvisorUsage) Used() bool {
	return u != nil && (u.Calls > 0 || u.Tokens != TokenUsage{})
}

func (u *AdvisorUsage) Cost(kind aiprovider.Kind) *decimal.Decimal {
	if !u.Used() {
		return nil
	}
	priced, ok := ListPriced(&aiprovider.Provider{Kind: kind}, u.Model)
	if !ok {
		return nil
	}

	return CostFor(priced, u.Tokens)
}

func (u *AdvisorUsage) splitsExecutor(summed TokenUsage) bool {
	if !u.Used() || len(u.Executor) == 0 {
		return false
	}
	var total TokenUsage
	for idx := range u.Executor {
		iteration := &u.Executor[idx]
		total.Input += iteration.Input
		total.Output += iteration.Output
		total.CacheRead += iteration.CacheRead
		total.CacheWrite += iteration.CacheWrite
	}

	return total == summed
}

func AdvisedCostForTask(
	p *aiprovider.Provider,
	t aitask.Task,
	usage TokenUsage,
	advisor *AdvisorUsage,
) *decimal.Decimal {
	if !t.Generates() {
		return InputCostFor(p, usage)
	}

	return AdvisedCostFor(p, usage, advisor)
}

func AdvisedCostFor(
	p *aiprovider.Provider,
	usage TokenUsage,
	advisor *AdvisorUsage,
) *decimal.Decimal {
	executor := executorCostFor(p, usage, advisor)
	if executor == nil || !advisor.Used() {
		return executor
	}
	advice := advisor.Cost(p.Kind)
	if advice == nil {
		return nil
	}
	total := executor.Add(*advice)

	return &total
}

func executorCostFor(
	p *aiprovider.Provider,
	usage TokenUsage,
	advisor *AdvisorUsage,
) *decimal.Decimal {
	if !p.Priced() || !advisor.splitsExecutor(usage) {
		return CostFor(p, usage)
	}
	total := decimal.Zero
	for idx := range advisor.Executor {
		total = total.Add(*CostFor(p, advisor.Executor[idx]))
	}

	return &total
}

func ListPriced(base *aiprovider.Provider, model string) (*aiprovider.Provider, bool) {
	card, ok := ListPrice(base.Kind, model)
	if !ok {
		return nil, false
	}
	input, output := card.InputCostPerMillion, card.OutputCostPerMillion
	read, write := card.CacheReadCostPerMillion(), card.CacheWriteCostPerMillion()
	priced := *base
	priced.Model = model
	priced.InputCostPerMillion = &input
	priced.OutputCostPerMillion = &output
	priced.CacheReadCostPerMillion = &read
	priced.CacheWriteCostPerMillion = &write

	return &priced, true
}
