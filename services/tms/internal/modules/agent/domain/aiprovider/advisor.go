package aiprovider

import "github.com/shopspring/decimal"

type AdvisorUsage struct {
	Model  string     `json:"model"`
	Calls  int        `json:"calls"`
	Tokens TokenUsage `json:"tokens"`
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

func WithAdvisorCost(executor *decimal.Decimal, kind Kind, advisor *AdvisorUsage) *decimal.Decimal {
	if executor == nil || !advisor.Used() {
		return executor
	}
	advice := advisor.Cost(kind)
	if advice == nil {
		return nil
	}
	total := executor.Add(*advice)

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
