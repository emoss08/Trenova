package aiprovider

import "github.com/shopspring/decimal"

type AdvisorUsage struct {
	Model    string       `json:"model"`
	Calls    int          `json:"calls"`
	Tokens   TokenUsage   `json:"tokens"`
	Executor []TokenUsage `json:"executor,omitempty"`
}

func (u *AdvisorUsage) Used() bool {
	return u != nil && (u.Calls > 0 || u.Tokens != TokenUsage{})
}

func (u *AdvisorUsage) Cost(kind Kind) *decimal.Decimal {
	if !u.Used() {
		return nil
	}
	priced, ok := ListPriced(&Provider{Kind: kind}, u.Model)
	if !ok {
		return nil
	}

	return priced.CostFor(u.Tokens)
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

func (p *Provider) AdvisedCostForTask(
	t Task,
	usage TokenUsage,
	advisor *AdvisorUsage,
) *decimal.Decimal {
	if !t.Generates() {
		return p.InputCostFor(usage)
	}

	return p.AdvisedCostFor(usage, advisor)
}

func (p *Provider) AdvisedCostFor(usage TokenUsage, advisor *AdvisorUsage) *decimal.Decimal {
	executor := p.executorCostFor(usage, advisor)
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

func (p *Provider) executorCostFor(usage TokenUsage, advisor *AdvisorUsage) *decimal.Decimal {
	if !p.Priced() || !advisor.splitsExecutor(usage) {
		return p.CostFor(usage)
	}
	total := decimal.Zero
	for idx := range advisor.Executor {
		total = total.Add(*p.CostFor(advisor.Executor[idx]))
	}

	return &total
}

func ListPriced(base *Provider, model string) (*Provider, bool) {
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
